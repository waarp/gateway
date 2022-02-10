package testhelpers

import (
	"fmt"
	"net"

	"github.com/smartystreets/goconvey/convey"
)

// GetFreePort returns the number of a random unused tcp port.
func GetFreePort(c convey.C) uint16 {
	l, err := net.Listen("tcp", "localhost:0")
	c.So(err, convey.ShouldBeNil)

	//nolint:forcetypeassert //no need, the type assertion will always succeed
	port := uint16(l.Addr().(*net.TCPAddr).Port)
	c.So(l.Close(), convey.ShouldBeNil)

	return port
}

// GetLocalAddress returns a local address ready to be used for a test server.
func GetLocalAddress(c convey.C) string {
	return fmt.Sprintf("localhost:%d", GetFreePort(c))
}
