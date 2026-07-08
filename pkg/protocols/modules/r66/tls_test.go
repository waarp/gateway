package r66

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

//nolint:gochecknoglobals // this variable is only used for tests
var (
	cliConfTLS  = &ClientConfig{}
	servConfTLS = &ServerConfig{SharedServerConfig: SharedServerConfig{ServerLogin: "r66_login"}}
	partConfTLS = &PartnerConfig{SharedPartnerConfig: SharedPartnerConfig{ServerLogin: "r66_login"}}
)

func init() {
	pipelinetest.Register(R66TLS, pipelinetest.ProtoFeatures{
		Protocol:     ModuleTLS{},
		TransID:      true,
		RuleName:     true,
		Size:         true,
		TransferInfo: true,
	})
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

				r66Cli.conns.CloseConn(r66Cli.pip)
			})

			r66conn, tErr := r66Cli.connect()
			if tErr != nil {
				return tErr
			}

			if aErr := r66Cli.authenticate(r66conn); aErr != nil {
				return aErr
			}

			return nil
		}

		remoteAccountCert := &model.Credential{
			Name:            "client_cert",
			RemoteAccountID: ctx.RemAccount.NullableID(),
			Value:           testhelpers.ClientFooCert,
			Value2:          testhelpers.ClientFooKey,
			Type:            auth.TLSCertificate,
		}

		localAccountCert := &model.Credential{
			Name:           "client_cert",
			LocalAccountID: ctx.LocAccount.NullableID(),
			Value:          testhelpers.ClientFooCert,
			Type:           auth.TLSTrustedCertificate,
		}

		localAgentCert := &model.Credential{
			Name:         "server_cert",
			LocalAgentID: ctx.Server.NullableID(),
			Value:        testhelpers.LocalhostCert,
			Value2:       testhelpers.LocalhostKey,
			Type:         auth.TLSCertificate,
		}

		remotePartnerCert := &model.Credential{
			Name:          "partner_cert",
			RemoteAgentID: ctx.Partner.NullableID(),
			Value:         testhelpers.LocalhostCert,
			Type:          auth.TLSTrustedCertificate,
		}

		Convey("Given that both provide a valid certificate", func(c C) {
			ctx.AddCreds(c, remoteAccountCert, localAccountCert, localAgentCert, remotePartnerCert)

			Convey("When connecting to the server", func() {
				SoMsg("Then it should not return an error", connect(), ShouldBeNil)
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
					connErr.Code(), ShouldEqual, types.TeBadAuthentication)
				So(connErr.Details(), ShouldContainSubstring, "tls: failed to verify certificate")
			})
		})

		Convey("Given that the client provides a legacy certificate", func() {
			compatibility.IsLegacyR66CertificateAllowed = true
			defer func() { compatibility.IsLegacyR66CertificateAllowed = false }()

			remAccLegacyCert := &model.Credential{
				Name:            "rem_acc_legacy_cert",
				RemoteAccountID: ctx.RemAccount.NullableID(),
				Type:            r66auth.AuthLegacyCertificate,
			}

			Convey("Given that the legacy certificate was expected", func(c C) {
				locAccLegacyCert := &model.Credential{
					Name:           "loc_acc_legacy_cert",
					LocalAccountID: ctx.LocAccount.NullableID(),
					Type:           r66auth.AuthLegacyCertificate,
				}

				ctx.AddCreds(c, remAccLegacyCert, locAccLegacyCert,
					localAgentCert, remotePartnerCert)

				Convey("When connecting to the server", func() {
					SoMsg("Then it should not return an error",
						connect(), ShouldBeNil)
				})
			})

			Convey("Given that the legacy certificate was expected but not provided", func() {
				compatibility.IsLegacyR66CertificateAllowed = true
				defer func() { compatibility.IsLegacyR66CertificateAllowed = false }()

				ctx.AddCreds(c, localAgentCert, remotePartnerCert)

				Convey("When connecting to the server", func() {
					SoMsg("Then it should not return an error",
						connect(), ShouldBeNil)
				})
			})

			Convey("Given that the legacy certificate was not expected", func(c C) {
				ctx.AddCreds(c, remAccLegacyCert, localAccountCert,
					localAgentCert, remotePartnerCert)

				Convey("When connecting to the server", func() {
					connErr := connect()

					SoMsg("Then it should return an error", connErr, ShouldNotBeNil)
					SoMsg("And it should be a bad certificate error",
						connErr.Code(), ShouldEqual, types.TeBadAuthentication)
					So(connErr.Details(), ShouldContainSubstring, "A: authentication failed")
				})
			})
		})

		Convey("Given that the server provides a legacy certificate", func() {
			compatibility.IsLegacyR66CertificateAllowed = true
			defer func() { compatibility.IsLegacyR66CertificateAllowed = false }()

			locAgLegacyCert := &model.Credential{
				Name:         "loc_ag_legacy_cert",
				LocalAgentID: ctx.Server.NullableID(),
				Type:         r66auth.AuthLegacyCertificate,
			}

			Convey("Given that the legacy certificate was expected", func(c C) {
				remAgLegacyCert := &model.Credential{
					Name:          "rem_ag_legacy_cert",
					RemoteAgentID: ctx.Partner.NullableID(),
					Type:          r66auth.AuthLegacyCertificate,
				}

				ctx.AddCreds(c, remoteAccountCert, localAccountCert,
					locAgLegacyCert, remAgLegacyCert)

				Convey("When connecting to the server", func() {
					SoMsg("Then it should not return an error",
						connect(), ShouldBeNil)
				})
			})

			Convey("Given that the legacy certificate was not expected", func(c C) {
				ctx.AddCreds(c, remoteAccountCert, localAccountCert,
					locAgLegacyCert, remotePartnerCert)

				Convey("When connecting to the server", func() {
					connErr := connect()

					SoMsg("Then it should return an error", connErr, ShouldNotBeNil)
					SoMsg("And it should be a bad certificate error",
						connErr.Code(), ShouldEqual, types.TeBadAuthentication)
					So(connErr.Details(), ShouldContainSubstring, "tls: failed to verify certificate")
				})
			})
		})
	})
}
