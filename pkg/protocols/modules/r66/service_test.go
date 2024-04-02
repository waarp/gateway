package r66

import (
	"context"
	"testing"
	"time"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestServiceStart(t *testing.T) {
	Convey("Given an R66 service", t, func(c C) {
		db := database.TestDatabase(c)
		server := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    R66,
			ProtoConfig: map[string]any{"blockSize": 512, "serverPassword": "c2VzYW1l"},
			Address:     testhelpers.GetLocalAddress(c),
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		serv := &service{db: db, agent: server}

		Convey("When calling the 'Start' function", func() {
			err := serv.Start()

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given an R66-TLS service", t, func(c C) {
		db := database.TestDatabase(c)
		server := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    R66,
			ProtoConfig: map[string]any{"blockSize": 512, "serverPassword": "c2VzYW1l", "isTLS": true},
			Address:     testhelpers.GetLocalAddress(c),
		}
		So(db.Insert(server).Run(), ShouldBeNil)
		cert := &model.Crypto{
			LocalAgentID: utils.NewNullInt64(server.ID),
			Name:         "r66_cert",
			PrivateKey:   testhelpers.LocalhostKey,
			Certificate:  testhelpers.LocalhostCert,
		}
		So(db.Insert(cert).Run(), ShouldBeNil)

		serv := &service{db: db, agent: server}

		Convey("When calling the 'Start' function", func() {
			err := serv.Start()

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	// Is this still relevant now that certificates can be added post start?
	SkipConvey("Given an R66-TLS service with no certificate", t, func(c C) {
		db := database.TestDatabase(c)
		server := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    R66,
			ProtoConfig: map[string]any{"blockSize": 512, "serverPassword": "c2VzYW1l", "isTLS": true},
			Address:     testhelpers.GetLocalAddress(c),
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		serv := &service{db: db, agent: server}

		Convey("When calling the 'Start' function", func() {
			err := serv.Start()

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, errNoCertificates)
			})
		})
	})
}

func TestServiceStop(t *testing.T) {
	Convey("Given a running R66 service", t, func(c C) {
		db := database.TestDatabase(c)
		server := &model.LocalAgent{
			Name:        "r66_server",
			Protocol:    R66,
			ProtoConfig: map[string]any{"blockSize": 512, "serverPassword": "c2VzYW1l"},
			Address:     "localhost:0",
		}
		So(db.Insert(server).Run(), ShouldBeNil)

		serv := &service{db: db, agent: server}
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
	Convey("Given an R66 server ready for push transfers", t, func(c C) {
		test := pipelinetest.InitServerPush(c, R66, servConf)

		serv := &service{db: test.DB, agent: test.Server}
		c.So(serv.Start(), ShouldBeNil)

		preTasksDone := make(chan bool)
		transferDone := make(chan bool)

		serv.SetTracer(func() pipeline.Trace {
			return pipeline.Trace{
				OnPreTask: func(int8) error {
					close(preTasksDone)

					return nil
				},
				OnTransferEnd: func() { close(transferDone) },
			}
		})

		Convey("Given a dummy R66 client", func(c C) {
			ses := makeDummyClient(c, test)

			Convey("Given that a push transfer started", func() {
				req := &r66.Request{
					ID:       1,
					Filepath: "test_in_shutdown.dst",
					FileSize: 100,
					Rule:     test.ServerRule.Name,
					IsRecv:   false,
					IsMD5:    false,
					Block:    10,
					Rank:     0,
				}
				_, err := ses.Request(req)
				So(err, ShouldBeNil)

				So(ses.SendUpdateRequest(&r66.UpdateInfo{
					Filename: req.Filepath,
					FileSize: req.FileSize,
					FileInfo: &r66.TransferData{
						UserContent: "",
						SystemData:  r66.SystemData{},
					},
				}), ShouldBeNil)

				Convey("When the server shuts down", func(c C) {
					go func() {
						utils.WaitChan(preTasksDone, time.Second)

						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()

						if err := serv.Stop(ctx); err != nil {
							panic(err)
						}
					}()

					f := func() ([]byte, error) { panic("should never be called") }
					file := testhelpers.NewSlowReader()
					_, err := ses.Send(file, f)

					So(err, ShouldBeError, &r66.Error{
						Code:   r66.Shutdown,
						Detail: "service is shutting down",
					})

					Convey("Then the transfer should have been interrupted", func(c C) {
						So(utils.WaitChan(transferDone, 5*time.Second), ShouldBeTrue)

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

func makeDummyClient(c C, test *pipelinetest.ServerContext) *r66.Session {
	logger := testhelpers.TestLogger(c, "r66_dummy_client")
	cli, err := r66.Dial(test.Server.Address, logger.AsStdLogger(log.LevelTrace))
	So(err, ShouldBeNil)

	ses, err := cli.NewSession()
	So(err, ShouldBeNil)
	_, err = ses.Authent(pipelinetest.TestLogin, []byte(pipelinetest.TestPassword), &r66.Config{})
	So(err, ShouldBeNil)

	return ses
}
