package http

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func TestSelfPushOK(t *testing.T) {
	Convey("Given a new HTTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			// ctx.AddFileInfo(c, "fi_name", "fi value")
			ctx.AddTransferInfo(c, "ti_name", "ti value")
			ctx.RunTransfer(c, false)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ServerShouldHavePostTasked(c)

				ctx.CheckEndTransferOK(c)
			})
		})
	})
}

func TestSelfPullOK(t *testing.T) {
	Convey("Given a new HTTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			// ctx.AddFileInfo(c, "fi_name", "fi value")
			ctx.AddTransferInfo(c, "ti_name", "ti value")
			ctx.RunTransfer(c, false)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ServerShouldHavePostTasked(c)

				ctx.CheckEndTransferOK(c)
			})
		})
	})
}

func TestSelfPushClientPreTasksFail(t *testing.T) {
	Convey("Given a new http push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given that an error occurs in client pre-tasks", func(c C) {
			ctx.AddClientPreTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task TASKERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepData)

				ctx.TestRetry(c,
					ctx.ServerShouldHavePostTasked,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPullServerPreTasksFail(t *testing.T) {
	Convey("Given a new http pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the server's pre-tasks", func(c C) {
			ctx.AddServerPreTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepSetup)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task TASKERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)

				ctx.TestRetry(c,
					ctx.ClientShouldHavePreTasked,
					ctx.ClientShouldHavePostTasked,
					ctx.ServerShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPushClientPostTasksFail(t *testing.T) {
	Convey("Given a new http push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the client's post-tasks", func(c C) {
			ctx.AddClientPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task TASKERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: post-tasks failed",
					types.StepData)

				ctx.TestRetry(c,
					ctx.ServerShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPushServerPostTasksFail(t *testing.T) {
	Convey("Given a new http push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the server's post-tasks", func(c C) {
			ctx.AddServerPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ClientShouldHavePreTasked(c)
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ServerShouldHavePostTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: post-tasks failed",
					types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task TASKERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)

				ctx.TestRetry(c)
			})
		})
	})
}

func TestSelfPullServerPostTasksFail(t *testing.T) {
	Convey("Given a new http pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the server's post-tasks", func(c C) {
			ctx.AddServerPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ServerShouldHavePostTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: post-tasks failed",
					types.StepData)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task TASKERR @ PULL POST[1]: task failed",
					types.StepPostTasks)

				ctx.TestRetry(c,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPushServerPreTasksFail(t *testing.T) {
	Convey("Given a new http push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the server's pre-tasks", func(c C) {
			ctx.AddServerPreTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				// Whether the client had time to execute its pre- & post-tasks
				// or not in this test is undefined. For this reason, we can
				// only check if the client's tasks have been executed after the
				// retry, when it's guaranteed that they have.
				// ctx.ClientShouldHavePreTasked(c)
				// ctx.ClientShouldHavePostTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepSetup, types.StepPreTasks, types.StepData, types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task TASKERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)
				ctx.TestRetry(c,
					ctx.ClientShouldHavePreTasked,
					ctx.ClientShouldHavePostTasked,
					ctx.ServerShouldHavePostTasked,
				)
			})
		})
	})
}

// The following test is a special case where the client & server will disagree
// on whether the transfer completed or not. Since there is no way for the client
// to inform the server when an error occurs after the request has been sent, the
// client will consider the transfer to be in error, while the server will
// consider it complete.
// As a result, the transfer also cannot be resumed, because the server cannot
// resume a finished transfer.
func TestSelfPullClientPostTasksFail(t *testing.T) {
	Convey("Given a new http pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the client's post-tasks", func(c C) {
			ctx.AddClientPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ServerShouldHavePostTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task TASKERR @ PULL POST[1]: task failed",
					types.StepPostTasks)
				ctx.CheckServerTransferOK(c)
				ctx.CheckDestFile(c)
			})
		})
	})
}

// This test is also a special case, because, like the previous one, the client
// and server might disagree on the completion of the transfer. However, unlike
// the previous test, the client & server might not always disagree. Depending
// on the timing of the error, the client may have enough time to inform the
// server (by closing the connection) that an error occurred, in which case, both
// will agree that an error occurred during the transfer (even though they might
// disagree on which error).
// For this reason, we don't check the state of the server transfer in this test,
// because that state is undefined.
func TestSelfPullClientPreTasksFail(t *testing.T) {
	Convey("Given a new http pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given that an error occurs in client pre-tasks", func(c C) {
			ctx.AddClientPreTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task TASKERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)
			})
		})
	})
}

func TestSelfPushClientDataFail(t *testing.T) {
	Convey("Given a new HTTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the data transfer", func(c C) {
			ctx.AddClientDataError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					pipelinetest.ErrTestError.Code,
					"test error: "+pipelinetest.ErrTestError.Details,
					types.StepData)
				ctx.CheckServerTransferError(c,
					pipelinetest.ErrTestError.Code,
					"Error on remote partner: "+pipelinetest.ErrTestError.Details,
					types.StepData)

				ctx.TestRetry(c)
			})
		})
	})
}

func TestSelfPushServerDataFail(t *testing.T) {
	Convey("Given a new HTTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the data transfer", func(c C) {
			ctx.AddServerDataError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					pipelinetest.ErrTestError.Code,
					"Error on remote partner: "+pipelinetest.ErrTestError.Details,
					types.StepData, types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					pipelinetest.ErrTestError.Code,
					"test error: "+pipelinetest.ErrTestError.Details,
					types.StepData)

				ctx.TestRetry(c)
			})
		})
	})
}

// Same as TestSelfPullClientPreTasksFail.
func TestSelfPullClientDataFail(t *testing.T) {
	Convey("Given a new HTTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the data transfer", func(c C) {
			ctx.AddClientDataError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					pipelinetest.ErrTestError.Code,
					"test error: "+pipelinetest.ErrTestError.Details,
					types.StepData)
			})
		})
	})
}

func TestSelfPullServerDataFail(t *testing.T) {
	Convey("Given a new HTTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "http", NewService, nil, nil)
		ctx.StartService(c)

		Convey("Given an error during the data transfer", func(c C) {
			ctx.AddServerDataError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					pipelinetest.ErrTestError.Code,
					"Error on remote partner: "+pipelinetest.ErrTestError.Details,
					types.StepData)
				ctx.CheckServerTransferError(c,
					pipelinetest.ErrTestError.Code,
					"test error: "+pipelinetest.ErrTestError.Details,
					types.StepData)

				ctx.TestRetry(c)
			})
		})
	})
}
