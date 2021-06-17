package r66

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/gatewayd"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-r66/r66"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestServiceStart(t *testing.T) {
	logger := log.NewLogger("test_r66_start")

	Convey("Given an R66 service", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		server := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    "r66",
			ProtoConfig: json.RawMessage(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:8066",
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
	logger := log.NewLogger("test_r66_stop")

	Convey("Given a running R66 service", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		server := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    "r66",
			ProtoConfig: json.RawMessage(`{"blockSize":512,"serverPassword":"c2VzYW1l"}`),
			Address:     "localhost:8067",
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

func TestR66ServerInterruption(t *testing.T) {

	Convey("Given an SFTP server ready for push transfers", t, func(c C) {
		test := pipelinetest.InitServerPush(c, "r66", servConf)

		serv := gatewayd.ServiceConstructors["r66"](test.DB, test.Server, log.NewLogger("server"))
		c.So(serv.Start(), ShouldBeNil)

		Convey("Given a dummy R66 client", func() {
			ses := makeDummyClient(test)

			Convey("Given that a push transfer started", func() {
				req := &r66.Request{
					ID:       1,
					Filepath: "/test_in_shutdown.dst",
					FileSize: 100,
					Rule:     test.Rule.Name,
					IsRecv:   false,
					IsMD5:    false,
					Block:    10,
					Rank:     0,
				}
				_, err := ses.Request(req)
				So(err, ShouldBeNil)

				Convey("When the server shuts down", func(c C) {
					go func() {
						time.Sleep(500 * time.Millisecond)
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
						defer cancel()
						if err := serv.Stop(ctx); err != nil {
							panic(err)
						}
					}()
					_, err := ses.Send(&dummyFile{}, func() ([]byte, error) { panic("should never be called") })
					So(err, ShouldBeError, "D: connection closed unexpectedly")

					Convey("Then the transfer should have been interrupted", func(c C) {
						test.PreTasksShouldBeOK(c)
						test.ShouldBeEndTransfer(c)

						var transfers model.Transfers
						So(test.DB.Select(&transfers).Run(), ShouldBeNil)
						So(transfers, ShouldNotBeEmpty)

						trans := model.Transfer{
							ID:               transfers[0].ID,
							RemoteTransferID: fmt.Sprint(req.ID),
							Start:            transfers[0].Start,
							IsServer:         true,
							AccountID:        test.LocAccount.ID,
							AgentID:          test.Server.ID,
							LocalPath: filepath.Join(test.Server.Root,
								test.Server.LocalTmpDir, "test_in_shutdown.dst.part"),
							RemotePath: "/test_in_shutdown.dst",
							Filesize:   100,
							RuleID:     test.Rule.ID,
							Status:     types.StatusInterrupted,
							Step:       types.StepData,
							Owner:      database.Owner,
							Progress:   transfers[0].Progress,
						}
						So(transfers[0], ShouldResemble, trans)

						ok := serv.(*Service).runningTransfers.Exists(trans.ID)
						So(ok, ShouldBeFalse)
					})
				})
			})
		})
	})
}

func makeDummyClient(test *pipelinetest.ServerContext) *r66.Session {
	logger := log.NewLogger("client")
	cli, err := r66.Dial(test.Server.Address, logger.AsStdLog(logging.DEBUG))
	So(err, ShouldBeNil)

	ses, err := cli.NewSession()
	So(err, ShouldBeNil)
	_, err = ses.Authent(pipelinetest.TestLogin, []byte(pipelinetest.TestPassword), &r66.Config{})
	So(err, ShouldBeNil)

	return ses
}

type dummyFile struct{}

func (d *dummyFile) ReadAt(p []byte, _ int64) (int, error) {
	time.Sleep(100 * time.Millisecond)
	return rand.Read(p)
}
