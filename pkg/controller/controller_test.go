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

		Convey("Given 2 transfer entries", func() {
			start := time.Now().Truncate(time.Second)

			trans1 := &model.Transfer{
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
			So(db.Create(trans1), ShouldBeNil)

			trans2 := &model.Transfer{
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
			So(db.Create(trans2), ShouldBeNil)

			Convey("Given a controller", func() {
				tick := time.Millisecond
				cont := &Controller{
					ticker: *time.NewTicker(tick),
					Db:     db,
					logger: log.NewLogger("test_controller", logConf),
					state:  service.State{},
					pool:   make(chan model.Transfer),
				}
				cont.state.Set(service.Running, "")

				Convey("When calling the `listen` method", func() {
					cont.listen()

					Convey("After waiting enough time", func() {
						test := <-cont.pool
						cont.state.Set(service.Offline, "")

						Convey("Then it should have retrieved the planned "+
							"transfer entry", func() {
							So(test, ShouldResemble, *trans1)
						})
					})
				})

				Convey("Given that the database stops responding", func() {
					db.State().Set(service.Error, "test error")

					Convey("When calling the `listen` method", func() {
						cont.listen()

						Convey("When the database comes back online", func() {
							<-cont.pool
							db.State().Set(service.Running, "")

							Convey("After waiting enough time", func() {
								test := <-cont.pool
								cont.state.Set(service.Offline, "")

								Convey("Then it should have retrieved the planned"+
									" transfer entry", func() {
									So(test, ShouldResemble, *trans1)
								})

								Convey("Then the running entry should now be "+
									"interrupted", func() {

									result := &model.Transfer{ID: trans2.ID}
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
