package testhelpers

import (
	"net"

	"github.com/smartystreets/goconvey/convey"
)

// GetFreePort returns the number of a random unused tcp port.
func GetFreePort(c convey.C) uint16 {
	l, err := net.Listen("tcp", "localhost:0")
	c.So(err, convey.ShouldBeNil)

	port := uint16(l.Addr().(*net.TCPAddr).Port)
	c.So(l.Close(), convey.ShouldBeNil)

	return port
}
