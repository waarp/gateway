package types

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrEmptyAddress = errors.New("address cannot be empty")

type Address struct {
	Host string
	Port uint16
}

// Addr returns a new Address with the given host and port. Note that the
// address is NOT validated.
func Addr(host string, port uint16) Address {
	return Address{Host: host, Port: port}
}

// NewAddress parses the given string as an address and returns it, if it is valid.
func NewAddress(str string) (*Address, error) {
	addr := &Address{}
	if err := addr.Set(str); err != nil {
		return nil, err
	}

	return addr, nil
}

func (a *Address) IsSet() bool               { return a.Host != "" || a.Port != 0 }
func (a *Address) FromDB(bytes []byte) error { return a.Set(string(bytes)) }
func (a *Address) ToDB() ([]byte, error)     { return []byte(a.String()), nil }

func (a *Address) String() string {
	if !a.IsSet() {
		return ""
	}

	return net.JoinHostPort(a.Host, utils.FormatUint(a.Port))
}

func (a *Address) Set(addr string) error {
	if addr == "" {
		return nil
	}

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("failed to parse the server address: %w", err)
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return fmt.Errorf("invalid port number %q in server address: %w", portStr, err)
	}

	a.Host = host
	a.Port = uint16(port)

	return nil
}

func (a *Address) Validate() error {
	if !a.IsSet() {
		return ErrEmptyAddress
	}

	if _, err := net.ResolveTCPAddr("tcp", a.String()); err != nil {
		return fmt.Errorf("invalid address %q: %w", a, err)
	}

	return nil
}

func (a Address) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(a.String())), nil
}
