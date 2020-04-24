package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
)

func main() {
	fileConfig := flag.String("config", "", "Configuration file")
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

func watch(source Source, wg *sync.WaitGroup) {
	source.Info(
		fmt.Sprintf("starting %s watch", source.Name),
		)
	defer source.Close()
	source.Watch()
	wg.Done()
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