package executor

import (
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	. "github.com/smartystreets/goconvey/convey"
)

var logConf = conf.LogConfig{
	Level: "DEBUG",
	LogTo: "stdout",
}

func TestExecutorRunTransfer(t *testing.T) {

	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		remote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "test",
			ProtoConfig: []byte(`{}`),
		}
		So(db.Create(remote), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "test login",
			Password:      []byte("test password"),
		}
		So(db.Create(account), ShouldBeNil)

		cert := &model.Cert{
			OwnerType:   remote.TableName(),
			OwnerID:     remote.ID,
			Name:        "test cert",
			PrivateKey:  nil,
			PublicKey:   []byte("public key"),
			Certificate: []byte("certificate"),
		}
		So(db.Create(cert), ShouldBeNil)

		Convey("Given an outgoing transfer", func() {
			rule := &model.Rule{
				Name:   "test_rule",
				IsSend: true,
			}
			So(db.Create(rule), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:     rule.ID,
				AgentID:    remote.ID,
				AccountID:  account.ID,
				SourcePath: "executor.go",
				DestPath:   "test/dest/path",
				Start:      time.Now(),
				Status:     model.StatusPlanned,
				Owner:      database.Owner,
			}
			So(db.Create(trans), ShouldBeNil)

			Convey("Given an executor", func() {
				exe := &Executor{
					Db:     db,
					Logger: log.NewLogger("test_executor", logConf),
				}
				stream, err := pipeline.NewTransferStream(exe.Logger, exe.Db, "", *trans)
				So(err.Code, ShouldEqual, model.TeOk)

				Convey("Given that the transfer is successful", func() {
					ClientsConstructors["test"] = NewAllSuccess

					Convey("When calling the `runTransfer` method", func() {
						exe.runTransfer(stream)

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry should exist", func() {
							hist := &model.TransferHistory{
								ID:             trans.ID,
								Owner:          trans.Owner,
								IsServer:       false,
								IsSend:         true,
								Account:        account.Login,
								Remote:         remote.Name,
								Protocol:       remote.Protocol,
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           rule.Name,
								Start:          trans.Start,
								Status:         model.StatusDone,
							}

							exist, err := db.Exists(hist)
							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the connection fails", func() {
					ClientsConstructors["test"] = NewConnectFail

					Convey("When calling the `runTransfer` method", func() {
						exe.runTransfer(stream)

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
								ID:             trans.ID,
								Owner:          trans.Owner,
								IsServer:       false,
								IsSend:         true,
								Account:        account.Login,
								Remote:         remote.Name,
								Protocol:       remote.Protocol,
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           rule.Name,
								Start:          trans.Start,
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeConnection,
								"connection failed")
							So(hist.Error, ShouldResemble, expErr)
						})
					})
				})

				Convey("Given that the authentication fails", func() {
					ClientsConstructors["test"] = NewAuthFail

					Convey("When calling the `runTransfer` method", func() {
						exe.runTransfer(stream)

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
								ID:             trans.ID,
								Owner:          trans.Owner,
								IsServer:       false,
								IsSend:         true,
								Account:        account.Login,
								Remote:         remote.Name,
								Protocol:       remote.Protocol,
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           rule.Name,
								Start:          trans.Start,
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeBadAuthentication,
								"authentication failed")
							So(hist.Error, ShouldResemble, expErr)
						})
					})
				})

				Convey("Given that the request fails", func() {
					ClientsConstructors["test"] = NewRequestFail

					Convey("When calling the `runTransfer` method", func() {
						exe.runTransfer(stream)

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
								ID:             trans.ID,
								Owner:          trans.Owner,
								IsServer:       false,
								IsSend:         true,
								Account:        account.Login,
								Remote:         remote.Name,
								Protocol:       remote.Protocol,
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           rule.Name,
								Start:          trans.Start,
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeForbidden,
								"request failed")
							So(hist.Error, ShouldResemble, expErr)
						})
					})
				})

				Convey("Given that the pre-tasks fail", func() {
					ClientsConstructors["test"] = NewAllSuccess

					preTask := &model.Task{
						RuleID: rule.ID,
						Chain:  model.ChainPre,
						Rank:   0,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(preTask), ShouldBeNil)

					Convey("When calling the `runTransfer` method", func() {
						exe.runTransfer(stream)

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
								ID:             trans.ID,
								Owner:          trans.Owner,
								IsServer:       false,
								IsSend:         true,
								Account:        account.Login,
								Remote:         remote.Name,
								Protocol:       remote.Protocol,
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           rule.Name,
								Start:          trans.Start,
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeExternalOperation,
								"Task TESTFAIL @ test_rule PRE[0]: task failed")
							So(hist.Error, ShouldResemble, expErr)
						})
					})
				})

				Convey("Given that the data transfer fails", func() {
					ClientsConstructors["test"] = NewDataFail

					Convey("When calling the `runTransfer` method", func() {
						exe.runTransfer(stream)

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
								ID:             trans.ID,
								Owner:          trans.Owner,
								IsServer:       false,
								IsSend:         true,
								Account:        account.Login,
								Remote:         remote.Name,
								Protocol:       remote.Protocol,
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           rule.Name,
								Start:          trans.Start,
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeDataTransfer,
								"data failed")
							So(hist.Error, ShouldResemble, expErr)
						})
					})
				})

				Convey("Given that the post-tasks fail", func() {
					ClientsConstructors["test"] = NewAllSuccess

					preTask := &model.Task{
						RuleID: rule.ID,
						Chain:  model.ChainPost,
						Rank:   0,
						Type:   "TESTFAIL",
						Args:   []byte("{}"),
					}
					So(db.Create(preTask), ShouldBeNil)

					Convey("When calling the `runTransfer` method", func() {
						exe.runTransfer(stream)

						Convey("Then the `Transfer` entry should no longer exist", func() {
							exist, err := db.Exists(trans)
							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})

						Convey("Then the corresponding `TransferHistory` entry "+
							"should exist with an ERROR status", func() {
							hist := &model.TransferHistory{
								ID:             trans.ID,
								Owner:          trans.Owner,
								IsServer:       false,
								IsSend:         true,
								Account:        account.Login,
								Remote:         remote.Name,
								Protocol:       remote.Protocol,
								SourceFilename: trans.SourcePath,
								DestFilename:   trans.DestPath,
								Rule:           rule.Name,
								Start:          trans.Start,
								Status:         model.StatusError,
							}

							So(db.Get(hist), ShouldBeNil)
							expErr := model.NewTransferError(model.TeExternalOperation,
								"Task TESTFAIL @ test_rule POST[0]: task failed")
							So(hist.Error, ShouldResemble, expErr)
						})
					})
				})
			})
		})
	})
}
