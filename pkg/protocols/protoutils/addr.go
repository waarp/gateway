package protoutils

import (
	"net/netip"
)

func GetIP(addr string) string {
	ip, err := netip.ParseAddrPort(addr)
	if err != nil {
		return addr
	}

	return ip.Addr().String()
}
