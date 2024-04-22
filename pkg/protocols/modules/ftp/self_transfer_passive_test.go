package ftp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

// IMPORTANT NOTE ABOUT FTP TESTS:
//
// 1) Because of how FTP was designed, in the case of a pull transfer, the
// server starts sending data to the client as soon as the request has been
// validated. This means that, if an error happens on the client side after
// the request has been validated, there is no guarantee that the server will
// receive said error before finishing the transfer. This means that, in some
// cases, the server might report the transfer as finished, while the client
// will not. There is unfortunately no way to prevent this without going
// against the protocol's specification.
//
// 2) FTP has no mechanism to allow the client to send errors to the server. This
// means that the only way we have for an FTP client to signal a problem to the
// server is to use the ABOR command, which simply tells the server to cancel
// the last command. This works fine in most cases, but it does have a fairly
// big limitation: the ABOR command has no effect if the last command has
// already finished. Case in point, in the case of a pull transfer, once the
// server has finished sending data to the client, the RETR command used to
// initiate the transfer is considered finished. This means that any ABOR
// command sent by the client after that point will have no effect. For example,
// this means that if an error happens during the client's post-tasks, the
// server will not receive said error, since, from the server's perspective,
// the transfer has already finished. Once again, there is no way to prevent
// this without violating the protocol's specification.
//
// 3) The FTP server library used in our implementation ignores errors returned
// by the file's `Close()` method in the case of a pull transfer. While normally
// this should not be an issue (closing a read file should never fail), in our
// case this is problematic because we use said method to execute the server's
// post-tasks. This means that, if an error happens during the server's post-tasks,
// the error will be ignored by the library, and thus won't be reported to the
// client. We could fix this problem, but it would probably require forking
// the library, as I doubt this change would be accepted upstream.

var (
	clientConfPassive = &ClientConfig{}
	serverConfPassive = &ServerConfig{
		DisableActiveMode:  true,
		PassiveModeMinPort: 10000, PassiveModeMaxPort: 20000,
	}
	partnerConfPassive = &PartnerConfig{DisableActiveMode: true}
)

func TestPassiveSelfPushOK(t *testing.T) {
	Convey("Given a new FTP passive push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPullOK(t *testing.T) {
	Convey("Given a new FTP passive pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPushClientPreError(t *testing.T) {
	Convey("Given a new FTP passive push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPushServerPreError(t *testing.T) {
	Convey("Given a new FTP passive push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPullClientPreError(t *testing.T) {
	Convey("Given a new FTP passive pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPullServerPreError(t *testing.T) {
	Convey("Given a new FTP passive pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPushClientDataFail(t *testing.T) {
	Convey("Given a new FTP passive push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPushServerDataFail(t *testing.T) {
	Convey("Given a new FTP passive push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPullClientDataFail(t *testing.T) {
	Convey("Given a new FTP passive push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPullServerDataFail(t *testing.T) {
	Convey("Given a new FTP passive push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPushClientPostError(t *testing.T) {
	Convey("Given a new FTP passive push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPushServerPostError(t *testing.T) {
	Convey("Given a new FTP passive push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPullClientPostError(t *testing.T) {
	Convey("Given a new FTP passive pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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

func TestPassiveSelfPullServerPostError(t *testing.T) {
	Convey("Given a new FTP passive pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTP, clientConfPassive,
			partnerConfPassive, serverConfPassive)
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
