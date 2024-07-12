package utils

import (
	"net"
	"strconv"
)

//nolint:wrapcheck //wrapping errors adds nothing here
func SplitHostPort(hostport string) (string, uint16, error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return "", 0, err
	}

	portnum, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return "", 0, err
	}

	return host, uint16(portnum), nil
}
