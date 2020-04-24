package main

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

type Config struct {
	Sources []SourceConfig `yaml:"sources",flow`
}

type Syslog struct {
	Facility string `yaml:"facility"`
	Tag string `yaml:"tag"`
	LogLevel string `yaml:"level"`
}

type SourceConfig struct {
	sync.Mutex
	Name     string   `yaml:"name"`
	Set      NftSet   `yaml:"nftables_set"`
	LogFile  string   `yaml:"logfile"`
	Patterns []string `yaml:"patterns"`
	Syslog   Syslog `yaml:"syslog"`
}

func parse(filename string) (sources []Source, err error) {
	config := Config{}
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return
	}

	for _, sourceConfig := range config.Sources {
		source, err := Init(sourceConfig)
		if err != nil {
			log.SetOutput(os.Stderr)
			log.Printf(
				"could not initialize source %s in configuration %s",
				sourceConfig.Name,
				filename,
				)
			log.Print(err)
			continue
		}
		sources = append(sources, source)
	}
	return
}
