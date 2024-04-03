package controller

import (
	"context"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocolstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestControllerListen(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		testFS := fstest.InitMemFS(c)
		logger := testhelpers.TestLogger(c, "test_controller")
		db := database.TestDatabase(c)

		client := &model.Client{Name: "client", Protocol: testProtocol}
		So(db.Insert(client).Run(), ShouldBeNil)

		cliService := &protocolstest.TestService{}
		So(cliService.Start(), ShouldBeNil)

		services.Clients[client.Name] = cliService
		defer delete(services.Clients, client.Name)

		remote := &model.RemoteAgent{
			Name: "test remote", Protocol: client.Protocol,
			Address: types.Addr("localhost", 1111),
		}
		So(db.Insert(remote).Run(), ShouldBeNil)

		account := &model.RemoteAccount{
			RemoteAgentID: remote.ID,
			Login:         "test login",
		}
		So(db.Insert(account).Run(), ShouldBeNil)

		rootDir := "memory:/controller-listen"
		rootPath := mkURL(rootDir)

		rule := &model.Rule{Name: "test rule", IsSend: true}
		So(db.Insert(rule).Run(), ShouldBeNil)

		Convey("Given a controller", func() {
			conf.GlobalConfig.Paths = conf.PathsConfig{
				GatewayHome: rootDir, DefaultInDir: "in",
				DefaultOutDir: "out", DefaultTmpDir: "tmp",
			}

			gwController := GatewayController{DB: db}
			cont := &Controller{
				Action: gwController.Run,
				ticker: time.NewTicker(time.Millisecond),
				logger: logger,
				wg:     new(sync.WaitGroup),
				ctx:    context.Background(),
			}

			Convey("Given a planned transfer", func(c C) {
				path1 := rootPath.JoinPath("out", "file_1")
				So(fs.MkdirAll(testFS, path1.Dir()), ShouldBeNil)
				So(fs.WriteFullFile(testFS, path1, []byte("hello world")), ShouldBeNil)

				trans := &model.Transfer{
					RuleID:          rule.ID,
					ClientID:        utils.NewNullInt64(client.ID),
					RemoteAccountID: utils.NewNullInt64(account.ID),
					SrcFilename:     "file_1",
					Start:           time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
					Status:          types.StatusPlanned,
					Owner:           conf.GlobalConfig.GatewayName,
				}
				So(db.Insert(trans).Run(), ShouldBeNil)

				Convey("When the controller starts new transfers", func() {
					cont.Action(cont.wg, *cont.logger)

					Convey("After waiting enough time", func() {
						cont.wg.Wait()

						Convey("Then it should have retrieved the planned "+
							"transfer entry", func() {
							var transfers model.Transfers

							So(db.Select(&transfers).Run(), ShouldBeNil)
							So(transfers, ShouldBeEmpty)
						})

						Convey("Then it should have created the new history entries", func() {
							var historyEntries model.HistoryEntries

							So(db.Select(&historyEntries).Run(), ShouldBeNil)
							So(historyEntries, ShouldNotBeEmpty)
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
							var historyEntries model.HistoryEntries

							So(db.Select(&historyEntries).Run(), ShouldBeNil)
							So(historyEntries, ShouldHaveLength, 1)
						})
					})
				})
			})

			Convey("Given a running transfer", func(c C) {
				path2 := rootPath.JoinPath("out", "file_2")
				So(fs.MkdirAll(testFS, path2.Dir()), ShouldBeNil)
				So(fs.WriteFullFile(testFS, path2, []byte("hello world")), ShouldBeNil)

				trans := &model.Transfer{
					RuleID:          rule.ID,
					ClientID:        utils.NewNullInt64(client.ID),
					RemoteAccountID: utils.NewNullInt64(account.ID),
					SrcFilename:     "file2",
					Start:           time.Date(2022, 1, 1, 1, 0, 0, 0, time.UTC),
					Status:          types.StatusRunning,
					Owner:           conf.GlobalConfig.GatewayName,
				}
				So(db.Insert(trans).Run(), ShouldBeNil)

				Convey("Given that the database stops responding", func() {
					gwController.wasDown = true

					Convey("When the database comes back online", func() {
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
