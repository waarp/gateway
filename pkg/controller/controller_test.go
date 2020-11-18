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

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestControllerListen(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.GetTestDatabase()

		remote := &model.RemoteAgent{
			Name:        "test remote",
			Protocol:    "test",
			ProtoConfig: json.RawMessage(`{}`),
			Address:     "localhost:1111",
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

		tmpDir := testhelpers.TempDir(c, "controller-listen")

		rule := &model.Rule{
			Name:    "test rule",
			Path:    "test_rule",
			IsSend:  true,
			OutPath: tmpDir,
		}
		So(db.Create(rule), ShouldBeNil)

		sleepTask := &testTaskSleep{}
		tasks.RunnableTasks["TESTSLEEP"] = sleepTask
		defer delete(tasks.RunnableTasks, "TESTSLEEP")

		model.ValidTasks["TESTSLEEP"] = sleepTask
		defer delete(tasks.RunnableTasks, "TESTSLEEP")

		ruleTask := &model.Task{
			RuleID: rule.ID,
			Chain:  model.ChainPre,
			Rank:   1,
			Type:   "TESTSLEEP",
			Args:   json.RawMessage([]byte("{}")),
		}
		So(db.Create(ruleTask), ShouldBeNil)

		start := time.Now().Truncate(time.Second)

		Convey("Given a controller", func() {
			tick := time.Nanosecond
			cont := &Controller{
				DB:     db,
				Conf:   &conf.ServerConfig{Paths: conf.PathsConfig{GatewayHome: "."}},
				ticker: time.NewTicker(tick),
				logger: log.NewLogger("test_controller"),
				wg:     new(sync.WaitGroup),
				ctx:    context.Background(),
			}

			Convey("Given a planned transfer", func() {
				err := ioutil.WriteFile(filepath.Join(tmpDir, "source_file_1"),
					[]byte("hello world"), 0o644)
				So(err, ShouldBeNil)

				trans := &model.Transfer{
					RuleID:       rule.ID,
					IsServer:     false,
					AgentID:      remote.ID,
					AccountID:    account.ID,
					TrueFilepath: "/filepath_1",
					SourceFile:   "source_file_1",
					DestFile:     "dest_file_1",
					Start:        start,
					Status:       types.StatusPlanned,
					Owner:        database.Owner,
				}
				So(db.Create(trans), ShouldBeNil)

				Convey("When the controller starts new transfers", func() {
					cont.startNewTransfers()
					Reset(func() {
						_ = os.RemoveAll("tmp")
						_ = os.RemoveAll(rule.Path)
					})

					Convey("After waiting enough time", func() {
						cont.wg.Wait()
						// time.Sleep(1000 * time.Millisecond)

						Convey("Then it should have retrieved the planned "+
							"transfer entry", func() {
							//err := db.Get(&model.TransferHistory{ID: trans.ID})
							//So(err, ShouldBeNil)
							var h []model.TransferHistory
							So(db.Select(&h, nil), ShouldBeNil)
							So(h, ShouldNotBeEmpty)
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
				err := ioutil.WriteFile(filepath.Join(tmpDir, "source_file_2"),
					[]byte("hello world"), 0o644)
				So(err, ShouldBeNil)

				trans := &model.Transfer{
					RuleID:       rule.ID,
					IsServer:     false,
					AgentID:      remote.ID,
					AccountID:    account.ID,
					TrueFilepath: "/filepath_2",
					SourceFile:   "source_file_2",
					DestFile:     "dest_file_2",
					Start:        start,
					Status:       types.StatusRunning,
					Owner:        database.Owner,
				}
				So(db.Create(trans), ShouldBeNil)

				Convey("Given that the database stops responding", func() {
					db.State().Set(service.Error, "test error")

					Convey("When the controller starts listening", func() {
						cont.listen()

						Convey("When the database comes back online", func() {
							time.Sleep(100 * time.Millisecond)
							db.State().Set(service.Running, "")

							Convey("After waiting enough time", func() {
								time.Sleep(100 * time.Millisecond)

								Convey("Then the running entry should now be "+
									"interrupted", func() {
									result := &model.Transfer{ID: trans.ID}
									So(db.Get(result), ShouldBeNil)
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

func (t *testTaskSleep) Run(map[string]string, *tasks.Processor) (string, error) {
	t.c++

	time.Sleep(30 * time.Millisecond)

	return "", nil
}
