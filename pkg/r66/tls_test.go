package r66

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"code.waarp.fr/waarp-r66/r66"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	servConfTLS = &config.R66ProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame", IsTLS: true}
	//partConfTLS = &config.R66ProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame", IsTLS: true}
)

func TestTLS(t *testing.T) {

	Convey("Given a TLS R66 server", t, func(c C) {
		ctx := pipelinetest.InitServerPush(c, "r66", servConfTLS)
		addCerts(c, ctx)
		ctx.StartService(c)

		Convey("When connecting to the server", func() {
			err := tlsConnect(ctx, true)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		SkipConvey("Given that the client does not provide a certificate", func() {
			err := tlsConnect(ctx, false)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, "authentication failed")
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
		Certificate: testhelpers.ClientCert,
	}
	ctx.AddCryptos(c, servCert, locAccCert)
}

func tlsConnect(ctx *pipelinetest.ServerContext, hasCliCert bool) error {
	cert, err := tls.X509KeyPair([]byte(testhelpers.ClientCert), []byte(testhelpers.ClientKey))
	So(err, ShouldBeNil)
	pool := x509.NewCertPool()
	So(pool.AppendCertsFromPEM([]byte(testhelpers.LocalhostCert)), ShouldBeTrue)

	conf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
	}
	if hasCliCert {
		conf.Certificates = append(conf.Certificates, cert)
	}

	conn, err := tls.Dial("tcp", ctx.Server.Address, conf)
	if err != nil {
		return err
	}
	defer conn.Close()

	cli, err := r66.NewClient(conn, nil)
	So(err, ShouldBeNil)
	defer cli.Close()

	ses, err := cli.NewSession()
	So(err, ShouldBeNil)
	defer ses.Close()

	sesConf := &r66.Config{DigestAlgo: "SHA-256"}
	//TODO: remove password and authenticate with certificate only
	_, err = ses.Authent("", []byte{}, sesConf)
	return err
}
