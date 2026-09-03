package types

import (
	"database/sql/driver"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"
)

const ipAddrSeparator = " | "

//nolint:recvcheck //a mix of pointer and non-pointer receivers is needed here to avoid nil pointers
type IPList []string

func (l IPList) Value() (driver.Value, error) {
	return l.String(), nil
}

func (l *IPList) Scan(src any) error {
	switch val := src.(type) {
	case string:
		return l.fromString(val)
	case []byte:
		return l.fromString(string(val))
	default:
		//nolint:err113 //too specific to have a base error
		return fmt.Errorf("unsupported IP address type %T", src)
	}
}

func (l *IPList) fromString(str string) error {
	if str == "" {
		return nil
	}

	addresses := strings.Split(str, ipAddrSeparator)
	l.Add(addresses...)

	return nil
}

func (l IPList) String() string     { return strings.Join(l, ipAddrSeparator) }
func (l *IPList) Add(ips ...string) { *l = append(*l, ips...) }
func (l *IPList) Contains(addr string) bool {
	ip, err := netip.ParseAddrPort(addr)
	if err != nil {
		return false
	}

	return slices.Contains(*l, ip.Addr().String())
}

func (l *IPList) Validate() error {
	for _, ip := range *l {
		if _, err := net.ResolveIPAddr("ip", ip); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}
