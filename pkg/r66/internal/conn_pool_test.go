package internal

import (
	"fmt"
	"net"
	"testing"
	"time"

	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func testR66Server() string {
	authHandler := testAuthHandler(func(*r66.Authent) (r66.SessionHandler, error) {
		return testSessionHandler(func(*r66.Request) (r66.TransferHandler, error) {
			return nil, fmt.Errorf("should not happen") //nolint:goerr113 //base test error
		}), nil
	})

	serv := r66.Server{
		Login:    "foobar",
		Password: []byte("sesame"),
		Conf:     &r66.Config{},
		Handler:  authHandler,
	}

	list, err := net.Listen("tcp", "localhost:0")
	So(err, ShouldBeNil)

	Reset(func() { list.Close() })

	go serv.Serve(list)

	return list.Addr().String()
}

func TestConnPool(t *testing.T) {
	Convey("Given an R66 server", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_r66_conn_pool")
		addr := testR66Server()
		pool := NewConnPool()

		pool.connGracePeriod = time.Millisecond

		Convey("When opening multiple connections to that server", func() {
			conn1, err := pool.Add(addr, nil, logger)
			So(err, ShouldBeNil)

			conn2, err := pool.Add(addr, nil, logger)
			So(err, ShouldBeNil)

			Convey("Then it should have used the same connection", func() {
				So(conn1, ShouldEqual, conn2)
			})

			Convey("When closing the connections", func() {
				So(pool.Exists(addr), ShouldBeTrue)

				pool.Done(addr)
				So(pool.Exists(addr), ShouldBeTrue)

				pool.Done(addr)
				time.Sleep(pool.connGracePeriod * 100)

				Convey("Then it should have closed the connection", func() {
					So(pool.Exists(addr), ShouldBeFalse)
				})
			})
		})
	})
}
