package r66

import (
	"crypto/tls"
	"crypto/x509"
	"testing"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

//nolint:gochecknoglobals // this variable is only used for tests
var (
	servConfTLS = &config.R66ProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame", IsTLS: true}
	partConfTLS = &config.R66ProtoConfig{ServerLogin: "r66_login", ServerPassword: "sesame", IsTLS: true}
)

func TestTLS(t *testing.T) {
	Convey("Given a TLS R66 server", t, func(c C) {
		errInvalidCert := &r66.Error{
			Code:   r66.BadAuthent,
			Detail: "invalid certificate: x509: certificate signed by unknown authority",
		}

		ctx := pipelinetest.InitServerPush(c, "r66", NewService, servConfTLS)
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

				ctx.AddCryptos(c, cert(ctx.LocAccount, "loc_acc_cert", legacyR66Cert, ""))

				err := tlsConnect(ctx, legacyR66Cert, legacyR66Key)

				Convey("Then it should not return an error", func(c C) {
					So(err, ShouldBeNil)
				})
			})

			Convey("Given that the legacy certificate was NOT configured", func(c C) {
				model.IsLegacyR66CertificateAllowed = true
				defer func() { model.IsLegacyR66CertificateAllowed = false }()

				err := tlsConnect(ctx, legacyR66Cert, legacyR66Key)

				Convey("Then it should return an error", func(c C) {
					r66Err := &r66.Error{}
					So(err, testhelpers.ShouldBeErrorType, &r66Err)
					So(r66Err.Code, ShouldEqual, r66.BadAuthent)
				})
			})

			Convey("Given that the legacy certificate is NOT allowed", func(c C) {
				err := tlsConnect(ctx, legacyR66Cert, legacyR66Key)

				Convey("Then it should return an error", func(c C) {
					r66Err := &r66.Error{}
					So(err, testhelpers.ShouldBeErrorType, &r66Err)
					So(r66Err.Code, ShouldEqual, r66.BadAuthent)
				})
			})
		})
	})

	Convey("Given a TLS R66 client", t, func(c C) {
		ctx := pipelinetest.InitClientPush(c, "r66", partConfTLS)
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
			tlsServer(ctx, testhelpers.LocalhostCert, testhelpers.LocalhostKey)

			Convey("Then it should not return an error", func(c C) {
				So(getClient().connect(), ShouldBeNil)
			})
		})

		Convey("Given that the server provides a bad certificate", func(c C) {
			ctx.AddCryptos(c, cert(ctx.Partner, "partner_cert",
				testhelpers.LocalhostCert, ""))
			tlsServer(ctx, testhelpers.OtherLocalhostCert,
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

				ctx.AddCryptos(c, cert(ctx.Partner, "partner_cert", legacyR66Cert, ""))
				tlsServer(ctx, legacyR66Cert, legacyR66Key)

				Convey("Then it should not return an error", func(c C) {
					So(getClient().connect(), ShouldBeNil)
				})
			})

			Convey("Given that the legacy certificate was NOT configured", func(c C) {
				model.IsLegacyR66CertificateAllowed = true
				defer func() { model.IsLegacyR66CertificateAllowed = false }()

				tlsServer(ctx, legacyR66Cert, legacyR66Key)

				Convey("Then it should return an error", func(c C) {
					So(getClient().connect(), ShouldBeError, types.NewTransferError(
						types.TeConnection, "failed to connect to remote host"))
				})
			})

			Convey("Given that the legacy certificate is NOT allowed", func(c C) {
				tlsServer(ctx, legacyR66Cert, legacyR66Key)

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

func tlsServer(ctx *pipelinetest.ClientContext, cert, key string) {
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
		Login:    partConfTLS.ServerLogin,
		Password: []byte(partConfTLS.ServerPassword),
		Handler: authentFunc(func(auth *r66.Authent) (r66.SessionHandler, error) {
			return nil, &r66.Error{Code: r66.BadAuthent, Detail: "bad authentication"}
		}),
		Logger: ctx.Logger.AsStdLog(logging.DEBUG),
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

const legacyR66Cert = `-----BEGIN CERTIFICATE-----
MIIDdzCCAl+gAwIBAgIENnEdtTANBgkqhkiG9w0BAQsFADBsMRAwDgYDVQQGEwdV
bmtub3duMRAwDgYDVQQIEwdVbmtub3duMRAwDgYDVQQHEwdVbmtub3duMRAwDgYD
VQQKEwdVbmtub3duMRAwDgYDVQQLEwdVbmtub3duMRAwDgYDVQQDEwdVbmtub3du
MB4XDTEzMDQyOTA5NTQ1N1oXDTEzMDcyODA5NTQ1N1owbDEQMA4GA1UEBhMHVW5r
bm93bjEQMA4GA1UECBMHVW5rbm93bjEQMA4GA1UEBxMHVW5rbm93bjEQMA4GA1UE
ChMHVW5rbm93bjEQMA4GA1UECxMHVW5rbm93bjEQMA4GA1UEAxMHVW5rbm93bjCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAITDuqCocEqDFTVuHusQmb24
L5haEN4tSRbULD9NHe0SehLU+3kXrSm97m6ffIbBj95ChocvMQhCpwQsfiTNa+pT
dMTlwWN/jJEwAgfphqsDndoI+laGYJeeEhByxUaFQ608QXBUVigCdirz/T5cbkXl
jmYWA9Rar259vefE6Eubfb/wS2kBKTbP96IqOH84R2Edsl45KM6tHVXh8/VynQdZ
MVqJrMg5julPF1d0/Y/4UYoemOV+qaVrnawriZvg7+o8MLb1v7I7yok3lxpt/9TB
Rs23OElpCzNlY7Zz3f0BD+lt8ZpoeXR7rN+1RMm0VwlIrr6Sske6211AtG3+qwEC
AwEAAaMhMB8wHQYDVR0OBBYEFEVpbVKGEkgeXHQ0tN5lzLr36mVsMA0GCSqGSIb3
DQEBCwUAA4IBAQBI17aBzzRZoQP8BSLTCPIJgApjoD0DWJ8TyKb76uLADcTVqtc4
B1m7di0B6PT2dIT7+E4Ek0twOvfpUyPcgNUz0auAfBF27PJMSu2hug9HdSndvFBx
aDANCVj1H7S+QgQpxQXNs6d9mwfpyOS/SCKE7Xy26/kKbQ1oWop2gV7w/+0LHK9A
AdsPDkoTxPZFcPS7kZwg+eAov+DEOksOkWeHGypdFEsn4RqT67RheCeOZsiED4zh
ADwdXdxu2+QYlSw3p/vuriC6FEWJ4E7MPMbFTUbax3Zx7ejvs8e/DgfjZZHkwkFd
SRJz9CF60Oo+fqyp/TvfM/p3So5W5kAs9MLs
-----END CERTIFICATE-----`

const legacyR66Key = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQCEw7qgqHBKgxU1
bh7rEJm9uC+YWhDeLUkW1Cw/TR3tEnoS1Pt5F60pve5un3yGwY/eQoaHLzEIQqcE
LH4kzWvqU3TE5cFjf4yRMAIH6YarA53aCPpWhmCXnhIQcsVGhUOtPEFwVFYoAnYq
8/0+XG5F5Y5mFgPUWq9ufb3nxOhLm32/8EtpASk2z/eiKjh/OEdhHbJeOSjOrR1V
4fP1cp0HWTFaiazIOY7pTxdXdP2P+FGKHpjlfqmla52sK4mb4O/qPDC29b+yO8qJ
N5cabf/UwUbNtzhJaQszZWO2c939AQ/pbfGaaHl0e6zftUTJtFcJSK6+krJHuttd
QLRt/qsBAgMBAAECggEAE9avj4w741Z9F9PRuOxtHMVmD0z+EkUQE+I2jmr2mtNU
/HVo8mpQTNl9xHf+gqBv4BVuxsqNeB+Fl4EShGtRwd0gqL9wS27m0VcsJoSFxA4x
S0BmMAG6c02Cg4Sy59vIBh3n5WIk0au0fqyg3e2v6K/pvGVzwwqeBlOxye1JjOqD
G3aL2UefVjxPgLLE1mDoqV5ZIN2+XRXGFHJlvhA50RVDq1KQldFcbWrVTZf+Igi7
XFLR+hIOFoZmLku2BHxXBjZRJO7REV8HbT/zIHi0iFv7IK/x+66r/wL8rLiwFGeK
yA61EF0jPECgOxXURTZgTxhDwC9QPDmNSdgM1F1IBQKBgQC+Gtrc0P0fOQjehgyP
4sHhvO/2BUKGUmi5c7QawE/ja2ueefosmGRU87l3bV4x2+GrR9yX5ymv08bVtJwC
u/yncnyx6mjkMaiNXBtfrdNhKWN4GQJDF2GNur+hpXNvtBmlvulSBngbCwPrxjKa
daflVYbADyreaO7iXMUgWjJZrwKBgQCyyLkem0Vm39r44Knxq/iGx/CAD3vsGnGI
FUx0a+bxhFIKYQm9MLJtGN5Ag6kP+76snBLxJ6JSwxIBpG9JYrFLaEN49oiswcty
mfO2zIUoZ8CHnFdoR0POXDTWLTLPWCd0ogxzDsVTKT4gavA9WErvFr0twIAMqS/Y
LzbV9+BiTwKBgH2tR0+AIjbH/+MMf7WH1WElBQaCB67BQFaJ9WFSDf5s/6KvRQLC
ZGH9FnmrpgAUOyZ+xYju25JP0T1qv1DXcnpIp8L/EwT5B1Mct0QTqJCtSgMVlXdB
N874zMNSm/QW/nWitqDxgelu6NKwHrgaXDqyxfimjlKm0HZ5miB/QJYlAoGAEyid
ZeE/w7Fzdr4kmAhUvqTIagC+x+NhjTKzGbrCadlDLWeOsp54UGac0o8JW/QfT8H9
6afUpkfPMyva3SNdWnZW3KyWouS1l5dV3Z33GwhbQm0HlN4mLwQEiXsYec25lK8U
5HONw8akqLas/fXrOcnXBgMd9b1fqiwNFUrV2dMCgYAnRZ7Ig3w+pkc5dAV22SNO
4M3JJYqCiGBoGJR/w5IP1FgT+IshA/5fIBJl7s8Cg8aaWWoRYuLLjA1xTFqw+Ma9
wvThKXCE78uQIzRIyp9X6W+enbMKesrtprpsZlBHU/lZ5m/bh3EXBuCFV1Q2rrVc
5VAeza4keDveGJVWVTdTlw==
-----END PRIVATE KEY-----`
