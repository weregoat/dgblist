package main

import (
	"net"
	"sync"
)

// Blacklist is a simple structure for handling list of blacklisted IP addresses.
type Blacklist struct {
	sync.Mutex
	addresses []net.IP
}

// Add adds the given IP addresses to the list of addresses if not nil or
// duplicates.
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

// Addresses returns the list of IP addresses in the blacklist.
func (b *Blacklist) Addresses() []net.IP {
	b.Lock()
	defer b.Unlock()
	return b.addresses
}

// contains return true if the given IP address is already present in the
// list of blacklisted IP addresses.
func (b *Blacklist) contains(ip net.IP) bool {
	for _, present := range b.addresses {
		if present.Equal(ip) {
			return true
		}
	}
	return false
}
