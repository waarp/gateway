package pipeline

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks/taskstest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewPipeline(t *testing.T) {
	logger := log.NewLogger("test_new_pipeline")

	Convey("Given a database", t, func(c C) {
		ctx := initTestDB(c)

		Convey("Given a send transfer", func(c C) {
			info := mkSendTransfer(ctx, "file")
			file := filepath.Join(ctx.root, ctx.send.LocalDir, "file")

			Convey("When initiating a new pipeline for this transfer", func(c C) {
				pip, err := newPipeline(ctx.db, logger, info)
				So(err, ShouldBeNil)

				Convey("Then it should create the corresponding pipeline", func(c C) {
					So(pip, ShouldNotBeNil)

					Convey("Then the pipeline's state machine should have been initiated", func(c C) {
						So(pip.machine.Current(), ShouldEqual, "init")
					})

					Convey("Then the transfer's paths should have been initiated", func(c C) {
						So(info.Transfer.LocalPath, ShouldEqual, file)
						So(info.Transfer.RemotePath, ShouldEqual, path.Join("/",
							ctx.send.RemoteDir, "file"))
					})
				})
			})

			Convey("Given that the file cannot be found", func(c3b C) {
				So(os.Remove(file), ShouldBeNil)

				Convey("When initiating a new pipeline for this transfer", func(c C) {
					_, err := newPipeline(ctx.db, logger, info)

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
					_, err := newPipeline(ctx.db, logger, info)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, errLimitReached)
					})
				})
			})

			Convey("Given that a database error occurs", func(c C) {
				database.SimulateError(c, ctx.db)

				Convey("When initiating a new pipeline for this transfer", func(c C) {
					_, err := newPipeline(ctx.db, logger, info)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, errDatabase)
					})
				})
			})
		})

		Convey("Given a receive transfer", func(c C) {
			filename := "file"
			info := mkRecvTransfer(ctx, filename)

			Convey("When initiating a new pipeline for this transfer", func(c C) {
				pip, err := newPipeline(ctx.db, logger, info)
				So(err, ShouldBeNil)

				Convey("Then it should create the corresponding pipeline", func(c C) {
					So(pip, ShouldNotBeNil)

					Convey("Then the pipeline's state machine should have been initiated", func(c C) {
						So(pip.machine.Current(), ShouldEqual, "init")
					})

					Convey("Then the transfer's paths should have been initiated", func(c C) {
						So(info.Transfer.LocalPath, ShouldEqual, filepath.Join(
							ctx.root, ctx.recv.LocalTmpDir, filename))
						So(info.Transfer.RemotePath, ShouldEqual, path.Join("/",
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
					_, err := newPipeline(ctx.db, logger, info)

					Convey("Then it should return an error", func(c C) {
						So(err, ShouldBeError, errLimitReached)
					})
				})
			})
		})
	})
}

func TestPipelinePreTasks(t *testing.T) {
	logger := log.NewLogger("test_pre_tasks")

	Convey("Given a transfer pipeline", t, func(c1 C) {
		ctx := initTestDB(c1)
		filename := "file"
		info := mkRecvTransfer(ctx, filename)

		taskstest.ClientCheckChannel = make(chan string, 10)
		Reset(func() { close(taskstest.ClientCheckChannel) })
		info.PreTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainPre,
			Rank:   0,
			Type:   taskstest.ClientOK,
			Args:   json.RawMessage(`{"msg":"TEST | PRE-TASKS[0]"}`),
		}}
		info.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.ClientOK,
			Args:   json.RawMessage(`{"msg":"TEST | ERROR-TASKS[0]"}`),
		}}

		pip, err := newPipeline(ctx.db, logger, info)
		So(err, ShouldBeNil)

		Convey("When calling the pre-tasks", func(c C) {
			So(pip.PreTasks(), ShouldBeNil)

			Convey("Then it should have executed the pre-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual,
					"CLIENT | recv | TEST | PRE-TASKS[0] | OK")
			})

			Convey("Then any subsequent calls will return an error", func(c C) {
				So(pip.PreTasks(), ShouldBeError, errStateMachine)
				waitEndTransfer(pip)
			})
		})

		Convey("Given that the transfer is a recovery", func(c C) {
			info.Transfer.Step = types.StepData

			Convey("When calling the pre-tasks", func(c C) {
				So(pip.PreTasks(), ShouldBeNil)
				taskstest.ClientCheckChannel <- ""

				Convey("Then it should do nothing", func(c C) {
					So(<-taskstest.ClientCheckChannel, ShouldEqual, "")
				})
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			database.SimulateError(c, ctx.db)

			Convey("When calling the pre-tasks", func(c C) {
				So(pip.PreTasks(), ShouldBeError, errDatabase)

				Convey("Then the transfer should end in error", func(c C) {
					So(<-taskstest.ClientCheckChannel, ShouldEqual,
						"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")
					waitEndTransfer(pip)
				})
			})
		})

		Convey("Given that on of the tasks fails", func(c C) {
			info.PreTasks = append(info.PreTasks, model.Task{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.ClientErr,
				Args:   json.RawMessage(`{"msg":"TEST | PRE-TASKS[1]"}`),
			})

			Convey("When calling the pre-tasks", func(c C) {
				So(pip.PreTasks(), ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "pre-tasks failed"))

				Convey("Then the transfer should end in error", func(c C) {
					So(<-taskstest.ClientCheckChannel, ShouldEqual,
						"CLIENT | recv | TEST | PRE-TASKS[0] | OK")
					So(<-taskstest.ClientCheckChannel, ShouldEqual,
						"CLIENT | recv | TEST | PRE-TASKS[1] | ERROR")
					So(<-taskstest.ClientCheckChannel, ShouldEqual,
						"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")
					waitEndTransfer(pip)
				})
			})
		})
	})
}

func TestPipelineStartData(t *testing.T) {
	logger := log.NewLogger("test_start_data")

	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		info := mkRecvTransfer(ctx, filename)

		taskstest.ClientCheckChannel = make(chan string, 10)
		Reset(func() { close(taskstest.ClientCheckChannel) })
		info.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.ClientOK,
			Args:   json.RawMessage(`{"msg":"TEST | ERROR-TASKS[0]"}`),
		}}

		pip, err := newPipeline(ctx.db, logger, info)
		So(err, ShouldBeNil)
		So(pip.machine.Transition("pre-tasks"), ShouldBeNil)
		So(pip.machine.Transition("pre-tasks done"), ShouldBeNil)

		Convey("When starting the data transfer", func(c C) {
			stream, err := pip.StartData()
			So(err, ShouldBeNil)
			Reset(func() { _ = stream.(*fileStream).file.Close() })

			Convey("Then it should return a filestream for the transfer file", func(c C) {
				So(stream, ShouldNotBeNil)
				So(stream, ShouldHaveSameTypeAs, &fileStream{})
			})

			Convey("Then it should have opened/created the file", func(c C) {
				_, err := os.Stat(filepath.Join(ctx.root, info.Rule.LocalTmpDir, filename+".part"))
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
					So(<-taskstest.ClientCheckChannel, ShouldEqual,
						"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")
					waitEndTransfer(pip)
				})
			})
		})
	})
}

func TestPipelineEndData(t *testing.T) {
	logger := log.NewLogger("test_end_data")

	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		info := mkRecvTransfer(ctx, filename)

		taskstest.ClientCheckChannel = make(chan string, 10)
		Reset(func() { close(taskstest.ClientCheckChannel) })
		info.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.ClientOK,
			Args:   json.RawMessage(`{"msg":"TEST | ERROR-TASKS[0]"}`),
		}}

		pip, err := newPipeline(ctx.db, logger, info)
		So(err, ShouldBeNil)
		So(pip.machine.Transition("pre-tasks"), ShouldBeNil)
		So(pip.machine.Transition("pre-tasks done"), ShouldBeNil)

		Convey("When ending the data transfer", func(c C) {
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			Convey("Then it should have closed and moved the file", func(c C) {
				_, err := os.Stat(filepath.Join(ctx.root, info.Rule.LocalDir, filename))
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
				So(<-taskstest.ClientCheckChannel, ShouldEqual,
					"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")
				waitEndTransfer(pip)
			})
		})
	})
}

func TestPipelinePostTasks(t *testing.T) {
	logger := log.NewLogger("test_post_tasks")

	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		info := mkRecvTransfer(ctx, filename)

		taskstest.ClientCheckChannel = make(chan string, 10)
		Reset(func() { close(taskstest.ClientCheckChannel) })
		info.PostTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainPost,
			Rank:   0,
			Type:   taskstest.ClientOK,
			Args:   json.RawMessage(`{"msg":"TEST | POST-TASKS[0]"}`),
		}}
		info.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.ClientOK,
			Args:   json.RawMessage(`{"msg":"TEST | ERROR-TASKS[0]"}`),
		}}

		pip, err := newPipeline(ctx.db, logger, info)
		So(err, ShouldBeNil)
		So(pip.machine.Transition("pre-tasks"), ShouldBeNil)
		So(pip.machine.Transition("pre-tasks done"), ShouldBeNil)
		So(pip.machine.Transition("start data"), ShouldBeNil)
		So(pip.machine.Transition("writing"), ShouldBeNil)
		So(pip.machine.Transition("end data"), ShouldBeNil)
		So(pip.machine.Transition("close"), ShouldBeNil)
		So(pip.machine.Transition("move"), ShouldBeNil)
		So(pip.machine.Transition("data ended"), ShouldBeNil)

		Convey("When calling the post-tasks", func(c C) {
			So(pip.PostTasks(), ShouldBeNil)

			Convey("Then it should have executed the post-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual,
					"CLIENT | recv | TEST | POST-TASKS[0] | OK")
			})

			Convey("Then any subsequent calls will return an error", func(c C) {
				So(pip.PostTasks(), ShouldBeError, errStateMachine)
				waitEndTransfer(pip)
			})
		})

		Convey("Given that the transfer is a recovery", func(c C) {
			info.Transfer.Step = types.StepFinalization

			Convey("When calling the post-tasks", func(c C) {
				So(pip.PostTasks(), ShouldBeNil)
				taskstest.ClientCheckChannel <- ""

				Convey("Then it should do nothing", func(c C) {
					So(<-taskstest.ClientCheckChannel, ShouldEqual, "")
				})
			})
		})

		Convey("Given that a database error occurs", func(c C) {
			database.SimulateError(c, ctx.db)

			Convey("When calling the post-tasks", func(c C) {
				So(pip.PostTasks(), ShouldBeError, errDatabase)

				Convey("Then the transfer should end in error", func(c C) {
					So(<-taskstest.ClientCheckChannel, ShouldEqual,
						"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")
					waitEndTransfer(pip)
				})
			})
		})

		Convey("Given that on of the tasks fails", func(c C) {
			info.PostTasks = append(info.PostTasks, model.Task{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.ClientErr,
				Args:   json.RawMessage(`{"msg":"TEST | POST-TASKS[1]"}`),
			})

			Convey("When calling the post-tasks", func(c C) {
				So(pip.PostTasks(), ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "post-tasks failed"))

				Convey("Then the transfer should end in error", func(c C) {
					So(<-taskstest.ClientCheckChannel, ShouldEqual,
						"CLIENT | recv | TEST | POST-TASKS[0] | OK")
					So(<-taskstest.ClientCheckChannel, ShouldEqual,
						"CLIENT | recv | TEST | POST-TASKS[1] | ERROR")
					So(<-taskstest.ClientCheckChannel, ShouldEqual,
						"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")
					waitEndTransfer(pip)
				})
			})
		})
	})
}

func TestPipelineSetError(t *testing.T) {
	logger := log.NewLogger("test_set_error")
	remErr := types.NewTransferError(types.TeUnknownRemote, "remote error")
	expErr := *types.NewTransferError(types.TeUnknownRemote,
		"Error on remote partner: remote error")

	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		info := mkRecvTransfer(ctx, filename)

		taskstest.ClientCheckChannel = make(chan string, 10)
		Reset(func() { close(taskstest.ClientCheckChannel) })
		info.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.ClientOK,
			Args:   json.RawMessage(`{"msg":"TEST | ERROR-TASKS[0]"}`),
		}}

		pip, err := newPipeline(ctx.db, logger, info)
		So(err, ShouldBeNil)

		Convey("Given an pre-transfer error", func(c C) {
			pip.SetError(remErr)
			waitEndTransfer(pip)

			Convey("Then it should have called the error-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual,
					"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusError)
					So(trans.Error, ShouldResemble, expErr)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the pre-tasks", func(c C) {
			info.PreTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | PRE-TASKS[0]","delay":"1000"}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | PRE-TASKS[1]"}`),
			}}

			errCheck := make(chan error, 1)
			go func() {
				errCheck <- pip.PreTasks()
			}()

			So(<-taskstest.ClientCheckChannel, ShouldEqual,
				"CLIENT | recv | TEST | PRE-TASKS[0] | OK")
			pip.SetError(remErr)

			Convey("Then it should have called the error-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual,
					"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")
				waitEndTransfer(pip)
				taskstest.ClientCheckChannel <- ""

				Convey("Then it should have interrupted the pre-tasks", func(c C) {
					So(<-errCheck, ShouldBeError, types.NewTransferError(
						types.TeExternalOperation, "pre-tasks failed"))
					So(<-taskstest.ClientCheckChannel, ShouldBeZeroValue)
				})

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusError)
					So(trans.Error, ShouldResemble, expErr)
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
				So(<-taskstest.ClientCheckChannel, ShouldEqual,
					"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusError)
					So(trans.Error, ShouldResemble, expErr)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PostTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the post-tasks", func(c C) {
			info.PostTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | POST-TASKS[0]","delay":"1000"}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | POST-TASKS[1]"}`),
			}}

			So(pip.PreTasks(), ShouldBeNil)
			_, err := pip.StartData()
			So(err, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			errCheck := make(chan error, 1)
			go func() { errCheck <- pip.PostTasks() }()

			So(<-taskstest.ClientCheckChannel, ShouldEqual,
				"CLIENT | recv | TEST | POST-TASKS[0] | OK")
			pip.SetError(remErr)

			Convey("Then it should have called the error-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual,
					"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")
				waitEndTransfer(pip)
				taskstest.ClientCheckChannel <- ""

				Convey("Then it should have interrupted the post-tasks", func(c C) {
					So(<-errCheck, ShouldBeError, types.NewTransferError(
						types.TeExternalOperation, "post-tasks failed"))
					So(<-taskstest.ClientCheckChannel, ShouldBeZeroValue)
				})

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusError)
					So(trans.Error, ShouldResemble, expErr)
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
				So(<-taskstest.ClientCheckChannel, ShouldEqual,
					"CLIENT | recv | TEST | ERROR-TASKS[0] | OK")

				Convey("Then the transfer should have the ERROR status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusError)
					So(trans.Error, ShouldResemble, expErr)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errStateMachine)
				})
			})
		})
	})
}

func TestPipelinePause(t *testing.T) {
	logger := log.NewLogger("test_pause")

	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		info := mkRecvTransfer(ctx, filename)

		taskstest.ClientCheckChannel = make(chan string, 10)
		Reset(func() { close(taskstest.ClientCheckChannel) })
		info.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.ClientOK,
			Args:   json.RawMessage(`{"msg":"TEST | ERROR-TASKS[0]"}`),
		}}

		pip, err := newPipeline(ctx.db, logger, info)
		So(err, ShouldBeNil)

		Convey("Given an pre-transfer pause", func(c C) {
			pip.Pause()
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given a pause during the pre-tasks", func(c C) {
			info.PreTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | PRE-TASKS[0]","delay":"1000"}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | PRE-TASKS[1]"}`),
			}}

			err := make(chan error, 1)
			go func() {
				err <- pip.PreTasks()
			}()

			So(<-taskstest.ClientCheckChannel, ShouldEqual,
				"CLIENT | recv | TEST | PRE-TASKS[0] | OK")
			pip.Pause()
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should have interrupted the pre-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")
				So(<-err, ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "pre-tasks failed"))

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusPaused)
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
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the post-tasks", func(c C) {
			info.PostTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | POST-TASKS[0]","delay":"1000"}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | POST-TASKS[1]"}`),
			}}

			So(pip.PreTasks(), ShouldBeNil)
			_, e := pip.StartData()
			So(e, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			err := make(chan error, 1)
			go func() {
				err <- pip.PostTasks()
			}()

			So(<-taskstest.ClientCheckChannel, ShouldEqual,
				"CLIENT | recv | TEST | POST-TASKS[0] | OK")
			pip.Pause()
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should have interrupted the post-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")
				So(<-err, ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "post-tasks failed"))

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusPaused)
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
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")

				Convey("Then the transfer should have the PAUSED status", func(c C) {
					var trans model.Transfer
					So(ctx.db.Get(&trans, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(trans.Status, ShouldEqual, types.StatusPaused)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errStateMachine)
				})
			})
		})
	})
}

func TestPipelineCancel(t *testing.T) {
	logger := log.NewLogger("test_cancel")

	Convey("Given a transfer pipeline", t, func(c C) {
		ctx := initTestDB(c)
		filename := "file"
		info := mkRecvTransfer(ctx, filename)

		taskstest.ClientCheckChannel = make(chan string, 10)
		Reset(func() { close(taskstest.ClientCheckChannel) })
		info.ErrTasks = model.Tasks{{
			RuleID: ctx.recv.ID,
			Chain:  model.ChainError,
			Rank:   0,
			Type:   taskstest.ClientOK,
			Args:   json.RawMessage(`{"msg":"TEST | ERROR-TASKS[0]"}`),
		}}

		pip, err := newPipeline(ctx.db, logger, info)
		So(err, ShouldBeNil)

		Convey("Given an pre-transfer cancel", func(c C) {
			pip.Cancel()
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")

				Convey("Then the transfer should have been cancelled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given a pause during the pre-tasks", func(c C) {
			info.PreTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   0,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | PRE-TASKS[0]","delay":"1000"}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPre,
				Rank:   1,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | PRE-TASKS[1]"}`),
			}}

			err := make(chan error, 1)
			go func() {
				err <- pip.PreTasks()
			}()

			So(<-taskstest.ClientCheckChannel, ShouldEqual,
				"CLIENT | recv | TEST | PRE-TASKS[0] | OK")
			pip.Cancel()
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should have interrupted the pre-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")
				So(<-err, ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "pre-tasks failed"))

				Convey("Then the transfer should have been cancelled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
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
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")

				Convey("Then the transfer should have been cancelled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.PreTasks(), ShouldBeError, errStateMachine)
				})
			})
		})

		Convey("Given an error during the post-tasks", func(c C) {
			info.PostTasks = model.Tasks{{
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   0,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | POST-TASKS[0]","delay":"1000"}`),
			}, {
				RuleID: ctx.recv.ID,
				Chain:  model.ChainPost,
				Rank:   1,
				Type:   taskstest.ClientOK,
				Args:   json.RawMessage(`{"msg":"TEST | POST-TASKS[1]"}`),
			}}

			So(pip.PreTasks(), ShouldBeNil)
			_, e := pip.StartData()
			So(e, ShouldBeNil)
			So(pip.EndData(), ShouldBeNil)

			err := make(chan error, 1)
			go func() {
				err <- pip.PostTasks()
			}()

			So(<-taskstest.ClientCheckChannel, ShouldEqual,
				"CLIENT | recv | TEST | POST-TASKS[0] | OK")
			pip.Cancel()
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should have interrupted the post-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")
				So(<-err, ShouldBeError, types.NewTransferError(
					types.TeExternalOperation, "post-tasks failed"))

				Convey("Then the transfer should have been cancelled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
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
			taskstest.ClientCheckChannel <- "END"

			Convey("Then it should NOT have called the error-tasks", func(c C) {
				So(<-taskstest.ClientCheckChannel, ShouldEqual, "END")

				Convey("Then the transfer should have been cancelled", func(c C) {
					var hist model.HistoryEntry
					So(ctx.db.Get(&hist, "id=?", info.Transfer.ID).Run(), ShouldBeNil)
					So(hist.Status, ShouldEqual, types.StatusCancelled)
				})

				Convey("Then any calls to the pipeline should return an error", func(c C) {
					So(pip.EndTransfer(), ShouldBeError, errStateMachine)
				})
			})
		})
	})
}
