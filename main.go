package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"time"
)

var dirs = []string{
	"/etc/",
	"/usr/local/etc/",
}

func main() {
	var fileConfig string
	flag.StringVar(&fileConfig, "config", "", "Configuration file")
	flag.Parse()
	if len(fileConfig) == 0 {
		filename := path.Base(os.Args[0]) + ".yaml"
		for _, dir := range dirs {
			fileConfig = path.Join(dir, filename)
			_, err := os.Stat(fileConfig)
			if err == nil {
				break
			}
		}
	}

	sources, err := parse(fileConfig)
	if err != nil {
		log.Fatal(err)
	}
	if len(sources) == 0 {
		log.Fatal("No valid sources to watch")
	}
	var wg sync.WaitGroup
	for _, source := range sources {
		wg.Add(1)
		go watch(source, &wg)
	}
	wg.Wait()

}

func watch(source Source, wg *sync.WaitGroup) {
	source.Info(
		fmt.Sprintf("starting %s watch", source.Name),
	)
	go stats(&source)
	source.Watch()
	wg.Done()
}

func stats(s *Source) {
	if s.Stats.Interval <= 0 {
		return
	}
	for {
		time.Sleep(s.Stats.Interval)
		s.LogStats()
	}
}
