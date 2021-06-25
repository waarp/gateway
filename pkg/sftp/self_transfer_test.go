package sftp

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeOK(c)
				ctx.ServerPosTasksShouldBeOK(c)
				ctx.ClientPosTasksShouldBeOK(c)

				ctx.CheckTransfersOK(c)
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
				ctx.ServerPreTasksShouldBeOK(c)
				ctx.ClientPreTasksShouldBeOK(c)
				ctx.ServerPosTasksShouldBeOK(c)
				ctx.ClientPosTasksShouldBeOK(c)

				ctx.CheckTransfersOK(c)
			})
		})
	})
}

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

				cTrans := &model.Transfer{
					Step:       types.StepPreTasks,
					Progress:   0,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Pre-tasks failed: Task CLIENTERR @ PUSH PRE[1]: task failed"),
				}
				sTrans := &model.Transfer{
					Step:       types.StepData,
					Filesize:   model.UnknownSize,
					Progress:   0,
					TaskNumber: 0,
					Error: *types.NewTransferError(types.TeConnectionReset,
						"Error on remote partner: session closed unexpectedly"),
				}

				ctx.CheckTransfersError(c, cTrans, sTrans)
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

				cTrans := &model.Transfer{
					Step:       types.StepSetup,
					Progress:   0,
					TaskNumber: 0,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Error on remote partner: pre-tasks failed"),
				}
				sTrans := &model.Transfer{
					Step:       types.StepPreTasks,
					Filesize:   model.UnknownSize,
					Progress:   0,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Pre-tasks failed: Task SERVERERR @ PUSH PRE[1]: task failed"),
				}

				ctx.CheckTransfersError(c, cTrans, sTrans)
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

				cTrans := &model.Transfer{
					Step:       types.StepPostTasks,
					Progress:   pipelinetest.ProgressComplete,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Post-tasks failed: Task CLIENTERR @ PUSH POST[1]: task failed"),
				}
				sTrans := &model.Transfer{
					Step:       types.StepData,
					Filesize:   model.UnknownSize,
					Progress:   pipelinetest.ProgressComplete,
					TaskNumber: 0,
					Error: *types.NewTransferError(types.TeConnectionReset,
						"Error on remote partner: session closed unexpectedly"),
				}

				ctx.CheckTransfersError(c, cTrans, sTrans)
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

				cTrans := &model.Transfer{
					Step:       types.StepPostTasks,
					Progress:   pipelinetest.ProgressComplete,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Error on remote partner: post-tasks failed"),
				}
				sTrans := &model.Transfer{
					Step:       types.StepPostTasks,
					Filesize:   pipelinetest.TestFileSize,
					Progress:   pipelinetest.ProgressComplete,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Post-tasks failed: Task SERVERERR @ PUSH POST[1]: task failed"),
				}

				ctx.CheckTransfersError(c, cTrans, sTrans)
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

				cTrans := &model.Transfer{
					Step:       types.StepPreTasks,
					Filesize:   model.UnknownSize,
					Progress:   0,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Pre-tasks failed: Task CLIENTERR @ PULL PRE[1]: task failed"),
				}
				sTrans := &model.Transfer{
					Step:       types.StepData,
					Progress:   0,
					TaskNumber: 0,
					Error: *types.NewTransferError(types.TeConnectionReset,
						"Error on remote partner: session closed unexpectedly"),
				}

				ctx.CheckTransfersError(c, cTrans, sTrans)
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

				cTrans := &model.Transfer{
					Step:       types.StepSetup,
					Filesize:   model.UnknownSize,
					Progress:   0,
					TaskNumber: 0,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Error on remote partner: pre-tasks failed"),
				}
				sTrans := &model.Transfer{
					Step:       types.StepPreTasks,
					Progress:   0,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Pre-tasks failed: Task SERVERERR @ PULL PRE[1]: task failed"),
				}

				ctx.CheckTransfersError(c, cTrans, sTrans)
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

				cTrans := &model.Transfer{
					Step:       types.StepPostTasks,
					Filesize:   pipelinetest.TestFileSize,
					Progress:   pipelinetest.ProgressComplete,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Post-tasks failed: Task CLIENTERR @ PULL POST[1]: task failed"),
				}
				sTrans := &model.Transfer{
					Step:       types.StepData,
					Progress:   pipelinetest.ProgressComplete,
					TaskNumber: 0,
					Error: *types.NewTransferError(types.TeConnectionReset,
						"Error on remote partner: session closed unexpectedly"),
				}

				ctx.CheckTransfersError(c, cTrans, sTrans)
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

				cTrans := &model.Transfer{
					Step:       types.StepPostTasks,
					Filesize:   pipelinetest.TestFileSize,
					Progress:   pipelinetest.ProgressComplete,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Error on remote partner: post-tasks failed"),
				}
				sTrans := &model.Transfer{
					Step:       types.StepPostTasks,
					Progress:   pipelinetest.ProgressComplete,
					TaskNumber: 1,
					Error: *types.NewTransferError(types.TeExternalOperation,
						"Post-tasks failed: Task SERVERERR @ PULL POST[1]: task failed"),
				}

				ctx.CheckTransfersError(c, cTrans, sTrans)
			})
		})
	})
}
