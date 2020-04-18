package main

import (
	"bufio"
	"flag"
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
	f, err := os.Open(*logfile)
	if err != nil {
		log.Fatal()
	}
	pos := read(0, f)
	f.Close()


	fd := watch(*logfile)
	events := make(chan int)
	var buf = make([]byte, unix.SizeofInotifyEvent+unix.NAME_MAX+1)

	go func() {
		for {
			_, err := unix.Read(fd, buf)
			if err != nil {
				log.Fatal(err)
			}
			event := *(*unix.InotifyEvent)(unsafe.Pointer(&buf[0]))
			switch event.Mask {
			case unix.IN_ATTRIB:
				log.Printf("received IN_ATTRIB(%d) event", event.Mask)
			case unix.IN_DELETE:
				log.Printf("received IN_DELETE(%d) event", event.Mask)
			case unix.IN_DELETE_SELF:
				log.Printf("received IN_DELETE_SELF(%d) event", event.Mask)
			}
			if event.Mask != unix.IN_MODIFY {
				time.Sleep(5*time.Second)
				fd = watch(*logfile)
			}
			events <- fd
		}
	}()


	currentFD := 0
	for {
		var file *os.File
		fd := <-events
		// Fd has changed, try to reload file
		if fd != currentFD {
			file, err = os.Open(*logfile)
			if err != nil {
				log.Print(err)
				break
			}
		}
		info, err := file.Stat()
		if err != nil {
			log.Print(err)
			break
		}
		if info.Size() < pos {
			pos = 0
		}
		pos += read(pos, file)
	}
}

func watch(file string) int {
	fd, err := unix.InotifyInit()
	if err != nil {
		log.Fatal(err)
	}
	_, err = unix.InotifyAddWatch(fd, file, unix.IN_MODIFY|unix.IN_ATTRIB|unix.IN_DELETE_SELF|unix.IN_DELETE)
	if err != nil {
		log.Fatal(err)
	}
	return fd
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

