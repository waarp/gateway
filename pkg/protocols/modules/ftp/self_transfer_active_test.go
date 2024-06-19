package ftp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

var (
	clientConfActive  = &ClientConfig{EnableActiveMode: true}
	serverConfActive  = &ServerConfig{DisablePassiveMode: true}
	partnerConfActive = &PartnerConfig{DisableEPSV: true}
)

func TestActiveSelfPushOK(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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

func TestActiveSelfPullOK(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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

func TestActiveSelfPushClientPreError(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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
					"Task TASKERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)
				ctx.CheckServerTransferError(c,
					types.TeConnectionReset,
					"data connection closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestActiveSelfPushServerPreError(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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
					"Task TASKERR @ PUSH PRE[1]: task failed",
					types.StepPreTasks)
			})
		})
	})
}

func TestActiveSelfPullClientPreError(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
		ctx.StartService(c)

		Convey("Given that an error occurs in client pre-tasks", func(c C) {
			ctx.AddClientPreTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				// The server's end status is undefined in this scenario.
				// See entry n°1 of the FTP test notes.
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Task TASKERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)

				if ctx.GetServerErrTaskNb() == 0 {
					ctx.ServerShouldHavePostTasked(c)
					ctx.CheckServerTransferOK(c)
				} else {
					ctx.CheckServerTransferError(c,
						types.TeConnectionReset,
						"data connection closed unexpectedly",
						types.StepData)
				}
			})
		})
	})
}

func TestActiveSelfPullServerPreError(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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
					"Task TASKERR @ PULL PRE[1]: task failed",
					types.StepPreTasks)
			})
		})
	})
}

func TestActiveSelfPushClientDataFail(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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
					types.TeConnectionReset,
					"data connection closed unexpectedly",
					types.StepData)

				ctx.TestRetry(c,
					ctx.ServerShouldHavePostTasked,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestActiveSelfPushServerDataFail(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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
					types.TeConnectionReset,
					"data connection closed unexpectedly",
					types.StepData)
				ctx.CheckServerTransferError(c,
					types.TeInternal,
					"write trace error: "+pipelinetest.ErrTestError.Error(),
					types.StepData)

				ctx.TestRetry(c,
					ctx.ServerShouldHavePostTasked,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestActiveSelfPullClientDataFail(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
		ctx.StartService(c)

		Convey("Given an error during the data transfer", func(c C) {
			ctx.AddClientDataError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)
				// ctx.ServerShouldHavePostTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeInternal,
					"write trace error: "+pipelinetest.ErrTestError.Error(),
					types.StepData)
				ctx.CheckServerTransferError(c,
					types.TeConnectionReset,
					"data connection closed unexpectedly",
					types.StepData)

				if ctx.GetServerTransfer(c).Progress > 0 {
					ctx.TestRetry(c,
						ctx.ServerShouldHavePostTasked,
						ctx.ClientShouldHavePostTasked,
					)
				}
			})
		})
	})
}

func TestActiveSelfPullServerDataFail(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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
					types.TeConnectionReset,
					"data connection closed unexpectedly",
					types.StepData)
				ctx.CheckServerTransferError(c,
					types.TeInternal,
					"read trace error: "+pipelinetest.ErrTestError.Error(),
					types.StepData)

				ctx.TestRetry(c,
					ctx.ServerShouldHavePostTasked,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestActiveSelfPushClientPostError(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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
					"Task TASKERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)
				ctx.CheckServerTransferError(c,
					types.TeConnectionReset,
					"data connection closed unexpectedly",
					types.StepData)
			})
		})
	})
}

func TestActiveSelfPushServerPostError(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
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
					"Task TASKERR @ PUSH POST[1]: task failed",
					types.StepPostTasks)
			})
		})
	})
}

func TestActiveSelfPullClientPostError(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
		ctx.StartService(c)

		Convey("Given that an error occurs in client post-tasks", func(c C) {
			ctx.AddClientPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				// Even though it shouldn't, the server transfer end normally
				// and without error. See entry n°2 of the FTP test notes.
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ServerShouldHavePostTasked(c)
				ctx.ClientShouldHaveErrorTasked(c)

				ctx.CheckClientTransferError(c,
					types.TeExternalOperation,
					"Task TASKERR @ PULL POST[1]: task failed",
					types.StepPostTasks)
				ctx.CheckServerTransferOK(c)
			})
		})
	})
}

func TestActiveSelfPullServerPostError(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfActive,
			partnerConfActive, serverConfActive)
		ctx.StartService(c)

		Convey("Given that an error occurs in server post-tasks", func(c C) {
			ctx.AddServerPostTaskError(c)
			ctx.RunTransfer(c, true)

			Convey("Then it should have executed all the tasks in order", func(c C) {
				// Even though it shouldn't, the client transfer end normally
				// and without error. See entry n°3 of the FTP test notes.
				ctx.ServerShouldHavePreTasked(c)
				ctx.ClientShouldHavePreTasked(c)
				ctx.ClientShouldHavePostTasked(c)
				ctx.ServerShouldHavePostTasked(c)
				ctx.ServerShouldHaveErrorTasked(c)

				ctx.CheckClientTransferOK(c)
				ctx.CheckServerTransferError(c,
					types.TeExternalOperation,
					"Task TASKERR @ PULL POST[1]: task failed",
					types.StepPostTasks)
			})
		})
	})
}
