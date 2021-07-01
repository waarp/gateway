package r66

import (
	"testing"

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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeOK(c)
				ctx.ClientPosTasksShouldBeOK(c)
				ctx.ServerPosTasksShouldBeOK(c)

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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeOK(c)
				ctx.ClientPosTasksShouldBeOK(c)
				ctx.ServerPosTasksShouldBeOK(c)

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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeError(c)
				ctx.CheckEndTransferError(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task CLIENTERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepPreTasks)

				ctx.TestRetry(c,
					ctx.ServerPosTasksShouldBeOK,
					ctx.ClientPosTasksShouldBeOK,
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
				ctx.ServerPreTasksShouldBeError(c)
				ctx.CheckEndTransferError(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepSetup)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task SERVERERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)

				ctx.TestRetry(c,
					ctx.ClientPreTasksShouldBeOK,
					ctx.ServerPosTasksShouldBeOK,
					ctx.ClientPosTasksShouldBeOK,
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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeError(c)
				ctx.CheckEndTransferError(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task CLIENTERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepData)

				ctx.TestRetry(c,
					ctx.ClientPosTasksShouldBeOK,
					ctx.ServerPosTasksShouldBeOK,
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
				ctx.ServerPreTasksShouldBeError(c)
				ctx.CheckEndTransferError(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: pre-tasks failed",
					types.StepSetup)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Pre-tasks failed: Task SERVERERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)

				ctx.TestRetry(c,
					ctx.ClientPreTasksShouldBeOK,
					ctx.ClientPosTasksShouldBeOK,
					ctx.ServerPosTasksShouldBeOK,
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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeOK(c)
				ctx.ServerPosTasksShouldBeOK(c)
				ctx.ClientPosTasksShouldBeError(c)
				ctx.CheckEndTransferError(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task CLIENTERR @ PUSH POST[1]: task failed",
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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeOK(c)
				ctx.ServerPosTasksShouldBeError(c)
				ctx.CheckEndTransferError(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: post-tasks failed",
					types.StepData)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task SERVERERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)

				ctx.TestRetry(c,
					ctx.ClientPosTasksShouldBeOK,
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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeOK(c)
				ctx.ClientPosTasksShouldBeError(c)
				ctx.CheckEndTransferError(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task CLIENTERR @ PULL POST[1]: task failed",
					types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: post-tasks failed",
					types.StepData)

				ctx.TestRetry(c,
					ctx.ServerPosTasksShouldBeOK,
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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeOK(c)
				ctx.ClientPosTasksShouldBeOK(c)
				ctx.ServerPosTasksShouldBeError(c)
				ctx.CheckEndTransferError(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Error on remote partner: post-tasks failed",
					types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task SERVERERR @ PULL POST[1]: task failed",
					types.StepPostTasks)

				ctx.TestRetry(c)
			})
		})
	})
}
