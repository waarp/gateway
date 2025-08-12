package utils

import (
	"net"
	"strconv"

	"golang.org/x/exp/constraints"
)

//nolint:wrapcheck //wrapping errors adds nothing here
func SplitHostPort[T constraints.Integer](hostport string) (string, T, error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", 0, err
	}

	portnum, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return "", 0, err
	}

	return host, T(portnum), nil
}
