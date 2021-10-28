package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestServiceStart(t *testing.T) {
	Convey("Given an HTTP service", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_http_start")
		db := database.TestDatabase(c)
		server := &model.LocalAgent{
			Name:        "http_server",
			Protocol:    "http",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:0",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		serv := NewService(db, logger)

		Convey("When calling the 'Start' function", func() {
			err := serv.Start(server)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestServiceStop(t *testing.T) {
	Convey("Given a running HTTP service", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_http_stop")
		db := database.TestDatabase(c)
		server := &model.LocalAgent{
			Name:        "http_server",
			Protocol:    "http",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:0",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		serv := NewService(db, logger)
		So(serv.Start(server), ShouldBeNil)

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
		logger := testhelpers.TestLogger(c, "http_server")

		serv := newService(test.DB, logger)
		c.So(serv.Start(test.Server), ShouldBeNil)

		Convey("Given a dummy HTTP client", func() {
			cli := http.DefaultClient

			Convey("Given that a push transfer started", func() {
				body := testhelpers.NewLimitedReader(3)

				url := fmt.Sprintf("http://%s/test_in_shutdown.dst?%s=%s",
					test.Server.Address, httpconst.Rule, test.ServerRule.Name)
				req, err := http.NewRequest(http.MethodPost, url, body)
				So(err, ShouldBeNil)
				req.SetBasicAuth(pipelinetest.TestLogin, pipelinetest.TestPassword)
				req.Close = true
				req.ContentLength = 1000

				stop := make(chan error, 1)

				Convey("When the server shuts down", func() {
					defer pipeline.Tester.WaitServerDone()

					go func() {
						pipeline.Tester.WaitServerPreTasks()

						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()

						stop <- serv.Stop(ctx)
					}()

					resp, err := cli.Do(req)
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

						var transfers model.Transfers
						So(test.DB.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)
						So(transfers[0].Status, ShouldEqual, types.StatusInterrupted)

						ok := serv.running.Exists(transfers[0].ID)
						So(ok, ShouldBeFalse)
					})
				})
			})
		})
	})
}
