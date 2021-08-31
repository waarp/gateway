package sftp

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"

	. "github.com/smartystreets/goconvey/convey"
)

func TestSelfPushOK(t *testing.T) {
	Convey("Given a new SFTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			ctx.RunTransfer(c)

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
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("When executing the transfer", func(c C) {
			ctx.RunTransfer(c)

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

/*
func TestPushClientPreError(t *testing.T) {
	Convey("Given a new SFTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
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
					types.TeConnectionReset,
					"Error on remote partner: session closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestPushServerPreError(t *testing.T) {
	Convey("Given a new SFTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in server pre-tasks", func(c C) {
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
			})
		})
	})
}

func TestPushClientPostError(t *testing.T) {
	Convey("Given a new SFTP push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in client post-tasks", func(c C) {
			ctx.AddClientPostTaskError(c)
			ctx.RunTransfer(c)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeOK(c)
				ctx.ClientPosTasksShouldBeError(c)
				ctx.CheckEndTransferError(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Post-tasks failed: Task CLIENTERR @ PUSH POST[1]: task failed",
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
		ctx := pipelinetest.InitSelfPushTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in server post-tasks", func(c C) {
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
					"Post-tasks failed: Task SERVERERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)
			})
		})
	})
}

func TestPullClientPreError(t *testing.T) {
	Convey("Given a new SFTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
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
					types.TeConnectionReset,
					"Error on remote partner: session closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestPullServerPreError(t *testing.T) {
	Convey("Given a new SFTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in server pre-tasks", func(c C) {
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
			})
		})
	})
}

func TestPullClientPostError(t *testing.T) {
	Convey("Given a new SFTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in client post-tasks", func(c C) {
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
					types.TeConnectionReset,
					"Error on remote partner: session closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestPullServerPostError(t *testing.T) {
	Convey("Given a new SFTP pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, "sftp", nil, nil)
		ctx.AddCryptos(c, makeCerts(ctx)...)
		ctx.StartService(c)

		Convey("Given that an error occurs in server post-tasks", func(c C) {
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
			})
		})
	})
}
*/
