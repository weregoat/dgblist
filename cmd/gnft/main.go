package main

import (
	"bytes"
	"flag"
	"fmt"
	"gitlab.com/weregoat/nftables"
	"log"
	"net"
	"strings"
	"time"
)

func main() {
	tableName := flag.String("table", "filter", "The name of the table.")
	chainName := flag.String("chain", "input", "The name of the chain.")
	setName := flag.String("set", "", "The set name.")
	add := flag.String("add", "", "Add the IP to the named set.")
	rm := flag.String("rm", "", "Remove the IP from the named set.")
	flag.Parse()
	if len(*setName) == 0 {
		log.Fatal("No named set specified")
	}

	c := nftables.Conn{}

	var table *nftables.Table
	var chain *nftables.Chain
	var set *nftables.Set

	chains, err := c.ListChains()
	check(err)

	for _, c := range chains {
		if strings.EqualFold(c.Name, *chainName) {
			chain = c
			break
		}
	}

	if chain == nil {
		log.Fatalf("no chain with name %s", *chainName)
	}

	tables, err := c.ListTables()
	check(err)

	for _, t := range tables {
		if t.Name == *tableName {
			table = t
			break
		}
	}

	if table == nil {
		log.Fatalf("no table with name %s", *tableName)
	}

	set, err = c.GetSetByName(table, *setName)
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

	if len(*add) > 0 {
		ip := net.ParseIP(*add).To4()
		if ip != nil {
			a := nftables.SetElement{
				Key:         ip,
				Val:         []byte{},
				IntervalEnd: false,
				VerdictData: nil,
				Timeout:     60 * time.Hour,
			}
			check(c.SetAddElements(set, []nftables.SetElement{a}))
			check(c.Flush())
		}
	}

	if len(*rm) > 0 {
		var remove []nftables.SetElement
		ip := net.ParseIP(*rm).To4()
		elements, err := c.GetSetElements(set)
		check(err)
		for _, e := range elements {
			if bytes.Equal(ip, e.Key) {
				remove = append(remove, e)
			}
		}
		check(c.SetDeleteElements(set, remove))
		check(c.Flush())
	}

	elements, err := c.GetSetElements(set)
	check(err)
	for _, e := range elements {
		var ip net.IP = e.Key
		fmt.Printf("%s\n", ip)
	}

}

func check(err error) {
	if err != nil {
		log.Panic(err)
	}
}
