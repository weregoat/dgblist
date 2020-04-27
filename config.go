package main

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

// Config is the general structure of the configuration file (a list of sources)
type Config struct {
	Sources []SourceConfig `yaml:"sources",flow`
}

// Syslog configuration for syslog.
type Syslog struct {
	Facility string `yaml:"facility"`
	Tag      string `yaml:"tag"`
	LogLevel string `yaml:"level"`
}

// SourceConfig configuration entry for source.
type SourceConfig struct {
	sync.Mutex
	Name     string   `yaml:"name"`
	Set      NftSet   `yaml:"nftables_set"`
	LogFile  string   `yaml:"logfile"`
	Patterns []string `yaml:"patterns"`
	Syslog   Syslog   `yaml:"syslog"`
}

// parse reads the configuration file and returns a list of sources to watch.
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
		} else {
			sources = append(sources, source)
		}
	}
	return
}
