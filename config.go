package main

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"regexp"
)

type Config struct {
	Sources []Source `yaml:"sources",flow`

}

type Syslog struct {
	Facility string `yaml:"facility"`
	Tag string `yaml:"tag"`
}

type Source struct {
	Name string `yaml:"name"`
	Set NftSet `yaml:"nftables_set"`
	LogFile  string   `yaml:"logfile"`
	Patterns []string `yaml:"patterns"`
	Regexps  []*regexp.Regexp
	Debug    bool `yaml:"debug"`
	Syslog Syslog `yaml:"syslog"`
	Events   chan uint32
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

	for _, source := range config.Sources {
		if len(source.Set.Name) > 0 {
			err = source.Set.Check()
			if err != nil {
				return
			}
		}
		_, err := os.Stat(source.LogFile)
		if err != nil {
			break
		}

		var regexps []*regexp.Regexp
		for _, pattern := range source.Patterns {
			r, err := regexp.Compile(pattern)
			if err != nil {
				check(err)
				continue
			}
			regexps = append(regexps, r)
		}
		source.Regexps = regexps
		sources = append(sources, source)
	}
	return
}
