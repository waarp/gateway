package r66

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

//nolint:gochecknoglobals // this variable is only used for tests
var (
	cliConfTLS  = &clientConfig{}
	servConfTLS = &serverConfig{ServerLogin: "r66_login"}
	partConfTLS = &partnerConfig{ServerLogin: "r66_login"}
)

func init() {
	pipelinetest.Protocols[R66TLS] = pipelinetest.ProtoFeatures{
		MakeClient: func(db *database.DB, cli *model.Client) services.Client {
			return &client{db: db, cli: cli}
		},
		MakeServer: func(db *database.DB, agent *model.LocalAgent) services.Server {
			return &service{db: db, agent: agent}
		},
		MakeServerConfig:  Module{}.MakeServerConfig,
		MakeClientConfig:  Module{}.MakeClientConfig,
		MakePartnerConfig: Module{}.MakePartnerConfig,
		TransID:           true,
		RuleName:          true,
	}
}

func TestTLS(t *testing.T) {
	Convey("Given an R66-TLS server & client", t, func(c C) {
		ctx := pipelinetest.InitSelfPushTransfer(c, R66TLS, cliConfTLS, partConfTLS, servConfTLS)
		ctx.AddCreds(c, serverPassword(ctx.Server), partnerPassword(ctx.Partner))
		ctx.StartService(c)

		connect := func() *pipeline.Error {
			pip, err := controller.NewClientPipeline(ctx.DB, ctx.ClientTrans)
			So(err, ShouldBeNil)

			cli, err := ctx.ClientService.InitTransfer(pip.Pip)
			So(err, ShouldBeNil)

			r66Cli, ok := cli.(*transferClient)
			So(ok, ShouldBeTrue)

			Reset(func() {
				if r66Cli.ses != nil {
					r66Cli.ses.Close()
				}

				r66Cli.conns.Done(r66Cli.pip.TransCtx.RemoteAgent.Address.String())
			})

			if tErr := r66Cli.connect(); tErr != nil {
				return tErr
			}

			return r66Cli.authenticate()
		}

		remoteAccountCert := &model.Credential{
			Name:            "client_cert",
			RemoteAccountID: utils.NewNullInt64(ctx.RemAccount.ID),
			Value:           testhelpers.ClientFooCert,
			Value2:          testhelpers.ClientFooKey,
			Type:            auth.TLSCertificate,
		}

		localAccountCert := &model.Credential{
			Name:           "client_cert",
			LocalAccountID: utils.NewNullInt64(ctx.LocAccount.ID),
			Value:          testhelpers.ClientFooCert,
			Type:           auth.TLSTrustedCertificate,
		}

		localAgentCert := &model.Credential{
			Name:         "server_cert",
			LocalAgentID: utils.NewNullInt64(ctx.Server.ID),
			Value:        testhelpers.LocalhostCert,
			Value2:       testhelpers.LocalhostKey,
			Type:         auth.TLSCertificate,
		}

		remotePartnerCert := &model.Credential{
			Name:          "partner_cert",
			RemoteAgentID: utils.NewNullInt64(ctx.Partner.ID),
			Value:         testhelpers.LocalhostCert,
			Type:          auth.TLSTrustedCertificate,
		}

		Convey("Given that both provide a valid certificate", func(c C) {
			ctx.AddCreds(c, remoteAccountCert, localAccountCert, localAgentCert, remotePartnerCert)

			Convey("When connecting to the server", func() {
				SoMsg("Then it should not return an error",
					connect(), ShouldBeNil)
			})
		})

		Convey("Given that the certificates were signed by a known authority", func(c C) {
			localhostAuthority := &model.Authority{
				Name:           "localhost_authority",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.LocalhostCert,
				ValidHosts:     []string{ctx.Server.Address.Host},
			}
			So(ctx.DB.Insert(localhostAuthority).Run(), ShouldBeNil)

			fooAuthority := &model.Authority{
				Name:           "foo_authority",
				Type:           auth.AuthorityTLS,
				PublicIdentity: testhelpers.ClientFooCert,
			}
			So(ctx.DB.Insert(fooAuthority).Run(), ShouldBeNil)

			ctx.AddCreds(c, remoteAccountCert, localAgentCert)

			Convey("When connecting to the server", func() {
				SoMsg("Then it should not return an error",
					connect(), ShouldBeNil)
			})
		})

		Convey("Given that the client provides a bad certificate", func(c C) {
			remoteAccountCert.Value = testhelpers.ClientFooCert2
			remoteAccountCert.Value2 = testhelpers.ClientFooKey2

			ctx.AddCreds(c, remoteAccountCert, localAccountCert, localAgentCert, remotePartnerCert)

			Convey("When connecting to the server", func() {
				connErr := connect()

				SoMsg("Then it should return an error", connErr, ShouldNotBeNil)
				SoMsg("And it should be a bad certificate error",
					connErr.Code(), ShouldEqual, types.TeBadAuthentication)
				So(connErr.Details(), ShouldContainSubstring, "remote error: tls: bad certificate")
			})
		})

		Convey("Given that the server provides a bad certificate", func(c C) {
			localAgentCert.Value = testhelpers.OtherLocalhostCert
			localAgentCert.Value2 = testhelpers.OtherLocalhostKey

			ctx.AddCreds(c, remoteAccountCert, localAccountCert, localAgentCert, remotePartnerCert)

			Convey("When connecting to the server", func() {
				connErr := connect()

				SoMsg("Then it should return an error", connErr, ShouldNotBeNil)
				SoMsg("And it should be a bad certificate error",
					connErr.Code(), ShouldEqual, types.TeConnection)
				So(connErr.Details(), ShouldContainSubstring, "tls: failed to verify certificate")
			})
		})

		Convey("Given that the client provides a legacy certificate", func() {
			compatibility.IsLegacyR66CertificateAllowed = true
			defer func() { compatibility.IsLegacyR66CertificateAllowed = false }()

			remoteAccountCert.Type = AuthLegacyCertificate

			Convey("Given that the legacy certificate was expected", func(c C) {
				localAccountCert.Type = AuthLegacyCertificate

				ctx.AddCreds(c, remoteAccountCert, localAccountCert,
					localAgentCert, remotePartnerCert)

				Convey("When connecting to the server", func() {
					SoMsg("Then it should not return an error",
						connect(), ShouldBeNil)
				})
			})

			Convey("Given that the legacy certificate was not expected", func(c C) {
				ctx.AddCreds(c, remoteAccountCert, localAccountCert,
					localAgentCert, remotePartnerCert)

				Convey("When connecting to the server", func() {
					connErr := connect()

					SoMsg("Then it should return an error", connErr, ShouldNotBeNil)
					SoMsg("And it should be a bad certificate error",
						connErr.Code(), ShouldEqual, types.TeBadAuthentication)
					So(connErr.Details(), ShouldContainSubstring, "A: invalid certificate")
				})
			})
		})

		Convey("Given that the server provides a legacy certificate", func() {
			compatibility.IsLegacyR66CertificateAllowed = true
			defer func() { compatibility.IsLegacyR66CertificateAllowed = false }()

			localAgentCert.Type = AuthLegacyCertificate

			Convey("Given that the legacy certificate was expected", func(c C) {
				remotePartnerCert.Type = AuthLegacyCertificate

				ctx.AddCreds(c, remoteAccountCert, localAccountCert,
					localAgentCert, remotePartnerCert)

				Convey("When connecting to the server", func() {
					SoMsg("Then it should not return an error",
						connect(), ShouldBeNil)
				})
			})

			Convey("Given that the legacy certificate was not expected", func(c C) {
				ctx.AddCreds(c, remoteAccountCert, localAccountCert,
					localAgentCert, remotePartnerCert)

				Convey("When connecting to the server", func() {
					connErr := connect()

					SoMsg("Then it should return an error", connErr, ShouldNotBeNil)
					SoMsg("And it should be a bad certificate error",
						connErr.Code(), ShouldEqual, types.TeConnection)
					So(connErr.Details(), ShouldContainSubstring, "tls: failed to verify certificate")
				})
			})
		})
	})
}
