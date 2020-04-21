package main

import (
	"bufio"
	"flag"
	"fmt"
	"gitlab.com/weregoat/nftables"
	"golang.org/x/sys/unix"
	"io"
	"log"
	"net"
	"os"
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
	ch := make(chan []net.IP)
	config, err := parse(*fileConfig)
	checkFatal(err)
	for _, source := range config.Sources {
		go watch(source, ch)
	}
	conn := &nftables.Conn{}
	tables, err := conn.ListTables()
	checkFatal(err)

	var set *nftables.Set
	for _, t := range tables {
		if t.Name == config.Set.Table {
			set, err = conn.GetSetByName(t, config.Set.Name)
			checkFatal(err)
		}
	}

	if set == nil {
		log.Fatalf("Could not find set %s in table %s", config.Set.Name, config.Set.Table)
	}
	for {
		blacklist := <-ch
		var elements []nftables.SetElement
		for _, ip := range blacklist {
			element := nftables.SetElement{
				Key: ip.To4(),
			}
			elements = append(elements, element)
		}
		err = conn.SetAddElements(set, elements)
		if !check(err) {
			if !check(conn.Flush()) {
				for _, e := range elements {
					ip := net.IPv4(e.Key[0], e.Key[1], e.Key[2], e.Key[3])
					log.Printf(
						"added %s to @%s",
						ip.String(),
						config.Set.Name,
					)
				}
			}
		}
	}

}

func watch(source Source, ch chan []net.IP) {
	log.Printf("%s started", source.Name)
	// Read file on start
	file, err := os.Open(source.LogFile)
	checkFatal(err)
	defer file.Close()
	fileInfo, err := file.Stat()
	checkFatal(err)
	pos, blacklist := read(0, file, source)
	ch <- blacklist
	// checkFatal(source.Set.Add(blacklist...))

	fd, err := unix.InotifyInit()
	checkFatal(err)
	wd := addWatch(fd, source.LogFile)
	events := make(chan uint32)
	var buf = make([]byte, unix.SizeofInotifyEvent+unix.NAME_MAX+1)

	go func() {
		for {
			_, err := unix.Read(fd, buf)
			checkFatal(err)
			event := *(*unix.InotifyEvent)(unsafe.Pointer(&buf[0]))
			events <- event.Mask
		}
	}()

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
		checkFatal(err)
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
				log.Printf("event %s resulted in new file", desc)
				file = f
				fileInfo = fi
				pos = 0
				unix.InotifyRmWatch(fd, uint32(wd))
				wd = addWatch(fd, source.LogFile)
			}
		}
		// truncated or overwritten
		if fileInfo.Size() < pos {
			log.Printf(
				"event %s caused the size of file %s to change to %d",
				desc,
				fileInfo.Name(),
				fileInfo.Size(),
			)
			pos = 0
		}
		bytesRead, blacklist := read(pos, file, source)
		pos += bytesRead
		ch <- blacklist
	}
}

func addWatch(fd int, file string) int {
	// I need to restrict the INotify events because they may come in sequence and it's difficult to log only the
	// ones that actually trigger the behaviours I am looking after, without logging most of them (and then sort
	// out the relevant ones manually).
	wd, err := unix.InotifyAddWatch(fd, file, unix.IN_MODIFY|unix.IN_MOVE_SELF|unix.IN_DELETE_SELF|unix.IN_ATTRIB)
	checkFatal(err)
	return wd
}

func read(pos int64, file *os.File, source Source) (int64, []net.IP) {
	var bytesRead int64 = 0
	reader := bufio.NewReader(file)
	file.Seek(pos, 0)
	var blacklist []net.IP
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
							blacklist = append(blacklist, ip)
						} else {
							log.Printf("invalid IPv4 (%s) at matching line %+q\n", m[1], m[0])
						}
					}
				}
			}
		}
		/*
			now := time.Now()
			if now.After(mark.Add(interval)) {
				log.Print(string(line))
				mark = now
			}

		*/
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
