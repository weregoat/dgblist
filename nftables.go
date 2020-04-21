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
	_, err := getSet(s.Table, s.Name)
	return err
}

func (s NftSet) Delete(address string) error {
	return nil
}

func (s NftSet) Add(ips ...net.IP) error {
	set, err := getSet(s.Table, s.Name)
	if check(err) {
		return err
	}
	c := nftables.Conn{}
	var elements []nftables.SetElement
	for _, ip := range ips {
		element := nftables.SetElement{
			Key: ip.To4(),
		}
		elements = append(elements, element)
	}
	err = c.SetAddElements(set, elements)
	if err == nil {
		err = c.Flush()
	}
	return err
}

func getSet(tableName, SetName string) (set *nftables.Set, err error) {
	var (
		//chain *nftables.Chain
		table *nftables.Table
	)

	c := nftables.Conn{}
	/*
		chains, err := c.ListChains()
		if err != nil {
			return
		}

		for _, c := range chains {
			if strings.EqualFold(c.Name, chainName) {
				chain = c
				break
			}
		}

		if chain == nil {
			err = fmt.Errorf("no chain with name %s", chainName)
			return
		}

	*/

	tables, err := c.ListTables()
	if err != nil {
		return
	}

	for _, t := range tables {
		if t.Name == tableName {
			table = t
			break
		}
	}

	if table == nil {
		err = fmt.Errorf("no table with name %s", tableName)
		return
	}

	set, err = c.GetSetByName(table, SetName)
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
