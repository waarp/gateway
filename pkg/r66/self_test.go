package r66

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"

	. "github.com/smartystreets/goconvey/convey"
)

var (
	servConf = &config.R66ProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame"}
	partConf = &config.R66ProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame"}
)

func TestSelfPushOK(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			ctx.RunTransfer(c)

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
		ctx := pipelinetest.InitSelfPullTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			ctx.RunTransfer(c)

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
		ctx := pipelinetest.InitSelfPushTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("Given that an error occurs in client pre-tasks", func(c C) {
			ctx.AddClientPreTaskError(c)
			ctx.RunTransfer(c)

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
					types.StepPreTasks)

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
		ctx := pipelinetest.InitSelfPushTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("Given an error during the server's pre-tasks", func(c C) {
			ctx.AddServerPreTaskError(c)
			ctx.RunTransfer(c)

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
		ctx := pipelinetest.InitSelfPullTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("Given that an error occurs in client pre-tasks", func(c C) {
			ctx.AddClientPreTaskError(c)
			ctx.RunTransfer(c)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task TASKERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepData)

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
		ctx := pipelinetest.InitSelfPullTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("Given an error during the server's pre-tasks", func(c C) {
			ctx.AddServerPreTaskError(c)
			ctx.RunTransfer(c)

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
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("Given an error during the client's post-tasks", func(c C) {
			ctx.AddClientPostTaskError(c)
			ctx.RunTransfer(c)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ServerShouldHavePostTasked(c)
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
					types.StepPostTasks)

				ctx.TestRetry(c)
			})
		})
	})
}

func TestSelfPushServerPostTasksFail(t *testing.T) {
	Convey("Given a new r66 push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("Given an error during the server's post-tasks", func(c C) {
			ctx.AddServerPostTaskError(c)
			ctx.RunTransfer(c)

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
					"Post-tasks failed: Task TASKERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)

				var transs model.Transfers
				_ = ctx.DB.Select(&transs).Run()
				ctx.TestRetry(c,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestSelfPullClientPostTasksFail(t *testing.T) {
	Convey("Given a new r66 pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("Given an error during the client's post-tasks", func(c C) {
			ctx.AddClientPostTaskError(c)
			ctx.RunTransfer(c)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task TASKERR @ PULL POST[1]: task failed",
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

func TestSelfPullServerPostTasksFail(t *testing.T) {
	Convey("Given a new r66 pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "r66", partConf, servConf)
		ctx.StartService(c)

		Convey("Given an error during the server's post-tasks", func(c C) {
			ctx.AddServerPostTaskError(c)
			ctx.RunTransfer(c)

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

				ctx.TestRetry(c)
			})
		})
	})
}
