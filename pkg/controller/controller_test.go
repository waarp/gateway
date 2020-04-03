package controller

import (
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error { return nil }
func (*TestProtoConfig) ValidClient() error { return nil }

func TestControllerListen(t *testing.T) {
	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}

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

		rule := &model.Rule{
			Name:   "test rule",
			IsSend: false,
		}
		So(db.Create(rule), ShouldBeNil)

		start := time.Now().Truncate(time.Second)

		Convey("Given a controller", func() {
			tick := time.Nanosecond
			cont := &Controller{
				Db:     db,
				ticker: time.NewTicker(tick),
				logger: log.NewLogger("test_controller", logConf),
			}

			Convey("Given a planned transfer", func() {
				trans := &model.Transfer{
					RuleID:     rule.ID,
					IsServer:   false,
					AgentID:    remote.ID,
					AccountID:  account.ID,
					SourcePath: "test/source/path1",
					DestPath:   "test/dest/path1",
					Start:      start,
					Status:     model.StatusPlanned,
					Owner:      database.Owner,
				}
				So(db.Create(trans), ShouldBeNil)

				Convey("When calling the `listen` method", func() {
					cont.listen()

					Convey("After waiting enough time", func() {
						time.Sleep(10 * time.Millisecond)

						Convey("Then it should have retrieved the planned "+
							"transfer entry", func() {
							err := db.Get(&model.TransferHistory{ID: trans.ID})
							So(err, ShouldBeNil)
						})
					})
				})
			})

			Convey("Given a running transfer", func() {
				trans := &model.Transfer{
					RuleID:     rule.ID,
					IsServer:   false,
					AgentID:    remote.ID,
					AccountID:  account.ID,
					SourcePath: "test/source/path2",
					DestPath:   "test/dest/path2",
					Start:      start,
					Status:     model.StatusRunning,
					Owner:      database.Owner,
				}
				So(db.Create(trans), ShouldBeNil)

				Convey("Given that the database stops responding", func() {
					db.State().Set(service.Error, "test error")

					Convey("When calling the `listen` method", func() {
						cont.listen()

						Convey("When the database comes back online", func() {
							time.Sleep(10 * time.Millisecond)
							db.State().Set(service.Running, "")

							Convey("After waiting enough time", func() {
								time.Sleep(10 * time.Millisecond)

								Convey("Then the running entry should now be "+
									"interrupted", func() {

									result := &model.Transfer{ID: trans.ID}
									So(db.Get(result), ShouldBeNil)
									So(result.Status, ShouldEqual, model.StatusInterrupted)
								})
							})
						})
					})
				})
			})
		})
	})
}
