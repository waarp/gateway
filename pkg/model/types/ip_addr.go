package types

import (
	"fmt"
	"net"
	"slices"
	"strings"
)

const ipAddrSeparator = " | "

//nolint:recvcheck //a mix of pointer and non-pointer receivers is needed here to avoid nil pointers
type IPList []string

func (l *IPList) FromDB(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}

	addresses := strings.Split(string(bytes), ipAddrSeparator)
	l.Add(addresses...)

	return nil
}

func (l *IPList) ToDB() ([]byte, error) { return []byte(l.String()), nil }
func (l *IPList) String() string        { return strings.Join(*l, ipAddrSeparator) }
func (l *IPList) Add(ips ...string)     { *l = append(*l, ips...) }
func (l *IPList) Contains(ip string) bool {
	return slices.Contains(*l, ip)
}

func (l *IPList) Validate() error {
	for _, ip := range *l {
		if _, err := net.ResolveIPAddr("ip", ip); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}
