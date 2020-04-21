package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"regexp"
)

type Config struct {
	Set     NftSet   `yaml:"nftables_set"`
	Sources []Source `yaml:"sources",flow`
}

type Source struct {
	Name string `yaml:"name"`
	//	Set NftSet `yaml:"nftables_set",omitempty`
	LogFile  string   `yaml:"logfile"`
	Patterns []string `yaml:"patterns"`
	Regexps  []*regexp.Regexp
	Events   chan uint32
}

func parse(filename string) (config Config, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return
	}

	if len(config.Set.Name) > 0 {
		err = config.Set.Check()
		if err != nil {
			return
		}
	}

	var sources []Source
	for _, source := range config.Sources {
		/*
			if len(source.Set.Name) == 0 {
				source.Set = config.Set
			}
			err = source.Set.Check()

			if err != nil {
				break
			}
		*/
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
	config.Sources = sources
	return
}
