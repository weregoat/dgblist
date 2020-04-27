package main

import (
	"fmt"
	"gitlab.com/weregoat/nftables"
	"net"
	"strings"
)

const IPV4 = "ipv4"
const IPV6 = "ipv6"

// NftSet is a struct defining some of the properties of a nftables set.
type NftSet struct {
	Table string `yaml:"table"`
	Name  string `yaml:"name"`
	Type  string `yaml:"type"`
}

// Check controls that a nftables exists or generate ones, if not.
func (s NftSet) Check() error {
	_, err := s.Get()
	return err
}

// Add adds the given address to the set.
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

// Get returns a pointer to the set.
// One is created if doesn't exist already.
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

	return c.GetSetByName(table, s.Name)
}
