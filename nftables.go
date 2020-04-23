package main

import (
	"fmt"
	"gitlab.com/weregoat/nftables"
	"net"
)

type NftSet struct {
	Table string `yaml:"table"`
	//	Chain string `yaml:"chain"`
	Name string `yaml:"name"`
}

func (s NftSet) Check() error {
	_, err := s.Get()
	return err
}

func (s NftSet) Delete(address string) error {
	return nil
}

func (s NftSet) Add(addresses ...string) error {
	set, err := s.Get()
	if err != nil {
		return err
	}
	c := nftables.Conn{}
	var elements []nftables.SetElement
	for _,address := range addresses {
		ip := net.ParseIP(address).To4()
		if ip != nil {
			element := nftables.SetElement{
				Key: ip.To4(),
			}
			elements = append(elements, element)
		}
	}
	err = c.SetAddElements(set, elements)
	if err != nil {
		return err
	}
	err = c.Flush()
	return err
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
	return
	/*
		if err != nil {
			set = &nftables.Set{
				Name:       *setName,
				Table:      table,
				KeyType:    nftables.TypeIPAddr,
				HasTimeout: true,
				Timeout:    90 * 24 * time.Hour,
			}
			check(c.AddSet(set, []nftables.SetElement{}))
			check(c.Flush())
		}

	*/
}
