package r66

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"code.waarp.fr/lib/log"
	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestTLS(t *testing.T) {
	Convey("Given a TLS R66 server", t, func(c C) {
		errInvalidCert := &r66.Error{
			Code:   r66.BadAuthent,
			Detail: "invalid certificate: x509: certificate signed by unknown authority",
		}

		ctx := pipelinetest.InitServerPush(c, ProtocolR66TLS, NewService, servConf)
		ctx.AddCryptos(c, cert(ctx.Server, "server_cert",
			testhelpers.LocalhostCert, testhelpers.LocalhostKey))
		ctx.StartService(c)

		Convey("When connecting to the server", func(c C) {
			ctx.AddCryptos(c, cert(ctx.LocAccount, "loc_acc_cert", testhelpers.ClientFooCert, ""))
			err := tlsConnect(ctx, testhelpers.ClientFooCert, testhelpers.ClientFooKey)

			Convey("Then it should not return an error", func(c C) {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given that the client provides a bad certificate", func(c C) {
			err := tlsConnect(ctx, testhelpers.ClientFooCert2, testhelpers.ClientFooKey2)

			FocusConvey("Then it should return an error", func(c C) {
				So(err, ShouldBeError, errInvalidCert)
			})
		})

		Convey("Given that the client provides the legacy certificate", func(c C) {
			Convey("Given that the legacy certificate is allowed", func(c C) {
				model.IsLegacyR66CertificateAllowed = true
				defer func() { model.IsLegacyR66CertificateAllowed = false }()

				ctx.AddCryptos(c, cert(ctx.LocAccount, "loc_acc_cert", testhelpers.LegacyR66Cert, ""))

				err := tlsConnect(ctx, testhelpers.LegacyR66Cert, testhelpers.LegacyR66Key)

				Convey("Then it should not return an error", func(c C) {
					So(err, ShouldBeNil)
				})
			})

			Convey("Given that the legacy certificate was NOT configured", func(c C) {
				model.IsLegacyR66CertificateAllowed = true
				defer func() { model.IsLegacyR66CertificateAllowed = false }()

				err := tlsConnect(ctx, testhelpers.LegacyR66Cert, testhelpers.LegacyR66Key)

				Convey("Then it should return an error", func(c C) {
					r66Err := &r66.Error{}
					So(err, testhelpers.ShouldBeErrorType, &r66Err)
					So(r66Err.Code, ShouldEqual, r66.BadAuthent)
				})
			})

			Convey("Given that the legacy certificate is NOT allowed", func(c C) {
				err := tlsConnect(ctx, testhelpers.LegacyR66Cert, testhelpers.LegacyR66Key)

				Convey("Then it should return an error", func(c C) {
					r66Err := &r66.Error{}
					So(err, testhelpers.ShouldBeErrorType, &r66Err)
					So(r66Err.Code, ShouldEqual, r66.BadAuthent)
				})
			})
		})
	})

	Convey("Given a TLS R66 client", t, func(c C) {
		ctx := pipelinetest.InitClientPush(c, ProtocolR66TLS, partConf)
		getClient := func() *client {
			pip, err := pipeline.NewClientPipeline(ctx.DB, ctx.ClientTrans)
			c.So(err, ShouldBeNil)

			cli, err := newClient(pip.Pip)
			So(err, ShouldBeError)

			return cli
		}

		Convey("When connecting to the server", func(c C) {
			ctx.AddCryptos(c, cert(ctx.Partner, "partner_cert",
				testhelpers.LocalhostCert, ""))
			tlsServer(c, ctx, testhelpers.LocalhostCert, testhelpers.LocalhostKey)

			Convey("Then it should not return an error", func(c C) {
				So(getClient().connect(), ShouldBeNil)
			})
		})

		Convey("Given that the server provides a bad certificate", func(c C) {
			ctx.AddCryptos(c, cert(ctx.Partner, "partner_cert",
				testhelpers.LocalhostCert, ""))
			tlsServer(c, ctx, testhelpers.OtherLocalhostCert,
				testhelpers.OtherLocalhostKey)

			Convey("Then it should return an error", func(c C) {
				So(getClient().connect(), ShouldBeError, types.NewTransferError(
					types.TeConnection, "failed to connect to remote host"))
			})
		})

		Convey("Given that the client provides the legacy certificate", func(c C) {
			Convey("Given that the legacy certificate is allowed", func(c C) {
				model.IsLegacyR66CertificateAllowed = true
				defer func() { model.IsLegacyR66CertificateAllowed = false }()

				ctx.AddCryptos(c, cert(ctx.Partner, "partner_cert", testhelpers.LegacyR66Cert, ""))
				tlsServer(c, ctx, testhelpers.LegacyR66Cert, testhelpers.LegacyR66Key)

				Convey("Then it should not return an error", func(c C) {
					So(getClient().connect(), ShouldBeNil)
				})
			})

			Convey("Given that the legacy certificate was NOT configured", func(c C) {
				model.IsLegacyR66CertificateAllowed = true
				defer func() { model.IsLegacyR66CertificateAllowed = false }()

				tlsServer(c, ctx, testhelpers.LegacyR66Cert, testhelpers.LegacyR66Key)

				Convey("Then it should return an error", func(c C) {
					So(getClient().connect(), ShouldBeError, types.NewTransferError(
						types.TeConnection, "failed to connect to remote host"))
				})
			})

			Convey("Given that the legacy certificate is NOT allowed", func(c C) {
				tlsServer(c, ctx, testhelpers.LegacyR66Cert, testhelpers.LegacyR66Key)

				Convey("Then it should return an error", func(c C) {
					So(getClient().connect(), ShouldBeError, types.NewTransferError(
						types.TeConnection, "failed to connect to remote host"))
				})
			})
		})
	})
}

type cryptoOwner interface {
	database.Table
	database.Identifier
}

func cert(owner cryptoOwner, name, cert, key string) model.Crypto {
	return model.Crypto{
		OwnerType:   owner.TableName(),
		OwnerID:     owner.GetID(),
		Name:        name,
		PrivateKey:  types.CypherText(key),
		Certificate: cert,
	}
}

func tlsServer(c C, ctx *pipelinetest.ClientContext, cert, key string) {
	certificate, err := tls.X509KeyPair([]byte(cert), []byte(key))
	So(err, ShouldBeNil)

	pool := x509.NewCertPool()

	So(pool.AppendCertsFromPEM([]byte(testhelpers.ClientFooCert)), ShouldBeTrue)

	conf := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      pool,
	}

	serv := r66.Server{
		Login:    partConf.ServerLogin,
		Password: []byte(partConf.ServerPassword),
		Handler: authentFunc(func(auth *r66.Authent) (r66.SessionHandler, error) {
			return nil, &r66.Error{Code: r66.BadAuthent, Detail: "bad authentication"}
		}),
		Logger: testhelpers.TestLogger(c, "http_trace").AsStdLogger(log.LevelTrace),
	}

	go serv.ListenAndServeTLS(ctx.Partner.Address, conf)
}

type authentFunc func(*r66.Authent) (r66.SessionHandler, error)

func (a authentFunc) ValidAuth(auth *r66.Authent) (r66.SessionHandler, error) { return a(auth) }

func tlsConnect(ctx *pipelinetest.ServerContext, cert, key string) error {
	certificate, err := tls.X509KeyPair([]byte(cert), []byte(key))
	So(err, ShouldBeNil)

	pool := x509.NewCertPool()

	So(pool.AppendCertsFromPEM([]byte(testhelpers.LocalhostCert)), ShouldBeTrue)

	conf := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      pool,
	}

	conn, err := tls.Dial("tcp", ctx.Server.Address, conf)
	if err != nil {
		return err //nolint:wrapcheck // this is a test
	}
	defer conn.Close()

	cli, err := r66.NewClient(conn, nil)
	So(err, ShouldBeNil)

	defer cli.Close()

	ses, err := cli.NewSession()
	So(err, ShouldBeNil)

	defer ses.Close()

	sesConf := &r66.Config{DigestAlgo: "SHA-256"}
	_, err = ses.Authent(ctx.LocAccount.Login, []byte(pipelinetest.TestPassword), sesConf)

	return err //nolint:wrapcheck // this is a test
}
