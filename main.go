package main

import (
	"bufio"
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"time"
	"unsafe"
)

func main() {
	fmt.Printf("started\n")

	events := make(chan unix.InotifyEvent)
	var buf = make([]byte, unix.SizeofInotifyEvent+unix.NAME_MAX+1)
	fd, err := Watch("/tmp/logfile.log")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			n, err := unix.Read(fd, buf)
			if err != nil {
				log.Fatal(err)
			}
			event := *(*unix.InotifyEvent)(unsafe.Pointer(&buf[0]))
			fmt.Printf("%v\n", event)
			events <- event
			if event.Mask == unix.IN_ATTRIB {
				/*
				_, err = unix.InotifyRmWatch(fd, wd)
				if err != nil {
					log.Fatal(err)
				}
				*/
				time.Sleep(30*time.Second)
				fd, err = Watch("/tmp/logfile.log")
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}()
	file, err := os.Open("/tmp/logfile.log")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	pos := int64(0)
	for {

		select {
			case data := <- events:
				switch data.Mask {
				case unix.IN_MODIFY:
					fmt.Println("IN_MODIFY")
				case unix.IN_ACCESS:
					fmt.Println("IN_ACCESS")
				case unix.IN_ATTRIB:
					fmt.Println("IN_ATTRIB")
				case unix.IN_CLOSE_WRITE:
					fmt.Println("IN_CLOSE_WRITE")
				case unix.IN_CLOSE_NOWRITE:
					fmt.Println("IN_CLOSE_NO_WRITE")
				case unix.IN_CREATE:
					fmt.Println("IN_CREATE")
				case unix.IN_DELETE:
					fmt.Println("IN_DELETE")
				case unix.IN_DELETE_SELF:
					fmt.Println("IN_DELETE_SELF")
				case unix.IN_MOVE_SELF:
					fmt.Println("IN_MOVE_SELF")
				case unix.IN_MOVED_FROM:
					fmt.Println("IN_MOVED_FROM")
				case unix.IN_MOVED_TO:
					fmt.Println("IN_MOVED_TO")
				case unix.IN_OPEN:
					fmt.Println("IN_OPEN")
					file.Seek(pos, 0)
					for {
						line, err := reader.ReadBytes('\n')
						pos += int64(len(line))
						fmt.Print(string(line))
						if err != nil {
								break
						}
					}
				default:
					fmt.Printf("Unhandled: %d", data.Mask)
				}


		}

	}
}

func Watch(file string) (int, error) {
	fd, err := unix.InotifyInit()
	if err != nil {
		log.Fatal(err)
	}

	_, err = unix.InotifyAddWatch(fd, file, unix.IN_ALL_EVENTS)
	return fd, err
}