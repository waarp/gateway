package executor

import (
	"fmt"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

var logConf = conf.LogConfig{
	Level: "DEBUG",
	LogTo: "stdout",
}

func TestTransferInfo(t *testing.T) {

	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		expectedRemote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
		}
		So(db.Create(expectedRemote), ShouldBeNil)

		expectedAccount := &model.RemoteAccount{
			RemoteAgentID: expectedRemote.ID,
			Login:         "test login",
			Password:      []byte("test password"),
		}
		So(db.Create(expectedAccount), ShouldBeNil)

		expectedCert := &model.Cert{
			OwnerType:   expectedRemote.TableName(),
			OwnerID:     expectedRemote.ID,
			Name:        "test cert",
			PrivateKey:  nil,
			PublicKey:   []byte("public key"),
			Certificate: []byte("certificate"),
		}
		So(db.Create(expectedCert), ShouldBeNil)

		expectedRule := &model.Rule{
			Name:  "test rule",
			IsGet: false,
		}
		So(db.Create(expectedRule), ShouldBeNil)

		Convey("Given a transfer entry", func() {
			trans := &model.Transfer{
				ID:         1,
				RuleID:     expectedRule.ID,
				RemoteID:   expectedRemote.ID,
				AccountID:  expectedAccount.ID,
				SourcePath: "test/source/path",
				DestPath:   "test/dest/path",
				Start:      time.Now(),
				Status:     model.StatusPlanned,
				Owner:      database.Owner,
			}

			Convey("When calling the `transferInfo` function", func() {
				info, err := newTransferInfo(db, trans)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return the correct partner", func() {
					So(info.remoteAgent, ShouldResemble, expectedRemote)
				})

				Convey("Then it should return the correct account", func() {
					So(info.remoteAccount, ShouldResemble, expectedAccount)
				})

				Convey("Then it should return the correct certificate", func() {
					So(info.remoteCert, ShouldResemble, expectedCert)
				})

				Convey("Then it should return the correct rule", func() {
					So(info.rule, ShouldResemble, expectedRule)
				})
			})
		})
	})
}

func TestExecutorLogTransfer(t *testing.T) {

	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		remote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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

		rule := &model.Rule{
			Name:  "test rule",
			IsGet: false,
		}
		So(db.Create(rule), ShouldBeNil)

		Convey("Given a transfer entry", func() {
			trans := &model.Transfer{
				RuleID:     rule.ID,
				RemoteID:   remote.ID,
				AccountID:  account.ID,
				SourcePath: "test/source/path",
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

				Convey("When calling the `logTransfer` method with NO error", func() {
					err := exe.logTransfer(trans, nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

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

				Convey("When calling the `logTransfer` method with an error", func() {
					err := exe.logTransfer(trans, fmt.Errorf("error"))

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the `Transfer` entry should no longer exist", func() {
						exist, err := db.Exists(trans)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})

					Convey("Then the corresponding `TransferHistory` entry should "+
						"exist with an ERROR status", func() {
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

						exist, err := db.Exists(hist)
						So(err, ShouldBeNil)
						So(exist, ShouldBeTrue)
					})
				})
			})
		})
	})
}

func TestExecutorRunTransfer(t *testing.T) {

	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		remote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "sftp",
			ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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

		rule := &model.Rule{
			Name:  "test rule",
			IsGet: false,
		}
		So(db.Create(rule), ShouldBeNil)

		Convey("Given a transfer entry", func() {
			trans := &model.Transfer{
				RuleID:     rule.ID,
				RemoteID:   remote.ID,
				AccountID:  account.ID,
				SourcePath: "test/source/path",
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

				Convey("Given that the SFTP transfer is successful", func() {
					run := func(*transferInfo) error {
						return nil
					}

					Convey("When calling the `runTransfer` method", func() {
						exe.runTransfer(trans, run)

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

				Convey("Given that the SFTP transfer is a failure", func() {
					run := func(*transferInfo) error {
						return fmt.Errorf("error")
					}

					Convey("When calling the `runTransfer` method", func() {
						exe.runTransfer(trans, run)

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

							exist, err := db.Exists(hist)
							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})
			})
		})
	})
}
