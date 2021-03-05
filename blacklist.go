package main

import (
	"net"
	"sync"
)

type Blacklist struct {
	sync.Mutex
	addresses []net.IP
}

func (b *Blacklist) Add(addresses ...net.IP) {
	b.Lock()
	defer b.Unlock()
	for _, address := range addresses {
		if address == nil {
			continue
		}
		if !contains(b.addresses, address) {
			b.addresses = append(b.addresses, address)
		}
	}
}

func (b *Blacklist) Addresses() []net.IP {
	b.Lock()
	defer b.Unlock()
	return b.addresses
}

func (b *Blacklist) contains(ip net.IP) bool {
	for _, present := range b.addresses {
		if present.Equal(ip) {
			return true
		}
	}
	return false
}
