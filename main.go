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

func main() {
	var fileConfig string
	flag.StringVar(&fileConfig, "config", "", "Configuration file")
	flag.Parse()
	if len(fileConfig) == 0 {
		filename := path.Base(os.Args[0]) + ".yaml"
		fileConfig = path.Join("/etc/", filename)
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
	defer source.Close()
	go stats(&source)
	source.Watch()
	wg.Done()
}

func getKeys(list map[string]string) []string {
	i := 0
	keys := make([]string, len(list))
	for ip := range list {
		keys[i] = ip
		i++
	}
	return keys
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
