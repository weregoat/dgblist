package main

import (
	"fmt"
	"gitlab.com/weregoat/nftables"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const IPV4 = "ipv4"
const IPV6 = "ipv6"

type NftSet struct {
	Table   string `yaml:"table"`
	Name    string `yaml:"name"`
	Timeout string `yaml:"timeout"`
	Type    string `yaml:"type"`
}

func (s NftSet) Check() error {
	_, err := s.Get()
	return err
}

func (s NftSet) Delete(address string) error {
	return nil
}

func (s NftSet) Add(addresses ...string) ([]net.IP, error) {
	var added []net.IP
	set, err := s.Get()
	if err != nil {
		return added, err
	}
	c := nftables.Conn{}
	for _, address := range addresses {
		ip := net.ParseIP(address)
		switch strings.ToLower(s.Type) {
		case IPV6:
			ip = ip.To16()
		default:
			ip = ip.To4()
		}
		if ip != nil {
			elements := make([]nftables.SetElement, 1)
			element := nftables.SetElement{
				Key: ip,
			}
			elements[0] = element
			err = c.SetAddElements(set, elements)
			if err == nil {
				added = append(added, ip)
			}
		}
	}
	if err != nil {
		return added, err
	}
	err = c.Flush()
	if err != nil {
		added = nil
	}
	return added, err
}

func (s NftSet) Get() (set *nftables.Set, err error) {
	var table *nftables.Table

	c := &nftables.Conn{}

	tables, err := c.ListTables()
	if err != nil {
		return
	}

	for _, t := range tables {
		if t.Name == s.Table {
			table = t
			break
		}
	}

	if table == nil {
		err = fmt.Errorf("no table with name %s", s.Table)
		return
	}

	set, err = c.GetSetByName(table, s.Name)
	if err != nil {
		kType := nftables.TypeIPAddr
		if strings.EqualFold(s.Type, IPV6) {
			kType = nftables.TypeIP6Addr
		}
		set = &nftables.Set{
			Name:       s.Name,
			Table:      table,
			KeyType:    kType,
			HasTimeout: true,
			Timeout:    parseTimeout(s.Timeout),
		}
		err = c.AddSet(set, []nftables.SetElement{})
		if err != nil {
			return
		}
		err = c.Flush()
	}
	return
}

func parseTimeout(timeout string) time.Duration {
	patterns := map[string]int64{
		"([0-9]+)d": (time.Hour * 24).Nanoseconds(),
		"([0-9]+)h": (time.Hour).Nanoseconds(),
		"([0-9]+)m": (time.Minute).Nanoseconds(),
		"([0-9]+)s": (time.Second).Nanoseconds(),
	}
	var t int64 = 0
	for pattern, lenght := range patterns {
		r := regexp.MustCompile(pattern)
		m := r.FindStringSubmatch(timeout)
		if len(m) > 0 {
			for i := 1; i < len(m); i++ {
				d, err := strconv.Atoi(m[i])
				if err == nil {
					t += lenght * int64(d)
				}
			}
		}
	}
	return time.Duration(t)
}
