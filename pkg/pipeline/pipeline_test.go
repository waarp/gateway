package pipeline

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks/taskstest"
)

func TestNewClientPipeline(t *testing.T) {
	TransferOutCount = &count{}
	TransferInCount = &count{}

	Convey("Given a database", t, func(c C) {
		ctx := initTestDB(c)

		Convey("Given a send transfer", func(c C) {
			trans := mkSendTransfer(ctx, "file")
			file := filepath.Join(ctx.root, ctx.send.LocalDir, "file")

			Convey("When initiating a new pipeline for this transfer", func(c C) {
				pip, err := NewClientPipeline(ctx.db, trans)
				So(err, ShouldBeNil)

				Convey("Then it should create the corresponding pipeline", func(c C) {
					So(pip, ShouldNotBeNil)

					Convey("Then the pipeline's state machine should have been initiated", func(c C) {
						So(pip.Pip.machine.Current(), ShouldEqual, stateInit)
					})

					Convey("Then the transfer's paths should have been initiated", func(c C) {
						So(trans.LocalPath, ShouldEqual, file)
						So(trans.RemotePath, ShouldEqual, path.Join("/",
							ctx.send.RemoteDir, "file"))
					})
				})
			})

			Convey("Given that the file cannot be found", func(c C) {
				So(os.Remove(file), ShouldBeNil)

				Convey("When initiating a new pipeline for this transfer", func(c C) {
					_, err := NewClientPipeline(ctx.db, trans)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, types.NewTransferError(
							types.TeFileNotFound, "file does not exist"))
					})
				})
			})

			Convey("Given that the transfer limit has been reached", func(c C) {
				TransferOutCount.SetLimit(1)
				TransferOutCount.Add()
				Reset(func() {
					TransferOutCount.Sub()
					TransferOutCount.SetLimit(0)
				})

				Convey("When initiating a new pipeline for this transfer", func(c C) {
					_, err := NewClientPipeline(ctx.db, trans)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, errLimitReached)
					})
				})
			})

			Convey("Given that a database error occurs", func(c C) {
				database.SimulateError(c, ctx.db)

				Convey("When initiating a new pipeline for this transfer", func(c C) {
					_, err := NewClientPipeline(ctx.db, trans)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, errDatabase)
					})
				})
			})
		})

		Convey("Given a receive transfer", func(c C) {
			filename := "file"
			trans := mkRecvTransfer(ctx, filename)

			Convey("When initiating a new pipeline for this transfer", func(c C) {
				pip, err := NewClientPipeline(ctx.db, trans)
				So(err, ShouldBeNil)

				Convey("Then it should create the corresponding pipeline", func(c C) {
					So(pip, ShouldNotBeNil)

					Convey("Then the pipeline's state machine should have been initiated", func(c C) {
						So(pip.Pip.machine.Current(), ShouldEqual, stateInit)
					})

					Convey("Then the transfer's paths should have been initiated", func(c C) {
						So(trans.LocalPath, ShouldEqual, filepath.Join(
							ctx.root, ctx.recv.TmpLocalRcvDir, filename))
						So(trans.RemotePath, ShouldEqual, path.Join("/",
							ctx.recv.RemoteDir, filename))
					})
				})
			})

			Convey("Given that the transfer limit has been reached", func(c C) {
				TransferInCount.SetLimit(1)
				TransferInCount.Add()
				Reset(func() {
					TransferInCount.Sub()
					TransferInCount.SetLimit(0)
				})

				Convey("When initiating a new pipeline for this transfer", func(c C) {
					_, err := NewClientPipeline(ctx.db, trans)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, errLimitReached)
					})
				})
			})
		})
	})
}

func TestPipelinePreTasks(t *testing.T) {
	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		trans := mkRecvTransfer(ctx, filename)
		pip := newTestPipeline(c, ctx.db, trans)

		pip.TransCtx.PreTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainPre,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   json.RawMessage(`{}`),
		}}
		pip.TransCtx.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   json.RawMessage(`{}`),
		}}

		Convey("When calling the pre-tasks", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)

			Convey("Then it should have executed the pre-tasks", func(c C) {
				So(Tester.CliPre, ShouldEqual, 1)
			})

			Convey("Then any subsequent calls will return an error", func(c C) {
				So(pip.PreTasks(), ShouldBeError, errStateMachine)
				waitEndTransfer(pip)
			})
		})

		Convey("Given that the transfer is a recovery", func(c C) {
			trans.Step = types.StepData

			Convey("When calling the pre-tasks", func(c C) {
				So(pip.PreTasks(), ShouldBeNil)

				Convey("Then it should do nothing", func(c C) {
					So(Tester.CliPre, ShouldEqual, 0)
				})
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			database.SimulateError(c, ctx.db)

			Convey("When calling the pre-tasks", func(c C) {
				So(pip.PreTasks(), ShouldBeError, errDatabase)

				Convey("Then the transfer should end in error", func(c C) {
					waitEndTransfer(pip)
					c.So(Tester.CliErr, ShouldEqual, 1)
				})
			})
		})

		Convey("Given that on of the tasks fails", func(c C) {
			pip.TransCtx.PreTasks = append(pip.TransCtx.PreTasks, model.Task{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.TaskErr,
				Args:   json.RawMessage(`{}`),
			})

			Convey("When calling the pre-tasks", func(c C) {
				So(pip.PreTasks(), ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "pre-tasks failed"))

				Convey("Then the transfer should end in error", func() {
					waitEndTransfer(pip)
					So(Tester.CliPre, ShouldEqual, 1)
					So(Tester.CliErr, ShouldEqual, 1)
				})
			})
		})
	})
}

func TestPipelineStartData(t *testing.T) {
	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		trans := mkRecvTransfer(ctx, filename)

		So(ctx.db.Insert(&model.Task{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   json.RawMessage(`{}`),
		}).Run(), ShouldBeNil)

		pip := newTestPipeline(c, ctx.db, trans)
		So(pip.machine.Transition(statePreTasks), ShouldBeNil)
		So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)

		Convey("When starting the data transfer", func(c C) {
			stream, err := pip.StartData()
			So(err, ShouldBeNil)
			//nolint:forcetypeassert //no need, the type assertion will always succeed
			Reset(func() { _ = stream.(*fileStream).file.Close() })

			Convey("Then it should return a filestream for the transfer file", func(c C) {
				So(stream, ShouldNotBeNil)
				So(stream, ShouldHaveSameTypeAs, &fileStream{})
			})

			Convey("Then it should have opened/created the file", func(c C) {
				_, err := os.Stat(filepath.Join(ctx.root, pip.TransCtx.Rule.
					TmpLocalRcvDir, filename+".part"))
				So(err, ShouldBeNil)
			})

			Convey("Then any subsequent calls to StartData should return an error", func(c C) {
				_, err := pip.StartData()
				So(err, ShouldBeError, errStateMachine)
				waitEndTransfer(pip)
			})
		})

		Convey("Given that the transfer is a recovery", func(c C) {
			pip.TransCtx.Transfer.Step = types.StepPostTasks

			Convey("When starting the data transfer", func(c C) {
				stream, err := pip.StartData()
				So(err, ShouldBeNil)

				Convey("Then it should return a dummy stream", func(c C) {
					So(stream, ShouldNotBeNil)
					So(stream, ShouldHaveSameTypeAs, &voidStream{})
				})
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			database.SimulateError(c, ctx.db)

			Convey("When starting the data transfer", func(c C) {
				_, err := pip.StartData()
				So(err, ShouldBeError, errDatabase)

				Convey("Then the transfer should end in error", func(c C) {
					waitEndTransfer(pip)
					c.So(Tester.CliErr, ShouldEqual, 1)
				})
			})
		})
	})
}

func TestPipelineEndData(t *testing.T) {
	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		trans := mkRecvTransfer(ctx, filename)

		So(ctx.db.Insert(&model.Task{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   json.RawMessage(`{}`),
		}).Run(), ShouldBeNil)

		pip := newTestPipeline(c, ctx.db, trans)
		So(pip.machine.Transition(statePreTasks), ShouldBeNil)
		So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)

		Convey("When ending the data transfer", func(c C) {
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			Convey("Then it should have closed and moved the file", func(c C) {
				_, err := os.Stat(filepath.Join(ctx.root, pip.TransCtx.Rule.
					LocalDir, filename))
				So(err, ShouldBeNil)
			})

			Convey("Then any subsequent calls to EndData should return an error", func(c C) {
				So(pip.EndData(), ShouldBeError, errStateMachine)
				waitEndTransfer(pip)
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			_, err := pip.StartData()
			So(err, ShouldBeNil)

			database.SimulateError(c, ctx.db)
			So(pip.EndData(), ShouldBeError, errDatabase)

			Convey("Then the transfer should end in error", func(c C) {
				waitEndTransfer(pip)
				c.So(Tester.CliErr, ShouldEqual, 1)
			})
		})
	})
}

func TestPipelinePostTasks(t *testing.T) {
	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		trans := mkRecvTransfer(ctx, filename)

		pip := newTestPipeline(c, ctx.db, trans)

		pip.TransCtx.PostTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainPost,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   json.RawMessage(`{}`),
		}}
		pip.TransCtx.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   json.RawMessage(`{}`),
		}}

		So(pip.machine.Transition(statePreTasks), ShouldBeNil)
		So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)
		So(pip.machine.Transition(stateWriting), ShouldBeNil)
		So(pip.machine.Transition(stateDataEnd), ShouldBeNil)
		So(pip.machine.Transition(stateDataEndDone), ShouldBeNil)

		Convey("When calling the post-tasks", func(c C) {
			So(pip.PostTasks(), ShouldBeNil)

			Convey("Then it should have executed the post-tasks", func(c C) {
				So(Tester.CliPost, ShouldEqual, 1)
			})

			Convey("Then any subsequent calls will return an error", func(c C) {
				So(pip.PostTasks(), ShouldBeError, errStateMachine)
				waitEndTransfer(pip)
			})
		})

		Convey("Given that the transfer is a recovery", func(c C) {
			trans.Step = types.StepFinalization

			Convey("When calling the post-tasks", func(c C) {
				So(pip.PostTasks(), ShouldBeNil)

				Convey("Then it should do nothing", func(c C) {
					So(Tester.CliPost, ShouldEqual, 0)
				})
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			database.SimulateError(c, ctx.db)

			Convey("When calling the post-tasks", func(c C) {
				So(pip.PostTasks(), ShouldBeError, errDatabase)

				Convey("Then the transfer should end in error", func(c C) {
					waitEndTransfer(pip)
					c.So(Tester.CliErr, ShouldEqual, 1)
				})
			})
		})

		Convey("Given that on of the tasks fails", func(c C) {
			pip.TransCtx.PostTasks = append(pip.TransCtx.PostTasks, model.Task{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.TaskErr,
				Args:   json.RawMessage(`{}`),
			})

			Convey("When calling the post-tasks", func(c C) {
				So(pip.PostTasks(), ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "post-tasks failed"))

				Convey("Then the transfer should end in error", func() {
					waitEndTransfer(pip)
					So(Tester.CliPost, ShouldEqual, 1)
					So(Tester.CliErr, ShouldEqual, 1)
				})
			})
		})
	})
}

func TestPipelineSetError(t *testing.T) {
	remErr := types.NewTransferError(types.TeUnknownRemote, "remote error")
	expErr := *types.NewTransferError(types.TeUnknownRemote,
		"Error on remote partner: remote error")

	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		trans := mkRecvTransfer(ctx, filename)
		pip := newTestPipeline(c, ctx.db, trans)

		pip.TransCtx.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   json.RawMessage(`{}`),
		}}

		Convey("Given an pre-transfer error", func(c C) {
			pip.SetError(remErr)
			waitEndTransfer(pip)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(Tester.CliErr, ShouldEqual, 1)

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.Error, ShouldResemble, expErr)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the pre-tasks", func(c C) {
			pip.TransCtx.PreTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   TaskWait,
				Args:   json.RawMessage(`{}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   json.RawMessage(`{}`),
			}}

			var taskErr error
			go func() { taskErr = pip.PreTasks() }()

			taskChan <- true
			pip.SetError(remErr)
			waitEndTransfer(pip)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(Tester.CliErr, ShouldEqual, 1)

				Convey("Then it should have interrupted the pre-tasks", func(c C) {
					So(taskErr, ShouldBeError, types.NewTransferError(
						types.TeExternalOperation, "pre-tasks failed"))
					So(Tester.CliPre, ShouldEqual, 1)
				})

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.Error, ShouldResemble, expErr)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					_, err := pip.StartData()
					So(err, ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the transfer", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)

			pip.SetError(remErr)
			waitEndTransfer(pip)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(Tester.CliErr, ShouldEqual, 1)

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.Error, ShouldResemble, expErr)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PostTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the post-tasks", func(c C) {
			pip.TransCtx.PostTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   TaskWait,
				Args:   json.RawMessage(`{}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   json.RawMessage(`{}`),
			}}

			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			var taskErr error
			go func() { taskErr = pip.PostTasks() }()

			taskChan <- true
			pip.SetError(remErr)
			waitEndTransfer(pip)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(Tester.CliErr, ShouldEqual, 1)

				Convey("Then it should have interrupted the post-tasks", func(c C) {
					So(taskErr, ShouldBeError, types.NewTransferError(
						types.TeExternalOperation, "post-tasks failed"))
					So(Tester.CliPost, ShouldEqual, 1)
				})

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.Error, ShouldResemble, expErr)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an post-transfer error", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)
			So(pip.PostTasks(), ShouldBeNil)

			pip.SetError(remErr)
			waitEndTransfer(pip)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(Tester.CliErr, ShouldEqual, 1)

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.Error, ShouldResemble, expErr)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errStateMachine)
				})
			})
		})
	})
}

func TestPipelinePause(t *testing.T) {
	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		trans := mkRecvTransfer(ctx, filename)
		pip := newTestPipeline(c, ctx.db, trans)

		pip.TransCtx.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   json.RawMessage(`{}`),
		}}

		Convey("Given an pre-transfer pause", func(c C) {
			pip.Pause()
			waitEndTransfer(pip)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(Tester.CliErr, ShouldEqual, 0)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given a pause during the pre-tasks", func(c C) {
			pip.TransCtx.PreTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   TaskWait,
				Args:   json.RawMessage(`{}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   json.RawMessage(`{}`),
			}}

			var taskErr error
			go func() { taskErr = pip.PreTasks() }()

			taskChan <- true
			pip.Pause()
			waitEndTransfer(pip)

			Convey("Then it should have interrupted the pre-tasks", func(c C) {
				So(taskErr, ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "pre-tasks failed"))
				So(Tester.CliPre, ShouldEqual, 1)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					_, err := pip.StartData()
					So(err, ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the transfer", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)

			pip.Pause()
			waitEndTransfer(pip)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(Tester.CliErr, ShouldEqual, 0)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the post-tasks", func(c C) {
			pip.TransCtx.PostTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   TaskWait,
				Args:   json.RawMessage(`{}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   json.RawMessage(`{}`),
			}}

			So(pip.PreTasks(), ShouldBeNil)
			_, e := pip.StartData()
			So(e, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			var taskErr error
			go func() { taskErr = pip.PostTasks() }()

			taskChan <- true
			pip.Pause()
			waitEndTransfer(pip)

			Convey("Then it should have interrupted the post-tasks", func(c C) {
				So(taskErr, ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "post-tasks failed"))
				So(Tester.CliPost, ShouldEqual, 1)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an post-transfer error", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)
			So(pip.PostTasks(), ShouldBeNil)

			pip.Pause()
			waitEndTransfer(pip)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(Tester.CliErr, ShouldEqual, 0)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer
					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errStateMachine)
				})
			})
		})
	})
}

func TestPipelineCancel(t *testing.T) {
	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		trans := mkRecvTransfer(ctx, filename)
		pip := newTestPipeline(c, ctx.db, trans)

		pip.TransCtx.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   json.RawMessage(`{}`),
		}}

		Convey("Given an pre-transfer cancel", func(c C) {
			pip.Cancel()
			waitEndTransfer(pip)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(Tester.CliErr, ShouldEqual, 0)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given a pause during the pre-tasks", func(c C) {
			pip.TransCtx.PreTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   TaskWait,
				Args:   json.RawMessage(`{}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   json.RawMessage(`{}`),
			}}

			var taskErr error
			go func() { taskErr = pip.PreTasks() }()

			taskChan <- true
			pip.Cancel()
			waitEndTransfer(pip)

			Convey("Then it should have interrupted the pre-tasks", func(c C) {
				So(taskErr, ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "pre-tasks failed"))
				So(Tester.CliPre, ShouldEqual, 1)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					_, err := pip.StartData()
					So(err, ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the transfer", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)

			pip.Cancel()
			waitEndTransfer(pip)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(Tester.CliErr, ShouldEqual, 0)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the post-tasks", func(c C) {
			pip.TransCtx.PostTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   TaskWait,
				Args:   json.RawMessage(`{}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   json.RawMessage(`{}`),
			}}

			So(pip.PreTasks(), ShouldBeNil)
			_, e := pip.StartData()
			So(e, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			var taskErr error
			go func() { taskErr = pip.PostTasks() }()

			taskChan <- true
			pip.Cancel()
			waitEndTransfer(pip)

			Convey("Then it should have interrupted the post-tasks", func(c C) {
				So(taskErr, ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "post-tasks failed"))
				So(Tester.CliPost, ShouldEqual, 1)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an post-transfer error", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)
			So(pip.PostTasks(), ShouldBeNil)

			pip.Cancel()
			waitEndTransfer(pip)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(Tester.CliErr, ShouldEqual, 0)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errStateMachine)
				})
			})
		})
	})
}
