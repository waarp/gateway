package http

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func TestServiceStart(t *testing.T) {
	logger := log.NewLogger("test_http_start")

	Convey("Given an HTTP service", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		server := &model.LocalAgent{
			Name:        "http_server",
			Protocol:    "http",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:0",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		serv := NewService(db, server, logger)

		Convey("When calling the 'Start' function", func() {
			err := serv.Start()

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestServiceStop(t *testing.T) {
	logger := log.NewLogger("test_http_stop")

	Convey("Given a running HTTP service", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		server := &model.LocalAgent{
			Name:        "http_server",
			Protocol:    "http",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:0",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		serv := NewService(db, server, logger)
		So(serv.Start(), ShouldBeNil)

		Convey("When calling the 'Stop' function", func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err := serv.Stop(ctx)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestServerInterruption(t *testing.T) {
	Convey("Given an SFTP server ready for push transfers", t, func(c C) {
		test := pipelinetest.InitServerPush(c, "http", NewService, nil)

		serv := NewService(test.DB, test.Server, log.NewLogger("server"))
		c.So(serv.Start(), ShouldBeNil)

		Convey("Given a dummy HTTP client", func() {
			cli := http.DefaultTransport

			Convey("Given that a push transfer started", func() {
				body := newLimitedReader(3)

				url := fmt.Sprintf("http://%s/test_in_shutdown.dst?%s=%s",
					test.Server.Address, httpconst.Rule, test.ServerRule.Name)
				req, err := http.NewRequest(http.MethodPost, url, body)
				So(err, ShouldBeNil)
				req.SetBasicAuth(pipelinetest.TestLogin, pipelinetest.TestPassword)

				stop := make(chan error, 1)

				go func() {
					time.Sleep(500 * time.Millisecond)

					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					stop <- serv.Stop(ctx)
				}()

				Convey("When the server shuts down", func() {
					resp, err := cli.RoundTrip(req)
					So(err, ShouldBeNil)

					So(<-stop, ShouldBeNil)

					defer resp.Body.Close()

					So(resp.StatusCode, ShouldEqual, http.StatusServiceUnavailable)
					So(resp.Header.Get(httpconst.TransferStatus), ShouldEqual, types.StatusInterrupted)
					body, err := ioutil.ReadAll(resp.Body)
					So(err, ShouldBeNil)
					So(string(body), ShouldResemble, "transfer interrupted by a server shutdown")

					Convey("Then the transfer should have been interrupted", func(c C) {
						test.ServerShouldHavePreTasked(c)
						test.TasksChecker.WaitServerDone()

						var transfers model.Transfers
						So(test.DB.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)
						So(transfers[0].Status, ShouldEqual, types.StatusInterrupted)

						//nolint:forcetypeassert //no need, the type assertion will always succeed
						ok := serv.(*httpService).running.Exists(transfers[0].ID)
						So(ok, ShouldBeFalse)
					})
				})
			})
		})
	})
}

func newLimitedReader(lim int) *limitedReader {
	return &limitedReader{lim: lim, tick: time.NewTicker(time.Second)}
}

type limitedReader struct {
	lim  int
	tick *time.Ticker
}

func (l *limitedReader) Read(b []byte) (int, error) {
	<-l.tick.C

	return rand.Read(b[:l.lim]) //nolint:wrapcheck //useless here, only used for tests
}
