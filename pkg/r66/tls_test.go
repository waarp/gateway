package r66

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"code.waarp.fr/waarp-r66/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

//nolint:gochecknoglobals // this variable is only used for tests
var servConfTLS = &config.R66ProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame", IsTLS: true}

// var partConfTLS = &config.R66ProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame", IsTLS: true}

func TestTLS(t *testing.T) {
	Convey("Given a TLS R66 server", t, func(c C) {
		ctx := pipelinetest.InitServerPush(c, "r66", NewService, servConfTLS)
		addCerts(c, ctx)
		ctx.StartService(c)

		Convey("When connecting to the server", func() {
			err := tlsConnect(ctx, true)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given that the client provides a bad certificate", func() {
			err := tlsConnect(ctx, false)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, "remote error: tls: bad certificate")
			})
		})
	})
}

func addCerts(c C, ctx *pipelinetest.ServerContext) {
	servCert := model.Crypto{
		OwnerType:   ctx.Server.TableName(),
		OwnerID:     ctx.Server.ID,
		Name:        "loc_agent_cert",
		PrivateKey:  testhelpers.LocalhostKey,
		Certificate: testhelpers.LocalhostCert,
	}
	locAccCert := model.Crypto{
		OwnerType:   ctx.LocAccount.TableName(),
		OwnerID:     ctx.LocAccount.ID,
		Name:        "loc_acc_cert",
		Certificate: testhelpers.ClientFooCert,
	}
	ctx.AddCryptos(c, servCert, locAccCert)
}

func tlsConnect(ctx *pipelinetest.ServerContext, validCert bool) error {
	var (
		cert tls.Certificate
		err  error
	)

	if validCert {
		cert, err = tls.X509KeyPair([]byte(testhelpers.ClientFooCert), []byte(testhelpers.ClientFooKey))
	} else {
		cert, err = tls.X509KeyPair([]byte(testhelpers.ClientBarCert), []byte(testhelpers.ClientBarKey))
	}

	So(err, ShouldBeNil)

	pool := x509.NewCertPool()

	So(pool.AppendCertsFromPEM([]byte(testhelpers.LocalhostCert)), ShouldBeTrue)

	conf := &tls.Config{
		Certificates: []tls.Certificate{cert},
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
