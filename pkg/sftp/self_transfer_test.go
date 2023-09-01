package sftp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func TestSelfPushOK(t *testing.T) {
	Convey("Given a new SFTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			ctx.RunTransfer(c, false)

			Convey("Then it should execute all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ServerShouldHavePostTasked(c)
				ctx.ClientShouldHavePostTasked(c)

				ctx.CheckEndTransferOK(c)
			})
		})
	})
}

func TestSelfPullOK(t *testing.T) {
	Convey("Given a new SFTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			ctx.RunTransfer(c, false)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ServerShouldHavePostTasked(c)
				ctx.ClientShouldHavePostTasked(c)

				ctx.CheckEndTransferOK(c)
			})
		})
	})
}

func TestPushClientPreError(t *testing.T) {
	Convey("Given a new SFTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in client pre-tasks", func(c C) {
			ctx.AddClientPreTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task TASKERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)
				ctx.CheckServerTransferError(c,
					types.TeConnectionReset,
					"Error on remote partner: session closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestPushServerPreError(t *testing.T) {
	Convey("Given a new SFTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in server pre-tasks", func(c C) {
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
					"Pre-tasks failed: Task TASKERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)
			})
		})
	})
}

func TestSelfPushClientDataFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
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
					types.TeConnectionReset,
					"Error on remote partner: session closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestSelfPushServerDataFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
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
			})
		})
	})
}

func TestSelfPullClientDataFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
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
					types.TeConnectionReset,
					"Error on remote partner: session closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestSelfPullServerDataFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
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
			})
		})
	})
}

func TestPushClientPostError(t *testing.T) {
	Convey("Given a new SFTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in client post-tasks", func(c C) {
			ctx.AddClientPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task TASKERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeConnectionReset,
					"Error on remote partner: session closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestPushServerPostError(t *testing.T) {
	Convey("Given a new SFTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in server post-tasks", func(c C) {
			ctx.AddServerPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
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
			})
		})
	})
}

func TestPullClientPreError(t *testing.T) {
	Convey("Given a new SFTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in client pre-tasks", func(c C) {
			ctx.AddClientPreTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task TASKERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)
				ctx.CheckServerTransferError(c,
					types.TeConnectionReset,
					"Error on remote partner: session closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestPullServerPreError(t *testing.T) {
	Convey("Given a new SFTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in server pre-tasks", func(c C) {
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
			})
		})
	})
}

func TestPullClientPostError(t *testing.T) {
	Convey("Given a new SFTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in client post-tasks", func(c C) {
			ctx.AddClientPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task TASKERR @ PULL POST[1]: task failed",
					types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeConnectionReset,
					"Error on remote partner: session closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestPullServerPostError(t *testing.T) {
	Convey("Given a new SFTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", NewService, nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in server post-tasks", func(c C) {
			ctx.AddServerPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
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
					"Post-tasks failed: Task TASKERR @ PULL POST[1]: task failed",
					types.StepPostTasks)
			})
		})
	})
}
