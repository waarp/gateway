package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/http/httpconst"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/gatewayd"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"
	. "github.com/smartystreets/goconvey/convey"
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
		test := pipelinetest.InitServerPush(c, "http", nil)

		serv := gatewayd.ServiceConstructors["http"](test.DB, test.Server, log.NewLogger("server"))
		c.So(serv.Start(), ShouldBeNil)

		Convey("Given a dummy HTTP client", func() {
			cli := http.DefaultClient

			Convey("Given that a push transfer started", func() {
				r, w, err := os.Pipe()
				So(err, ShouldBeNil)
				defer w.Close()

				url := fmt.Sprintf("http://%s/test_in_shutdown.dst?%s=%s",
					test.Server.Address, httpconst.Rule, test.ServerRule.Name)
				req, err := http.NewRequest(http.MethodPost, url, r)
				So(err, ShouldBeNil)
				req.SetBasicAuth(pipelinetest.TestLogin, pipelinetest.TestPassword)

				resChan := make(chan struct {
					*http.Response
					error
				})
				go func() {
					resp, err := cli.Do(req)
					resChan <- struct {
						*http.Response
						error
					}{Response: resp, error: err}
				}()
				_, err = w.Write([]byte("abc"))
				So(err, ShouldBeNil)

				Convey("When the server shuts down", func() {
					time.Sleep(500 * time.Millisecond)
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
					defer cancel()
					So(serv.Stop(ctx), ShouldBeNil)

					result := <-resChan
					So(result.error, ShouldBeNil)
					So(result.StatusCode, ShouldEqual, http.StatusServiceUnavailable)
					So(result.Header.Get(httpconst.TransferStatus), ShouldEqual, types.StatusInterrupted)
					body, err := ioutil.ReadAll(result.Body)
					So(err, ShouldBeNil)
					So(string(body), ShouldResemble, "transfer interrupted by a server shutdown")

					Convey("Then the transfer should have been interrupted", func(c C) {
						test.ServerShouldHavePreTasked(c)
						test.TasksChecker.WaitServerDone()

						var transfers model.Transfers
						So(test.DB.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)

						trans := model.Transfer{
							ID:               transfers[0].ID,
							RemoteTransferID: "",
							Start:            transfers[0].Start,
							IsServer:         true,
							AccountID:        test.LocAccount.ID,
							AgentID:          test.Server.ID,
							LocalPath: filepath.Join(test.Server.Root,
								test.Server.LocalTmpDir, "test_in_shutdown.dst.part"),
							RemotePath: "/test_in_shutdown.dst",
							Filesize:   -1,
							RuleID:     test.ServerRule.ID,
							Status:     types.StatusInterrupted,
							Step:       types.StepData,
							Owner:      database.Owner,
							Progress:   3,
						}
						So(transfers[0], ShouldResemble, trans)

						ok := serv.(*httpService).running.Exists(trans.ID)
						So(ok, ShouldBeFalse)
					})
				})
			})
		})
	})
}
