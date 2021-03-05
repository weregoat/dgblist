package main

import (
	"bufio"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"log/syslog"
	"net"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// Source is the struct defining the log source to watch.
type Source struct {
	sync.Mutex
	Name            string
	Set             NftSet
	LogFile         string
	Regexps         []*regexp.Regexp
	Events          chan uint32
	Logger          *syslog.Writer
	Pos             uint64
	FileInfo        os.FileInfo
	FileDescriptor  int
	WatchDescriptor int
	LogLevel        syslog.Priority
	Config          *SourceConfig
	Stats           Stats
	WhiteList       []net.IP
}

// Init initialise the source according to the configuration entry.
func Init(config *SourceConfig) (source *Source, err error) {
	source = &Source{}
	source.LogFile = config.LogFile
	source.Name = config.Name
	source.Lock()
	defer source.Unlock()
	if len(config.Syslog.Facility) == 0 {
		config.Syslog.Facility = DEFAULT_FACILITY
		log.Printf("No syslog facility specified; using %s", config.Syslog.Facility)
	}
	if len(config.Syslog.LogLevel) == 0 {
		config.Syslog.LogLevel = DEFAULT_LEVEL
		log.Printf("No minimum syslog severity specified; using %s", config.Syslog.LogLevel)
	}
	if len(config.Syslog.Tag) == 0 {
		config.Syslog.Tag = path.Base(os.Args[0])
		log.Printf("No syslog tag specified; using %s", config.Syslog.Tag)
	}
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
			return source, fmt.Errorf("invalid nft set @%s: %w", config.Set.Name, err)
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
	if len(regexps) == 0 {
		source.Warning(
			fmt.Sprintf(
				"No valid regular expression defined for source %s",
				source.Name),
		)
	}
	source.Regexps = regexps

	var whitelist []net.IP
	for _, address := range config.Whitelist {
		ip := net.ParseIP(address)
		if ip != nil {
			whitelist = append(whitelist, ip)
		} else {
			source.Warning(
				fmt.Sprintf("Invalid whitelist address %q in source %q", address, source.Name),
			)
		}
	}
	source.WhiteList = whitelist

	source.Config = config
	if len(config.StatsInterval) > 0 {
		interval, err := time.ParseDuration(config.StatsInterval)
		if err != nil {
			source.Warning(err.Error())
		}
		if interval > 0 {
			source.Stats.Interval = interval
		}
	}
	return
}

// Close tried to close the open files the source is using.
func (source *Source) Close() {
	source.Info(
		fmt.Sprintf("ending %s watch", source.Name),
	)
	unix.InotifyRmWatch(source.FileDescriptor, uint32(source.WatchDescriptor))
	source.Logger.Close()
	source.Unlock()
}

// Watch starts watching the source file for matching patterns.
func (source *Source) Watch() {
	defer source.Close()
	var err error
	source.Stats.Started = time.Now()
	// Read file on start
	source.Blacklist(source.read()...)
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
			source.Stats.Events++
			desc := fmt.Sprintf("%d", event)
			switch event {
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
					source.Err(err.Error())
					return
				}
			}
			source.Blacklist(source.read()...)
		}
	}
}

// Blacklist add the IP addresses into the nftables set defined for the source.
func (source *Source) Blacklist(addresses ...net.IP) {
	added, err := source.Set.Add(addresses...)
	if err != nil {
		source.Err(err.Error())
	}
	source.Stats.IPAdded += len(added)
	for _, ip := range added {
		source.Info(
			fmt.Sprintf(
				"added %s to @%s",
				ip.String(), source.Set.Name,
			),
		)
	}
}

// inotifyAddWatch adds a inotify watch for the given source.
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
		unix.IN_MODIFY|unix.IN_MOVE_SELF|unix.IN_DELETE_SELF,
	)
	return err
}

// Refresh re-open a logfile if it has changed.
func (source *Source) Refresh() error {
	source.Lock()
	defer source.Unlock()
	current, err := os.Open(source.LogFile)
	if err != nil {
		return err
	}
	defer current.Close()
	fileInfo, err := current.Stat()
	if err != nil {
		return err
	}
	// Deleted or moved
	if !os.SameFile(fileInfo, source.FileInfo) {
		source.Logger.Info(
			fmt.Sprintf("re-opening %s file", source.LogFile),
		)
		source.FileInfo = fileInfo
		source.Pos = 0
		unix.InotifyRmWatch(source.FileDescriptor, uint32(source.WatchDescriptor))
		err = source.inotifyAddWatch()
		if err != nil {
			return err
		}
	}
	return nil
}

// read looks for new log entries in the file and matches to the regexps.
func (source *Source) read() []net.IP {
	source.Lock()
	defer source.Unlock()
	blacklist := Blacklist{}
	file, err := os.Open(source.LogFile)
	if err != nil {
		source.Err(err.Error())
		return blacklist.Addresses()
	}
	defer file.Close()
	source.FileInfo, err = file.Stat()
	if err != nil {
		source.Err(err.Error())
		return blacklist.Addresses()
	}
	if source.FileInfo.Size() < int64(source.Pos) {
		source.Info(
			fmt.Sprintf(
				"file %s size changed to %d",
				source.FileInfo.Name(),
				source.FileInfo.Size(),
			),
		)
		source.Pos = 0
	}
	var bytesRead uint64 = 0
	file.Seek(int64(source.Pos), 0)
	reader := bufio.NewReader(file)
	for {
		var line []byte
		line, err = reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				source.Err(err.Error())
			}
			break
		}
		source.Stats.LinesRead++
		bytesRead += uint64(len(line))
		for _, r := range source.Regexps {
			addresses := source.parse(string(line), r)
			if err != nil {
				source.Warning(err.Error())
				continue
			}
			blacklist.Add(addresses...)
		}
	}
	source.Pos += bytesRead
	source.Stats.BytesRead += bytesRead

	return blacklist.Addresses()
}

// parse extracts the IP addresses from a given regexp from all the submatch and of the same type
// as the nft set.
func (source *Source) parse(line string, r *regexp.Regexp) []net.IP {
	var addresses []net.IP
	sm := r.FindAllStringSubmatch(line, -1)
	// No match
	if sm == nil {
		return addresses
	}
	// There could be multiple matching
	for _, m := range sm {
		// m[0] is the matched text
		// m[1] would be the first sub-match/capturing group
		// m[2] the second capturing group if present, etc.
		if len(m) < 2 {
			// We don't bother if there are no sub-matches
			continue
		}

		for i := 1; i < len(m); i++ {
			if len(m[i]) == 0 {
				// Empty submatch, no point in trying to parse it.
				continue
			}
			ip := net.ParseIP(m[i])
			if ip == nil {
				source.Warningf(
					"Invalid captured address %q from regexp %s on match %+q",
					m[i], r.String(), m[0],
				)
				continue
			}

			// We want to be sure we will not be feeding IPv6 addresses into a IPv4 nft set.
			// It's a bit complex as all net.IP are 16 bytes, but the quickest way to decide if a net.IP is IPv4
			// is through the net.IP.To4() function.
			if source.Set.Type == IPV4 && ip.To4() == nil {
				source.Warningf(
					"Matched address %s from %q is not a valid IPv4 address", m[i], m[0],
				)
				continue
			}

			// Remove the IP from the matching string to avoid the regexp to match it again if the log is feed to the
			// same log file.
			source.Debugf(
				"Address %s from %+q",
				m[i], strings.Replace(
					m[0], m[i], "{address was here}",
					-1),
			)
			// Try to avoid duplicates
			if contains(addresses, ip) {
				continue
			}

			// Skip whitelisted addresses
			add := true
			for _, whitelisted := range source.WhiteList {
				if ip.Equal(whitelisted) {
					source.Debugf("IP address %s is whitelisted", ip.String())
					add = false
					break
				}
			}
			if add {
				addresses = append(addresses, ip)
			}
		}
	}
	return addresses
}

// contains a simple function to check if an IP is already contained in an existing
// list of IPs.
func contains(list []net.IP, ip net.IP) bool {
	for _, present := range list {
		if ip.Equal(present) {
			return true
		}
	}
	return false
}
