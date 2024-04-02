package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestServiceStart(t *testing.T) {
	Convey("Given an HTTP service", t, func(c C) {
		db := database.TestDatabase(c)
		server := &model.LocalAgent{
			Name:     "http_server",
			Protocol: HTTP,
			Address:  "localhost:0",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		serv := httpService{db: db, agent: server}

		Convey("When calling the 'Start' function", func() {
			err := serv.Start()

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestServiceStop(t *testing.T) {
	Convey("Given a running HTTP service", t, func(c C) {
		db := database.TestDatabase(c)
		server := &model.LocalAgent{
			Name:     "http_server",
			Protocol: HTTP,
			Address:  "localhost:0",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		serv := httpService{db: db, agent: server}
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
		test := pipelinetest.InitServerPush(c, HTTP, nil)

		serv := httpService{db: test.DB, agent: test.Server}
		c.So(serv.Start(), ShouldBeNil)

		dataReady := make(chan bool)
		transferDone := make(chan bool)

		serv.SetTracer(func() pipeline.Trace {
			return pipeline.Trace{
				OnDataStart: func() error {
					close(dataReady)

					return nil
				},
				OnTransferEnd: func() {
					close(transferDone)
				},
			}
		})

		Convey("Given a dummy HTTP client", func() {
			cli := http.DefaultClient

			Convey("Given that a push transfer started", func() {
				body := testhelpers.NewSlowReader()

				url := fmt.Sprintf("http://%s/test_in_shutdown.dst?%s=%s",
					test.Server.Address, httpconst.Rule, test.ServerRule.Name)
				req, err := http.NewRequest(http.MethodPost, url, body)
				So(err, ShouldBeNil)
				req.SetBasicAuth(pipelinetest.TestLogin, pipelinetest.TestPassword)
				req.Close = true
				req.ContentLength = 1000

				stop := make(chan error, 1)

				Convey("When the server shuts down", func() {
					defer utils.WaitChan(transferDone, 5*time.Second)

					go func() {
						utils.WaitChan(dataReady, time.Second)

						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()

						stop <- serv.Stop(ctx)
					}()

					resp, err := cli.Do(req)
					So(err, ShouldBeNil)

					So(<-stop, ShouldBeNil)

					defer resp.Body.Close()

					So(resp.StatusCode, ShouldEqual, http.StatusServiceUnavailable)
					So(resp.Header.Get(httpconst.TransferStatus), ShouldEqual,
						string(types.StatusInterrupted))

					body, err := io.ReadAll(resp.Body)
					So(err, ShouldBeNil)
					So(string(body), ShouldResemble, "transfer interrupted by a server shutdown")

					Convey("Then the transfer should have been interrupted", func(c C) {
						utils.WaitChan(dataReady, time.Second)

						var transfers model.Transfers

						So(test.DB.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)
						So(transfers[0].Status, ShouldEqual, types.StatusInterrupted)

						ok := pipeline.List.Exists(transfers[0].ID)
						So(ok, ShouldBeFalse)
					})
				})
			})
		})
	})
}
