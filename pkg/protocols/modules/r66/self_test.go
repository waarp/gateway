package r66

import (
	"context"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

func resetClient(client protocol.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	So(client.Stop(ctx), ShouldBeNil)
	So(client.Start(), ShouldBeNil)
}

func TestSelfPushOK(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, R66, cliConf, partConf, servConf)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			// ctx.AddFileInfo(c, internal.FollowID, float64(123))
			ctx.AddTransferInfo(c, internal.UserContent, "foobar")

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
	Convey("Given a new r66 pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, R66, cliConf, partConf, servConf)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			// ctx.AddFileInfo(c, internal.FollowID, float64(123))
			ctx.AddTransferInfo(c, internal.UserContent, "foobar")

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
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, R66, cliConf, partConf, servConf)
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
					"Task TASKERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepPreTasks)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c,
					ctx.ServerShouldHavePostTasked,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPushServerPreTasksFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, R66, cliConf, partConf, servConf)
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
					"Task TASKERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c,
					ctx.ClientShouldHavePreTasked,
					ctx.ServerShouldHavePostTasked,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPullClientPreTasksFail(t *testing.T) {
	Convey("Given a new r66 pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, R66, cliConf, partConf, servConf)
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
					"Task TASKERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepData)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c,
					ctx.ClientShouldHavePostTasked,
					ctx.ServerShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPullServerPreTasksFail(t *testing.T) {
	Convey("Given a new r66 pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, R66, cliConf, partConf, servConf)
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
					"Task TASKERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c,
					ctx.ClientShouldHavePreTasked,
					ctx.ClientShouldHavePostTasked,
					ctx.ServerShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPushClientDataFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, R66, cliConf, partConf, servConf)
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
					types.TeInternal,
					"read trace error: "+pipelinetest.ErrTestError.Error(),
					types.StepData)
				ctx.CheckServerTransferError(c,
					types.TeInternal,
					"Error on remote partner: failed to read file",
					types.StepData)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c)
			})
		})
	})
}

func TestSelfPushServerDataFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, R66, cliConf, partConf, servConf)
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
					types.TeInternal,
					"Error on remote partner: failed to write file",
					types.StepData)
				ctx.CheckServerTransferError(c,
					types.TeInternal,
					"write trace error: "+pipelinetest.ErrTestError.Error(),
					types.StepData)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c)
			})
		})
	})
}

func TestSelfPullClientDataFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, R66, cliConf, partConf, servConf)
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
					types.TeInternal,
					"write trace error: "+pipelinetest.ErrTestError.Error(),
					types.StepData)
				ctx.CheckServerTransferError(c,
					types.TeInternal,
					"Error on remote partner: failed to write file",
					types.StepData)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c)
			})
		})
	})
}

func TestSelfPullServerDataFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, R66, cliConf, partConf, servConf)
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
					types.TeInternal,
					"Error on remote partner: failed to read file",
					types.StepData)
				ctx.CheckServerTransferError(c,
					types.TeInternal,
					"read trace error: "+pipelinetest.ErrTestError.Error(),
					types.StepData)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c)
			})
		})
	})
}

func TestSelfPushClientPostTasksFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, R66, cliConf, partConf, servConf)
		ctx.StartService(c)

		Convey("Given an error during the client's post-tasks", func(c C) {
			ctx.AddClientPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ServerShouldHavePostTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Task TASKERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: post-tasks failed",
					types.StepPostTasks)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c)
			})
		})
	})
}

func TestSelfPushServerPostTasksFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, R66, cliConf, partConf, servConf)
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
					"Task TASKERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPullClientPostTasksFail(t *testing.T) {
	Convey("Given a new r66 pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, R66, cliConf, partConf, servConf)
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
					"Task TASKERR @ PULL POST[1]: task failed",
					types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: post-tasks failed",
					types.StepData)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c,
					ctx.ServerShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPullServerPostTasksFail(t *testing.T) {
	Convey("Given a new r66 pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, R66, cliConf, partConf, servConf)
		ctx.StartService(c)

		Convey("Given an error during the server's post-tasks", func(c C) {
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
					"Task TASKERR @ PULL POST[1]: task failed",
					types.StepPostTasks)

				resetClient(ctx.ClientService)
				ctx.TestRetry(c)
			})
		})
	})
}
