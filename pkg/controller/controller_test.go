package controller

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestControllerListen(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		remote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "test",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:1111",
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

		sleepTask := &testTaskSleep{}

		model.ValidTasks["TESTSLEEP"] = sleepTask
		defer delete(model.ValidTasks, "TESTSLEEP")

		ruleTask := &model.Task{
			RuleID: rule.ID,
			Chain:  model.ChainPre,
			Rank:   1,
			Type:   "TESTSLEEP",
			Args:   json.RawMessage("{}"),
		}
		So(db.Insert(ruleTask).Run(), ShouldBeNil)

		start := time.Now()

		Convey("Given a controller", func() {
			tick := time.Nanosecond
			cont := &Controller{
				DB:     db,
				ticker: time.NewTicker(tick),
				logger: log.NewLogger("test_controller"),
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
					Start:      start,
					Status:     types.StatusPlanned,
					Owner:      database.Owner,
				}
				So(db.Insert(trans).Run(), ShouldBeNil)

				Convey("When the controller starts new transfers", func() {
					cont.startNewTransfers()
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

							var hist model.HistoryEntries
							So(db.Select(&hist).Run(), ShouldBeNil)
							So(hist, ShouldNotBeEmpty)
						})
					})
				})

				Convey("When the transfer lasts longer than a controller tick", func() {
					Convey("When the controller starts new transfers several times", func() {
						cont.startNewTransfers()
						time.Sleep(10 * time.Millisecond)

						cont.startNewTransfers()
						time.Sleep(10 * time.Millisecond)

						cont.startNewTransfers()

						cont.wg.Wait()

						Convey("Then the transfer has only been started once", func() {
							So(sleepTask.c, ShouldEqual, 1)
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
					Start:      start,
					Status:     types.StatusRunning,
					Owner:      database.Owner,
				}
				So(db.Insert(trans).Run(), ShouldBeNil)

				Convey("Given that the database stops responding", func() {
					db.State().Set(service.Error, "test error")

					Convey("When the controller starts listening", func() {
						cont.listen()
						Reset(func() {
							ctx, cancel := context.WithTimeout(context.Background(), time.Second)
							defer cancel()
							So(cont.Stop(ctx), ShouldBeNil)
						})

						Convey("When the database comes back online", func() {
							time.Sleep(100 * time.Millisecond)
							db.State().Set(service.Running, "")

							Convey("After waiting enough time", func() {
								time.Sleep(100 * time.Millisecond)

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
	})
}

type testTaskSleep struct {
	c int
}

func (t *testTaskSleep) Validate(map[string]string) error {
	return nil
}

func (t *testTaskSleep) Run(context.Context, map[string]string, *database.DB, *model.TransferContext) (string, error) {
	t.c++

	time.Sleep(30 * time.Millisecond)

	return "", nil
}
