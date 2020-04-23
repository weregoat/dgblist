package main

import (
	"bufio"
	"flag"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"log/syslog"
	"net"
	"os"
	"sync"
	"time"
	"unsafe"
)

// var mark time.Time

// var interval time.Duration

func main() {
	// 	mark = time.Now()
	fileConfig := flag.String("config", "", "Configuration file")
	//	flag.DurationVar(&interval, "interval", time.Hour*2, "polling interval")
	flag.Parse()
	if len(*fileConfig) == 0 {
		log.Fatal("No configuration file")
	}

	sources, err := parse(*fileConfig)
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	for _, source := range sources {
		wg.Add(1)
		go watch(source, &wg)
	}
	wg.Wait()

}

/*
func addToSet(source Source, blacklist []net.IP) error {

	var elements []nftables.SetElement
	for _, ip := range blacklist {
		element := nftables.SetElement{
			Key: ip.To4(),
		}
		elements = append(elements, element)
	}

	conn := &nftables.Conn{}
	tables, err := conn.ListTables()
	if check(err) {
		return err
	}

	var set *nftables.Set
	for _, t := range tables {
		if t.Name == source.Set.Table {
			set, err = conn.GetSetByName(t, source.Set.Name)
			if err != nil {
				return err
			}
		}
	}

	if set == nil {
		return fmt.Errorf(
			"no set %s in table %s",
			source.Set.Name, source.Set.Table,
		)
	}

	err = conn.SetAddElements(set, elements)
	if err != nil {
		return err
	}
	err = conn.Flush()
	if err != nil {
		return err
	}
	for _, e := range elements {
		ip := net.IPv4(e.Key[0], e.Key[1], e.Key[2], e.Key[3])
		log.Printf(
			"added %s to @%s from source %s",
			ip.String(),
			source.Set.Name,
			source.Name,
		)
	}
	return nil
}
*/

func watch(source Source, wg *sync.WaitGroup) {
	debug := source.Debug
	defer wg.Done()
	log, err := syslog.New(
		facility(source.Syslog.Facility)|syslog.LOG_INFO,
		source.Syslog.Tag,
	)
	if err != nil {
		return
	}
	// LIFO
	defer log.Close()
	defer log.Info(fmt.Sprintf("%s watch ended", source.Name))

	log.Info(
		fmt.Sprintf("%s watch started", source.Name),
	)
	// Read file on start
	file, err := os.Open(source.LogFile)
	if err != nil {
		log.Err(err.Error())
		return
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		log.Err(err.Error())
		return
	}

	pos, blacklist := read(0, file, source)
	if debug {
		logBlacklist(log, blacklist)
	}
	err = source.Set.Add(getKeys(blacklist)...)
	if err != nil {
		log.Err(err.Error())
	}

	fd, err := unix.InotifyInit()
	if err != nil {
		log.Err(err.Error())
		return
	}

	wd := addWatch(fd, source.LogFile)
	events := make(chan uint32)
	var buf = make([]byte, unix.SizeofInotifyEvent+unix.NAME_MAX+1)

	go func(log *syslog.Writer) {
		for {
			_, err := unix.Read(fd, buf)
			if err != nil {
				log.Err(err.Error())
				break
			}
			event := *(*unix.InotifyEvent)(unsafe.Pointer(&buf[0]))
			events <- event.Mask
		}
	}(log)

	for {
		event := <-events
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
		fileInfo, err = file.Stat()
		if err != nil {
			log.Err(err.Error())
			break
		}

		// Any event that is not modify can lead to a new file, I
		// don't know yet which events are relevant. When I do I will
		// check filter them in (instead of using IN_ALL_EVENTS)
		if event != unix.IN_MODIFY {
			time.Sleep(1 * time.Second)
			f, err := os.Open(source.LogFile)
			if check(err) {
				break
			}
			fi, err := f.Stat()
			if check(err) {
				break
			}
			// Deleted or moved
			if !os.SameFile(fi, fileInfo) {
				log.Info(
					fmt.Sprintf("event %s resulted in new file", desc),
				)
				file = f
				fileInfo = fi
				pos = 0
				unix.InotifyRmWatch(fd, uint32(wd))
				wd = addWatch(fd, source.LogFile)
			}
		}
		// truncated or overwritten with '>' and similar.
		if fileInfo.Size() < pos {
			log.Info(
				fmt.Sprintf(
					"event %s caused the size of file %s to change to %d",
					desc,
					fileInfo.Name(),
					fileInfo.Size(),
				),
			)
			pos = 0
		}
		bytesRead, blacklist := read(pos, file, source)
		pos += bytesRead
		if debug {
			logBlacklist(log, blacklist)
		}
		err = source.Set.Add(getKeys(blacklist)...)
		if err != nil {
			log.Err(err.Error())
		}
	}
}

func addWatch(fd int, file string) int {
	// Logrotate on my system generates IN_MODIFY for `copytruncate` and IN_DELETE_SELF for the default behaviour.
	// IN_MODIFY is also on write.
	// IN_ATTRIB showed up when I deleted the file manually with 'rm'.
	wd, err := unix.InotifyAddWatch(fd, file, unix.IN_MODIFY|unix.IN_MOVE_SELF|unix.IN_DELETE_SELF|unix.IN_ATTRIB)
	checkFatal(err)
	return wd
}

func read(pos int64, file *os.File, source Source) (int64, map[string]string) {
	var bytesRead int64 = 0
	reader := bufio.NewReader(file)
	file.Seek(pos, 0)
	var blacklist = make(map[string]string)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				check(err)
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
							log.Printf("invalid IPv4 (%s) at matching line %+q\n", m[1], m[0])
						}
					}
				}
			}
		}
	}
	return bytesRead, blacklist
}

func check(err error) bool {
	if err != nil {
		log.Print(err)
		return true
	}
	return false
}

func checkFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func facility(keyword string) syslog.Priority {
	switch keyword {
	case "user":
		return syslog.LOG_USER
	case "mail":
		return syslog.LOG_MAIL
	case "daemon":
		return syslog.LOG_DAEMON
	case "auth":
		return syslog.LOG_AUTH
	case "authpriv":
		return syslog.LOG_AUTHPRIV
	case "local0":
		return syslog.LOG_LOCAL0
	case "local1":
		return syslog.LOG_LOCAL1
	case "local2":
		return syslog.LOG_LOCAL2
	case "local3":
		return syslog.LOG_LOCAL3
	case "local4":
		return syslog.LOG_LOCAL4
	case "local5":
		return syslog.LOG_LOCAL5
	case "local6":
		return syslog.LOG_LOCAL6
	case "local7":
		return syslog.LOG_LOCAL7
	}
	return syslog.LOG_LOCAL5
}

func logBlacklist(log *syslog.Writer, blacklist map[string]string) {
	for ip, match := range blacklist {
		text := fmt.Sprintf("address %s matches from %+q", ip, match)
		log.Debug(text)
	}
}

func getKeys(list map[string]string) []string {
	i:= 0
	keys := make([]string, len(list))
	for ip := range list {
		keys[i] = ip
		i++
	}
	return keys
}