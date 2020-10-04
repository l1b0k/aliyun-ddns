package types

import "net"

type Addr struct {
	IPv4 net.IP
	IPv6 net.IP
}

func NewAddrFromSlice(addrs []string) *Addr {
	res := &Addr{}
	for _, s := range addrs {
		a := net.ParseIP(s)
		if a == nil {
			continue
		}
		if a.To4() == nil {
			res.IPv6 = a
		} else {
			res.IPv4 = a
		}
	}

	return res
}
