package main

import (
	"bufio"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"log/syslog"
	"net"
	"os"
	"regexp"
	"sync"
	"time"
	"unsafe"
)

type Source struct {
	sync.Mutex
	Name            string
	Set             NftSet
	LogFile         string
	Regexps         []*regexp.Regexp
	Events          chan uint32
	Logger          *syslog.Writer
	Pos             int64
	File            *os.File
	FileInfo        os.FileInfo
	FileDescriptor  int
	WatchDescriptor int
	LogLevel        syslog.Priority
	Config          SourceConfig
}

func Init(config SourceConfig) (source Source, err error) {
	source = Source{}
	source.LogFile = config.LogFile
	source.Name = config.Name
	source.Lock()
	defer source.Unlock()
	logger, err := syslog.New(
		facility(config.Syslog.Facility)|severity(config.Syslog.LogLevel),
		config.Syslog.Tag,
	)
	if err != nil {
		return
	}
	source.Logger = logger
	source.LogLevel = severity(config.Syslog.LogLevel)

	if len(config.Set.Name) > 0 {
		err = config.Set.Check()
		if err != nil {
			return
		}
		source.Set = config.Set
	}

	_, err = os.Stat(config.LogFile)
	if err != nil {
		return
	}

	var regexps []*regexp.Regexp
	for _, pattern := range config.Patterns {
		r, err := regexp.Compile(pattern)
		if err != nil {
			source.Warning(
				fmt.Sprintf(
					"failed to compile pattern %s for source %s with error: %s",
					pattern,
					source.Name,
					err.Error(),
				),
			)
		} else {
			regexps = append(regexps, r)
		}
	}
	source.Regexps = regexps
	source.Config = config
	return
}

func (source *Source) Close() {
	source.Info(
		fmt.Sprintf("ending %s watch", source.Name),
	)
	source.File.Close()
	source.Logger.Close()
	source.Unlock()
}

func (source *Source) Watch() {
	// Read file on start
	file, err := os.Open(source.LogFile)
	if err != nil {
		source.Err(err.Error())
		return
	}
	source.File = file
	fileInfo, err := file.Stat()
	if err != nil {
		source.Err(err.Error())
		return
	}
	source.FileInfo = fileInfo
	blacklist := source.read()
	source.addBlacklist(blacklist)

	/*
		inotify_init(2)
		inotify_init() initializes a new inotify instance and returns a file
		descriptor associated with a new inotify event queue.
	*/
	source.FileDescriptor, err = unix.InotifyInit()
	if err != nil {
		source.Err(err.Error())
		return
	}

	err = source.inotifyAddWatch()
	if err != nil {
		source.Err(err.Error())
		return
	}
	events := make(chan uint32)
	errors := make(chan error)

	go func() {
		var buf = make([]byte, unix.SizeofInotifyEvent+unix.NAME_MAX+1)
		for {
			_, err := unix.Read(source.FileDescriptor, buf)
			if err != nil {
				errors <- err
			}
			event := *(*unix.InotifyEvent)(unsafe.Pointer(&buf[0]))
			events <- event.Mask
		}
	}()

	for {
		select {
		case err := <-errors:
			source.Logger.Err(err.Error())
			return
		case event := <-events:
			desc := fmt.Sprintf("%d", event)
			switch event {
			case unix.IN_ATTRIB:
				desc = fmt.Sprintf("IN_ATTRIB(%d)", event)
			case unix.IN_MOVE_SELF:
				desc = fmt.Sprintf("IN_MOVE_SELF(%d)", event)
			case unix.IN_MODIFY:
				desc = fmt.Sprintf("IN_MODIFY(%d)", event)
			case unix.IN_DELETE_SELF:
				desc = fmt.Sprintf("IN_DELETE_SELF(%d)", event)
			}

			// Any event that is not modify can lead to a new file, I
			// don't know yet which events are relevant. When I do I will
			// check filter them in (instead of using IN_ALL_EVENTS)
			if event != unix.IN_MODIFY {
				source.Debug(
					fmt.Sprintf(
						"inotify event %s on file %s", desc, source.LogFile,
					),
				)
				time.Sleep(1 * time.Second)
				err = source.Refresh()
				if err != nil {
					source.Logger.Err(err.Error())
					return
				}
			}
			blacklist := source.read()
			source.addBlacklist(blacklist)
		}
	}
}

func (source *Source) addBlacklist(blacklist map[string]string) {
	added, err := source.Set.Add(getKeys(blacklist)...)
	if err != nil {
		source.Err(err.Error())
	}
	for _, ip := range added {
		source.Info(
			fmt.Sprintf(
				"added %s to set %s",
				ip.String(), source.Set.Name,
			),
		)
	}

}

func (source *Source) inotifyAddWatch() error {
	var err error
	/*
		inotify_add_watch(2)
		inotify_add_watch() adds a new watch, or modifies an existing watch,
		for the file whose location is specified in pathname; [...]
		The fd argument is a file descriptor referring to the inotify instance
		whose watch list is to be modified.
		The events to be monitored for pathname are specified in the mask
		bit-mask argument.
		See inotify(7) for a description of the bits that can be set in mask.
	*/
	source.WatchDescriptor, err = unix.InotifyAddWatch(
		source.FileDescriptor, source.LogFile,
		unix.IN_MODIFY|unix.IN_MOVE_SELF|unix.IN_DELETE_SELF|unix.IN_ATTRIB,
	)
	return err
}

func (source *Source) Refresh() error {
	source.Lock()
	defer source.Unlock()
	f, err := os.Open(source.LogFile)
	if err != nil {
		return err
	}
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	// Deleted or moved
	if !os.SameFile(fi, source.FileInfo) {
		source.Logger.Info(
			fmt.Sprintf("re-opening %s file", source.LogFile),
		)
		source.File = f
		source.FileInfo = fi
		source.Pos = 0
		unix.InotifyRmWatch(source.FileDescriptor, uint32(source.WatchDescriptor))
		err = source.inotifyAddWatch()
		if err != nil {
			return err
		}
	}
	return nil
}

func (source *Source) read() map[string]string {
	var blacklist = make(map[string]string)
	source.Lock()
	if source.FileInfo.Size() < source.Pos {
		source.Logger.Info(
			fmt.Sprintf(
				"file %s size changed to %d",
				source.FileInfo.Name(),
				source.FileInfo.Size(),
			),
		)
		source.Pos = 0
	}
	defer source.Unlock()
	var bytesRead int64 = 0
	source.File.Seek(source.Pos, 0)
	for {
		reader := bufio.NewReader(source.File)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				source.Logger.Err(err.Error())
			}
			break
		}
		bytesRead += int64(len(line))
		for _, r := range source.Regexps {
			sm := r.FindAllStringSubmatch(string(line), -1)
			if len(sm) > 0 {
				for _, m := range sm {
					if len(m) >= 2 {
						ip := net.ParseIP(m[1]).To4()
						if ip != nil {
							blacklist[m[1]] = m[0]
						} else {
							source.Warning(
								fmt.Sprintf(
									"invalid IPv4 (%s) from regexp %+q\n", m[1], r.String(),
									),
							)
						}
					}
				}
			}
		}
	}
	source.Pos += bytesRead
	if source.LogLevel >= syslog.LOG_DEBUG {
		for ip, match := range blacklist {
			text := fmt.Sprintf("address %s matches from %+q", ip, match)
			source.Debug(text)
		}
	}
	return blacklist
}
