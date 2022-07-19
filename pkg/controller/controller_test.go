package controller

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestControllerListen(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		logger := testhelpers.TestLogger(c, "test_controller")
		db := database.TestDatabase(c)

		remote := &model.RemoteAgent{
			Name:     "test remote",
			Protocol: testProtocol,
			Address:  "localhost:1111",
		}
		So(db.Insert(remote).Run(), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "test login",
			Password:      "test password",
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		tmpDir := testhelpers.TempDir(c, "controller-listen")

		rule := &model.Rule{
			Name:     "test rule",
			Path:     "test_rule",
			IsSend:   true,
			LocalDir: tmpDir,
		}
		So(db.Insert(rule).Run(), ShouldBeNil)

		Convey("Given a controller", func() {
			conf.GlobalConfig.Paths = conf.PathsConfig{GatewayHome: tmpDir}
			gwController := GatewayController{DB: db}
			cont := &Controller{
				Action: gwController.Run,
				ticker: time.NewTicker(time.Millisecond),
				logger: logger,
				wg:     new(sync.WaitGroup),
				ctx:    context.Background(),
			}

			Convey("Given a planned transfer", func() {
				path1 := filepath.Join(tmpDir, "file_1")
				err := ioutil.WriteFile(path1, []byte("hello world"), 0o644)
				So(err, ShouldBeNil)

				trans := &model.Transfer{
					RuleID:     rule.ID,
					IsServer:   false,
					AgentID:    remote.ID,
					AccountID:  account.ID,
					LocalPath:  path1,
					RemotePath: "/file_1",
					Start:      time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
					Status:     types.StatusPlanned,
					Owner:      conf.GlobalConfig.GatewayName,
				}
				So(db.Insert(trans).Run(), ShouldBeNil)

				Convey("When the controller starts new transfers", func() {
					cont.Action(cont.wg, *cont.logger)
					Reset(func() {
						_ = os.RemoveAll("tmp")
						_ = os.RemoveAll(rule.Path)
					})

					Convey("After waiting enough time", func() {
						cont.wg.Wait()

						Convey("Then it should have retrieved the planned "+
							"transfer entry", func() {
							var trans model.Transfers
							So(db.Select(&trans).Run(), ShouldBeNil)
							So(trans, ShouldBeEmpty)
						})

						Convey("Then it should have created the new history entries", func() {
							var hist model.HistoryEntries
							So(db.Select(&hist).Run(), ShouldBeNil)
							So(hist, ShouldNotBeEmpty)
						})
					})
				})

				Convey("When the transfer lasts longer than a controller tick", func() {
					Convey("When the controller starts new transfers several times", func() {
						cont.Action(cont.wg, *cont.logger)
						time.Sleep(10 * time.Millisecond)

						cont.Action(cont.wg, *cont.logger)
						time.Sleep(10 * time.Millisecond)

						cont.Action(cont.wg, *cont.logger)

						cont.wg.Wait()

						Convey("Then the transfer has only been started once", func() {
							var hist model.HistoryEntries
							So(db.Select(&hist).Run(), ShouldBeNil)
							So(hist, ShouldHaveLength, 1)
						})
					})
				})
			})

			Convey("Given a running transfer", func() {
				path2 := filepath.Join(tmpDir, "file_2")
				err := ioutil.WriteFile(path2, []byte("hello world"), 0o644)
				So(err, ShouldBeNil)

				trans := &model.Transfer{
					RuleID:     rule.ID,
					IsServer:   false,
					AgentID:    remote.ID,
					AccountID:  account.ID,
					LocalPath:  path2,
					RemotePath: "/file_2",
					Start:      time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
					Status:     types.StatusRunning,
					Owner:      conf.GlobalConfig.GatewayName,
				}
				So(gwController.DB.Insert(trans).Run(), ShouldBeNil)

				Convey("Given that the database stops responding", func() {
					gwController.DB.State().Set(service.Error, "test error")
					gwController.wasDown = true

					Convey("When the database comes back online", func() {
						gwController.DB.State().Set(service.Running, "")

						Convey("When the controller starts new transfers again", func() {
							cont.Action(cont.wg, *cont.logger)
							So(gwController.wasDown, ShouldBeFalse)

							Convey("Then the running entry should now be "+
								"interrupted", func() {
								result := &model.Transfer{}
								So(db.Get(result, "id=?", trans.ID).Run(), ShouldBeNil)
								So(result.Status, ShouldEqual, types.StatusInterrupted)
							})
						})
					})
				})
			})
		})
	})
}
