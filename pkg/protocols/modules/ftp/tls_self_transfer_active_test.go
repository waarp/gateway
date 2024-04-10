package ftp

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

var (
	clientTLSConfActive = &ClientConfigTLS{
		ClientConfig: ClientConfig{
			EnableActiveMode:  true,
			ActiveModeMinPort: 20000, ActiveModeMaxPort: 30000,
		},
	}
	serverTLSConfActive = &ServerConfigTLS{
		ServerConfig:   ServerConfig{DisablePassiveMode: true},
		TLSRequirement: TLSMandatory,
	}
	partnerTLSConfActive = &PartnerConfigTLS{DisableTLSSessionReuse: true}
)

func addClientCert(c C, ctx *pipelinetest.SelfContext) {
	c.So(ctx.DB.DeleteAll(&model.Credential{}).Where("local_account_id=?",
		ctx.LocAccount.ID).Run(), ShouldBeNil)
	c.So(ctx.DB.DeleteAll(&model.Credential{}).Where("remote_account_id=?",
		ctx.RemAccount.ID).Run(), ShouldBeNil)

	ctx.AddCreds(c,
		&model.Credential{
			LocalAccountID: utils.NewNullInt64(ctx.LocAccount.ID),
			Name:           "local-account-cert",
			Type:           auth.TLSTrustedCertificate,
			Value:          testhelpers.ClientFooCert,
		},
		&model.Credential{
			RemoteAccountID: utils.NewNullInt64(ctx.RemAccount.ID),
			Name:            "remote-account-cert",
			Type:            auth.TLSCertificate,
			Value:           testhelpers.ClientFooCert,
			Value2:          testhelpers.ClientFooKey,
		})
}

func TestTLSActiveSelfPushOK(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPullOK(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPushClientPreError(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPushServerPreError(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPullClientPreError(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPullServerPreError(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPushClientDataFail(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPushServerDataFail(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPullClientDataFail(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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
					"use of closed network connection",
					types.StepData)

				ctx.TestRetry(c,
					ctx.ServerShouldHavePostTasked,
					ctx.ClientShouldHavePostTasked,
				)
			})
		})
	})
}

func TestTLSActiveSelfPullServerDataFail(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPushClientPostError(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPushServerPostError(t *testing.T) {
	Convey("Given a new FTP active push transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPullClientPostError(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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

func TestTLSActiveSelfPullServerPostError(t *testing.T) {
	Convey("Given a new FTP active pull transfer", t, func(c C) {
		ctx := pipelinetest.InitSelfPullTransfer(c, FTPS, clientTLSConfActive,
			partnerTLSConfActive, serverTLSConfActive)
		addServerCert(c, ctx)
		addClientCert(c, ctx)

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
