package pipeline

import (
	"context"
	"path"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks/taskstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	errStateMachine = NewError(types.TeInternal, "internal logic error")
	errDatabase     = NewError(types.TeInternal, "database error")
)

func resetPip(pip *Pipeline) {
	if pip != nil {
		Reset(func() {
			if List.Exists(pip.TransCtx.Transfer.ID) {
				pip.doneOK()
			}
		})
	}
}

func TestNewClientPipeline(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		ctx := initTestDB(c)

		Convey("Given a send transfer", func(c C) {
			trans := mkSendTransfer(ctx, "file")
			file := mkPath(ctx.root, ctx.send.LocalDir, "file")

			transCtx, err := model.GetTransferContext(ctx.db, ctx.logger, trans)
			So(err, ShouldBeNil)

			Convey("When initiating a new pipeline for this transfer", func(c C) {
				pip, err := NewClientPipeline(ctx.db, ctx.logger, transCtx, nil)
				So(err, ShouldBeNil)
				resetPip(pip)

				Convey("Then it should create the corresponding pipeline", func(c C) {
					So(pip, ShouldNotBeNil)

					Convey("Then the pipeline's state machine should have been initiated", func(c C) {
						So(pip.machine.Current(), ShouldEqual, stateInit)
					})

					Convey("Then the transfer's paths should have been initiated", func(c C) {
						So(trans.LocalPath.String(), ShouldEqual, file.String())
						So(trans.RemotePath, ShouldEqual, path.Join(
							ctx.send.RemoteDir, "file"))
						So(trans.Filesize, ShouldEqual, len(testTransferFileContent))
					})
				})
			})

			Convey("Given that the file cannot be found", func(c C) {
				So(fs.Remove(ctx.fs, file), ShouldBeNil)

				Convey("When initiating a new pipeline for this transfer", func(c C) {
					pip, err := NewClientPipeline(ctx.db, ctx.logger, transCtx, nil)
					resetPip(pip)

					Convey("Then it should NOT return an error", func(c C) {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the transfer limit has been reached", func(c C) {
				List.countClient++
				List.SetLimits(NoLimit, 1)

				Reset(func() {
					List.countClient--
					List.SetLimits(NoLimit, NoLimit)
				})

				Convey("When initiating a new pipeline for this transfer", func(c C) {
					pip, err := NewClientPipeline(ctx.db, ctx.logger, transCtx, nil)
					resetPip(pip)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, ErrLimitReached)
					})
				})
			})
		})

		Convey("Given a receive transfer", func(c C) {
			filename := "file"
			trans := mkRecvTransfer(ctx, filename)

			transCtx, err := model.GetTransferContext(ctx.db, ctx.logger, trans)
			So(err, ShouldBeNil)

			Convey("When initiating a new pipeline for this transfer", func(c C) {
				pip, err := NewClientPipeline(ctx.db, ctx.logger, transCtx, nil)
				So(err, ShouldBeNil)
				resetPip(pip)

				Convey("Then it should create the corresponding pipeline", func(c C) {
					So(pip, ShouldNotBeNil)

					Convey("Then the pipeline's state machine should have been initiated", func(c C) {
						So(pip.machine.Current(), ShouldEqual, stateInit)
					})

					Convey("Then the transfer's paths should have been initiated", func(c C) {
						So(trans.LocalPath.String(), ShouldEqual, path.Join(
							ctx.root, ctx.recv.TmpLocalRcvDir, filename))
						So(trans.RemotePath, ShouldEqual, path.Join(
							ctx.recv.RemoteDir, filename))
					})
				})
			})

			Convey("Given that the transfer limit has been reached", func(c C) {
				List.countClient++
				List.SetLimits(NoLimit, 1)

				Reset(func() {
					List.countClient--
					List.SetLimits(NoLimit, NoLimit)
				})

				Convey("When initiating a new pipeline for this transfer", func(c C) {
					pip, err := NewClientPipeline(ctx.db, ctx.logger, transCtx, nil)
					resetPip(pip)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, ErrLimitReached)
					})
				})
			})
		})
	})
}

func TestPipelinePreTasks(t *testing.T) {
	errPreTasks := NewError(types.TeExternalOperation, "pre-tasks failed")

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
			Args:   map[string]string{},
		}}
		pip.TransCtx.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   map[string]string{},
		}}

		Convey("When calling the pre-tasks", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)

			Convey("Then it should have executed the pre-tasks", func(c C) {
				So(pip.preTasks, ShouldEqual, 1)
			})

			Convey("Then any subsequent calls will return an error", func(c C) {
				So(pip.PreTasks(), ShouldBeError, errStateMachine)
				utils.WaitChan(pip.transDone, time.Second)
			})
		})

		Convey("Given that the transfer is a recovery", func(c C) {
			trans.Step = types.StepData

			Convey("When calling the pre-tasks", func(c C) {
				So(pip.PreTasks(), ShouldBeNil)

				Convey("Then it should do nothing", func(c C) {
					So(pip.preTasks, ShouldEqual, 0)
				})
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			database.SimulateError(c, ctx.db)
			time.Sleep(testTransferUpdateInterval)

			Convey("When calling the pre-tasks", func(c C) {
				So(pip.PreTasks(), ShouldBeError, errDatabase)

				Convey("Then the transfer should end in error", func(c C) {
					utils.WaitChan(pip.transDone, time.Second)
					c.So(pip.errTasks, ShouldEqual, 1)
				})
			})
		})

		Convey("Given that on of the tasks fails", func(c C) {
			pip.TransCtx.PreTasks = append(pip.TransCtx.PreTasks, &model.Task{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.TaskErr,
				Args:   map[string]string{},
			})

			Convey("When calling the pre-tasks", func(c C) {
				So(pip.PreTasks(), ShouldBeError, errPreTasks)

				Convey("Then the transfer should end in error", func() {
					utils.WaitChan(pip.transDone, time.Second)
					So(pip.preTasks, ShouldEqual, 1)
					So(pip.errTasks, ShouldEqual, 1)
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
			Args:   map[string]string{},
		}).Run(), ShouldBeNil)

		pip := newTestPipeline(c, ctx.db, trans)
		So(pip.machine.Transition(statePreTasks), ShouldBeNil)
		So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)

		Convey("When starting the data transfer", func(c C) {
			stream, err := pip.StartData()
			So(err, ShouldBeNil)
			// Reset(func() { _ = stream.file.Close() })

			Convey("Then it should return a filestream for the transfer file", func(c C) {
				So(stream, ShouldNotBeNil)
				So(stream, ShouldHaveSameTypeAs, &FileStream{})
			})

			Convey("Then it should have opened/created the file", func(c C) {
				file := mkPath(ctx.root, pip.TransCtx.Rule.TmpLocalRcvDir,
					filename+".part")
				_, err := fs.Stat(ctx.fs, file)
				So(err, ShouldBeNil)
			})

			Convey("Then any subsequent calls to StartData should return an error", func(c C) {
				_, err := pip.StartData()
				So(err, ShouldBeError, errStateMachine)
				utils.WaitChan(pip.transDone, time.Second)
			})
		})

		Convey("Given that the transfer is a recovery", func(c C) {
			pip.TransCtx.Transfer.Step = types.StepPostTasks

			Convey("When starting the data transfer", func(c C) {
				stream, err := pip.StartData()
				So(err, ShouldBeNil)

				Convey("Then it should return a file stream", func(c C) {
					So(stream, ShouldNotBeNil)
				})
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			database.SimulateError(c, ctx.db)
			time.Sleep(testTransferUpdateInterval)

			Convey("When starting the data transfer", func(c C) {
				_, err := pip.StartData()
				So(err, ShouldBeError, errDatabase)

				Convey("Then the transfer should end in error", func(c C) {
					utils.WaitChan(pip.transDone, time.Second)
					c.So(pip.errTasks, ShouldEqual, 1)
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
			Args:   map[string]string{},
		}).Run(), ShouldBeNil)

		pip := newTestPipeline(c, ctx.db, trans)
		So(pip.machine.Transition(statePreTasks), ShouldBeNil)
		So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)

		Convey("When ending the data transfer", func(c C) {
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			Convey("Then it should have closed and moved the file", func(c C) {
				_, err := fs.Stat(ctx.fs, mkPath(ctx.root, pip.TransCtx.Rule.
					LocalDir, filename))
				So(err, ShouldBeNil)
			})

			Convey("Then any subsequent calls to EndData should return an error", func(c C) {
				So(pip.EndData(), ShouldBeError, errStateMachine)
				utils.WaitChan(pip.transDone, time.Second)
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			_, err := pip.StartData()
			So(err, ShouldBeNil)

			database.SimulateError(c, ctx.db)
			time.Sleep(testTransferUpdateInterval)
			So(pip.EndData(), ShouldBeError, errDatabase)

			Convey("Then the transfer should end in error", func(c C) {
				utils.WaitChan(pip.transDone, time.Second)
				c.So(pip.errTasks, ShouldEqual, 1)
			})
		})
	})
}

func TestPipelinePostTasks(t *testing.T) {
	errPostTasks := NewError(types.TeExternalOperation, "post-tasks failed")

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
			Args:   map[string]string{},
		}}
		pip.TransCtx.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.TaskOK,
			Args:   map[string]string{},
		}}

		So(pip.machine.Transition(statePreTasks), ShouldBeNil)
		So(pip.machine.Transition(statePreTasksDone), ShouldBeNil)
		So(pip.machine.Transition(stateDataStart), ShouldBeNil)
		So(pip.machine.Transition(stateWriting), ShouldBeNil)
		So(pip.machine.Transition(stateDataEnd), ShouldBeNil)
		So(pip.machine.Transition(stateDataEndDone), ShouldBeNil)

		Convey("When calling the post-tasks", func(c C) {
			So(pip.PostTasks(), ShouldBeNil)

			Convey("Then it should have executed the post-tasks", func(c C) {
				So(pip.postTasks, ShouldEqual, 1)
			})

			Convey("Then any subsequent calls will return an error", func(c C) {
				So(pip.PostTasks(), ShouldBeError, errStateMachine)
				utils.WaitChan(pip.transDone, time.Second)
			})
		})

		Convey("Given that the transfer is a recovery", func(c C) {
			trans.Step = types.StepFinalization

			Convey("When calling the post-tasks", func(c C) {
				So(pip.PostTasks(), ShouldBeNil)

				Convey("Then it should do nothing", func(c C) {
					So(pip.postTasks, ShouldEqual, 0)
				})
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			database.SimulateError(c, ctx.db)
			time.Sleep(testTransferUpdateInterval)

			Convey("When calling the post-tasks", func(c C) {
				So(pip.PostTasks(), ShouldBeError, errDatabase)

				Convey("Then the transfer should end in error", func(c C) {
					utils.WaitChan(pip.transDone, time.Second)
					c.So(pip.errTasks, ShouldEqual, 1)
				})
			})
		})

		Convey("Given that on of the tasks fails", func(c C) {
			pip.TransCtx.PostTasks = append(pip.TransCtx.PostTasks, &model.Task{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.TaskErr,
				Args:   map[string]string{},
			})

			Convey("When calling the post-tasks", func(c C) {
				So(pip.PostTasks(), ShouldBeError, errPostTasks)

				Convey("Then the transfer should end in error", func() {
					utils.WaitChan(pip.transDone, time.Second)
					So(pip.postTasks, ShouldEqual, 1)
					So(pip.errTasks, ShouldEqual, 1)
				})
			})
		})
	})
}

func TestPipelineSetError(t *testing.T) {
	const errCode, errMsg = types.TeUnknownRemote, "remote error"

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
			Args:   map[string]string{},
		}}

		Convey("Given an pre-transfer error", func(c C) {
			pip.SetError(errCode, errMsg)
			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(pip.errTasks, ShouldEqual, 1)

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.ErrCode, ShouldResemble, errCode)
					So(dbTrans.ErrDetails, ShouldResemble, errMsg)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, NewError(errCode, errMsg))
				})
			})
		})

		Convey("Given an error during the pre-tasks", func(c C) {
			pip.TransCtx.PreTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}}

			taskChan := make(chan bool)
			pip.Trace.OnPreTask = func(rank int8) error {
				atomic.AddUint32(&pip.preTasks, 1)
				taskChan <- true

				<-pip.Runner.Ctx.Done()

				return nil
			}

			taskErr := make(chan error)
			go func() { taskErr <- pip.PreTasks() }()

			<-taskChan
			pip.SetError(errCode, errMsg)

			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(pip.errTasks, ShouldEqual, 1)

				expErr := NewError(errCode, errMsg)

				Convey("Then it should have interrupted the pre-tasks", func(c C) {
					So(<-taskErr, ShouldBeError, expErr)
					So(pip.preTasks, ShouldEqual, 1)
				})

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.ErrCode, ShouldResemble, errCode)
					So(dbTrans.ErrDetails, ShouldResemble, errMsg)
				})

				Convey("Then any calls to the pipeline should return the same error", func(c C) {
					_, err := pip.StartData()
					So(err, ShouldBeError, expErr)
				})
			})
		})

		Convey("Given an error during the transfer", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)

			pip.SetError(errCode, errMsg)
			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(pip.errTasks, ShouldEqual, 1)

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.ErrCode, ShouldResemble, errCode)
					So(dbTrans.ErrDetails, ShouldResemble, errMsg)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PostTasks(), ShouldBeError, NewError(errCode, errMsg))
				})
			})
		})

		Convey("Given an error during the post-tasks", func(c C) {
			pip.TransCtx.PostTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}}

			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			taskChan := make(chan bool)
			pip.Trace.OnPostTask = func(rank int8) error {
				atomic.AddUint32(&pip.postTasks, 1)
				taskChan <- true

				<-pip.Runner.Ctx.Done()

				return nil
			}

			taskErr := make(chan error)
			go func() { taskErr <- pip.PostTasks() }()

			<-taskChan
			pip.SetError(errCode, errMsg)

			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(pip.errTasks, ShouldEqual, 1)

				expErr := NewError(errCode, errMsg)

				Convey("Then it should have interrupted the post-tasks", func(c C) {
					So(<-taskErr, ShouldBeError, expErr)
					So(pip.postTasks, ShouldEqual, 1)
				})

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.ErrCode, ShouldResemble, errCode)
					So(dbTrans.ErrDetails, ShouldResemble, errMsg)
				})

				Convey("Then any calls to the pipeline should return the same error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, expErr)
				})
			})
		})

		Convey("Given an post-transfer error", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)
			So(pip.PostTasks(), ShouldBeNil)

			pip.SetError(errCode, errMsg)
			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should have called the error-tasks", func(c C) {
				c.So(pip.errTasks, ShouldEqual, 1)

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusError)
					So(dbTrans.ErrCode, ShouldResemble, errCode)
					So(dbTrans.ErrDetails, ShouldResemble, errMsg)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, NewError(errCode, errMsg))
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
			Args:   map[string]string{},
		}}

		Convey("Given an pre-transfer pause", func(c C) {
			So(pip.Pause(context.Background()), ShouldBeNil)
			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(pip.errTasks, ShouldEqual, 0)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errPause)
				})
			})
		})

		Convey("Given a pause during the pre-tasks", func(c C) {
			pip.TransCtx.PreTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}}

			taskChan := make(chan bool)
			pip.Trace.OnPreTask = func(rank int8) error {
				atomic.AddUint32(&pip.preTasks, 1)
				taskChan <- true

				<-pip.Runner.Ctx.Done()

				return nil
			}

			taskErr := make(chan error)
			go func() { taskErr <- pip.PreTasks() }()

			<-taskChan
			So(pip.Pause(context.Background()), ShouldBeNil)

			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should have interrupted the pre-tasks", func(c C) {
				So(<-taskErr, ShouldBeError, errPause)
				So(pip.preTasks, ShouldEqual, 1)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					_, err := pip.StartData()
					So(err, ShouldBeError, errPause)
				})
			})
		})

		Convey("Given a pause during the transfer", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)

			So(pip.Pause(context.Background()), ShouldBeNil)
			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(pip.errTasks, ShouldEqual, 0)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errPause)
				})
			})
		})

		Convey("Given a pause during the post-tasks", func(c C) {
			pip.TransCtx.PostTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}}

			So(pip.PreTasks(), ShouldBeNil)
			_, e := pip.StartData()
			So(e, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			taskChan := make(chan bool)
			pip.Trace.OnPostTask = func(rank int8) error {
				atomic.AddUint32(&pip.postTasks, 1)
				taskChan <- true

				<-pip.Runner.Ctx.Done()

				return nil
			}

			taskErr := make(chan error)
			go func() { taskErr <- pip.PostTasks() }()

			<-taskChan
			So(pip.Pause(context.Background()), ShouldBeNil)

			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should have interrupted the post-tasks", func(c C) {
				So(<-taskErr, ShouldBeError, errPause)
				So(pip.postTasks, ShouldEqual, 1)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errPause)
				})
			})
		})

		Convey("Given an post-transfer error", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)
			So(pip.PostTasks(), ShouldBeNil)

			So(pip.Pause(context.Background()), ShouldBeNil)
			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(pip.errTasks, ShouldEqual, 0)

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var dbTrans model.Transfer

					So(ctx.db.Get(&dbTrans, "id=?", trans.ID).Run(), ShouldBeNil)
					So(dbTrans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errPause)
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
			Args:   map[string]string{},
		}}

		Convey("Given an pre-transfer cancel", func(c C) {
			So(pip.Cancel(context.Background()), ShouldBeNil)
			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(pip.errTasks, ShouldEqual, 0)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry

					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errCanceled)
				})
			})
		})

		Convey("Given a pause during the pre-tasks", func(c C) {
			pip.TransCtx.PreTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}}

			taskChan := make(chan bool)
			pip.Trace.OnPreTask = func(rank int8) error {
				atomic.AddUint32(&pip.preTasks, 1)
				taskChan <- true

				<-pip.Runner.Ctx.Done()

				return nil
			}

			taskErr := make(chan error)
			go func() { taskErr <- pip.PreTasks() }()

			<-taskChan
			So(pip.Cancel(context.Background()), ShouldBeNil)

			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should have interrupted the pre-tasks", func(c C) {
				So(<-taskErr, ShouldBeError, errCanceled)
				So(pip.preTasks, ShouldEqual, 1)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry

					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					_, err := pip.StartData()
					So(err, ShouldBeError, errCanceled)
				})
			})
		})

		Convey("Given an error during the transfer", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)

			So(pip.Cancel(context.Background()), ShouldBeNil)
			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(pip.errTasks, ShouldEqual, 0)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry

					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errCanceled)
				})
			})
		})

		Convey("Given an error during the post-tasks", func(c C) {
			pip.TransCtx.PostTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.TaskOK,
				Args:   map[string]string{},
			}}

			So(pip.PreTasks(), ShouldBeNil)
			_, e := pip.StartData()
			So(e, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			taskChan := make(chan bool)
			pip.Trace.OnPostTask = func(rank int8) error {
				atomic.AddUint32(&pip.postTasks, 1)
				taskChan <- true

				<-pip.Runner.Ctx.Done()

				return nil
			}

			taskErr := make(chan error)
			go func() { taskErr <- pip.PostTasks() }()

			<-taskChan
			So(pip.Cancel(context.Background()), ShouldBeNil)

			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should have interrupted the post-tasks", func(c C) {
				So(<-taskErr, ShouldBeError, errCanceled)
				So(pip.postTasks, ShouldEqual, 1)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry

					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errCanceled)
				})
			})
		})

		Convey("Given an post-transfer error", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)
			So(pip.PostTasks(), ShouldBeNil)

			So(pip.Cancel(context.Background()), ShouldBeNil)
			utils.WaitChan(pip.transDone, time.Second)

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(pip.errTasks, ShouldEqual, 0)

				Convey("Then the transfer should have been canceled", func(c C) {
					var hist model.HistoryEntry

					So(ctx.db.Get(&hist, "id=?", trans.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errCanceled)
				})
			})
		})
	})
}
