package main

import (
	"bufio"
	"flag"
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"time"
	"unsafe"
)

var mark time.Time
var interval time.Duration

func main() {
	mark = time.Now()
	defer log.Print("glogg terminated")
	logfile := flag.String("logfile", "", "Logfile to watch")
	flag.DurationVar(&interval, "interval", time.Hour*2, "polling interval")
	flag.Parse()
	if len(*logfile) == 0 {
		log.Fatal("No file to watch")
	}

	log.Printf("glogg started on %s", *logfile)
	// Read file on start
	file, err := os.Open(*logfile)
	if err != nil {
		log.Fatal()
	}
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	pos := read(0, file)
	defer file.Close()

	fd, err := unix.InotifyInit()
	if err != nil {
		log.Fatal(err)
	}
	wd := watch(fd, *logfile)
	events := make(chan uint32)
	var buf = make([]byte, unix.SizeofInotifyEvent+unix.NAME_MAX+1)

	go func() {
		for {
			_, err := unix.Read(fd, buf)
			if err != nil {
				log.Fatal(err)
			}
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
		if err != nil {
			log.Fatal(err)
		}
		// Any event that is not modify can lead to a new file, I
		// don't know yet which events are relevant. When I do I will
		// check filter them in (instead of using IN_ALL_EVENTS)
		if event != unix.IN_MODIFY {
			time.Sleep(1 * time.Second)
			f, err := os.Open(*logfile)
			if err != nil {
				log.Print(err)
				break
			}
			fi, err := f.Stat()
			if err != nil {
				log.Print(err)
				break
			}
			// Deleted or moved
			if ! os.SameFile(fi, fileInfo) {
				log.Printf("event %s resulted in new file", desc)
				file = f
				fileInfo = fi
				pos = 0
				unix.InotifyRmWatch(fd, uint32(wd))
				wd = watch(fd, *logfile)
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
		pos += read(pos, file)
	}
}

func watch(fd int, file string) int {
	// I need to restrict the INotify events because they may come in sequence and it's difficult to log only the
	// ones that actually trigger the behaviours I am looking after, without logging most of them (and then sort
	// out the relevant ones manually).
	wd, err := unix.InotifyAddWatch(fd, file, unix.IN_MODIFY|unix.IN_MOVE_SELF|unix.IN_DELETE_SELF|unix.IN_ATTRIB)
	if err != nil {
		log.Fatal(err)
	}
	return wd
}

func read(pos int64, file *os.File) int64 {
	var bytesRead int64 = 0
	reader := bufio.NewReader(file)
	file.Seek(pos, 0)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}
		bytesRead += int64(len(line))
		now := time.Now()
		if now.After(mark.Add(interval)) {
			log.Print(string(line))
			mark = now
		}
	}
	return bytesRead
}
