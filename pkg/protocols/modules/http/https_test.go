package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestHTTPSClient(t *testing.T) {
	setAddress := func(agent *model.RemoteAgent, url string) {
		err := agent.Address.Set(strings.TrimPrefix(url, schemeHTTPS))
		So(err, ShouldBeNil)
	}

	Convey("Given an external HTTPS server", t, func(c C) {
		src := testhelpers.NewTestReader(c)
		serv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				_, _ = io.Copy(w, src)
			case http.MethodPost:
				_, _ = io.ReadAll(r.Body)
				_ = r.Body.Close()

				w.WriteHeader(http.StatusCreated)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}))
		serverCert, err := tls.X509KeyPair([]byte(testhelpers.LocalhostCert),
			[]byte(testhelpers.LocalhostKey))
		c.So(err, ShouldBeNil)

		serv.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
		serv.StartTLS()
		Reset(serv.Close)

		Convey("Given a new HTTPS push transfer", func(c C) {
			ctx := pipelinetest.InitClientPush(c, HTTPS, nil, nil)

			setAddress(ctx.Partner, serv.URL)
			So(ctx.DB.Update(ctx.Partner).Cols("address").Run(), ShouldBeNil)

			cert := &model.Credential{
				RemoteAgentID: utils.NewNullInt64(ctx.Partner.ID),
				Name:          "partner_cert",
				Type:          auth.TLSTrustedCertificate,
				Value:         testhelpers.LocalhostCert,
			}
			ctx.AddCryptos(c, cert)

			Convey("Once the transfer has been processed", func(c C) {
				ctx.RunTransfer(c)

				Convey("Then it should have executed all the tasks in order", func(c C) {
					ctx.ClientShouldHavePreTasked(c)
					ctx.ClientShouldHavePostTasked(c)

					ctx.CheckTransferOK(c)
				})
			})
		})

		Convey("Given a new HTTPS pull transfer", func(c C) {
			ctx := pipelinetest.InitClientPull(c, HTTPS, src.Content(), nil, nil)

			setAddress(ctx.Partner, serv.URL)
			So(ctx.DB.Update(ctx.Partner).Cols("address").Run(), ShouldBeNil)

			cert := &model.Credential{
				RemoteAgentID: utils.NewNullInt64(ctx.Partner.ID),
				Name:          "partner_cert",
				Type:          auth.TLSTrustedCertificate,
				Value:         testhelpers.LocalhostCert,
			}
			ctx.AddCryptos(c, cert)

			Convey("Once the transfer has been processed", func(c C) {
				ctx.RunTransfer(c)

				Convey("Then it should have executed all the tasks in order", func(c C) {
					ctx.ClientShouldHavePreTasked(c)
					ctx.ClientShouldHavePostTasked(c)

					ctx.CheckTransferOK(c)
				})
			})
		})
	})
}

func TestHTTPSServer(t *testing.T) {
	Convey("Given a HTTPS server for push transfers", t, func(c C) {
		ctx := pipelinetest.InitServerPush(c, HTTPS, &serverConfig{})
		serverCert := &model.Credential{
			LocalAgentID: utils.NewNullInt64(ctx.Server.ID),
			Name:         "server_cert",
			Type:         auth.TLSCertificate,
			Value:        testhelpers.LocalhostCert,
			Value2:       testhelpers.LocalhostKey,
		}
		clientCert := &model.Credential{
			LocalAccountID: utils.NewNullInt64(ctx.LocAccount.ID),
			Name:           "client_cert",
			Type:           auth.TLSTrustedCertificate,
			Value:          testhelpers.ClientFooCert,
		}
		ctx.AddAuths(c, serverCert, clientCert)

		ctx.StartService(c)

		Convey("Given an external HTTPS client", func(c C) {
			rootCAs := x509.NewCertPool()
			c.So(rootCAs.AppendCertsFromPEM([]byte(testhelpers.LocalhostCert)), ShouldBeTrue)
			cert, err := tls.X509KeyPair([]byte(testhelpers.ClientFooCert), []byte(testhelpers.ClientFooKey))
			c.So(err, ShouldBeNil)

			transport := &http.Transport{TLSClientConfig: &tls.Config{
				RootCAs:      rootCAs,
				Certificates: []tls.Certificate{cert},
			}}
			client := &http.Client{Transport: transport, Timeout: time.Second}
			addr := schemeHTTPS + ctx.Server.Address.String() + "/" + ctx.Filename()

			Convey("When executing the transfer", func(c C) {
				file := testhelpers.NewTestReader(c)

				url := fmt.Sprintf("%s?%s=%s", addr, httpconst.Rule, ctx.ServerRule.Name)
				req, err := http.NewRequest(http.MethodPost, url, file) //nolint:noctx // this is a test
				So(err, ShouldBeNil)

				req.SetBasicAuth(ctx.LocAccount.Login, "")
				resp, err := client.Do(req)
				So(err, ShouldBeNil)

				defer resp.Body.Close()

				So(resp.StatusCode, ShouldEqual, http.StatusCreated)

				Convey("Then it should have executed all the tasks in order", func(c C) {
					ctx.ServerShouldHavePreTasked(c)
					ctx.ServerShouldHavePostTasked(c)

					ctx.CheckTransferOK(c)
				})
			})
		})

		Convey("Given an invalid certificate", func(c C) {
			rootCAs := x509.NewCertPool()
			c.So(rootCAs.AppendCertsFromPEM([]byte(testhelpers.LocalhostCert)), ShouldBeTrue)
			cert, err := tls.X509KeyPair([]byte(testhelpers.ClientBarCert), []byte(testhelpers.ClientBarKey))
			c.So(err, ShouldBeNil)

			transport := &http.Transport{TLSClientConfig: &tls.Config{
				RootCAs:      rootCAs,
				Certificates: []tls.Certificate{cert},
			}}
			client := &http.Client{Transport: transport, Timeout: time.Hour}
			addr := schemeHTTPS + ctx.Server.Address.String() + "/" + ctx.Filename()

			Convey("When executing the transfer", func(c C) {
				file := testhelpers.NewTestReader(c)

				url := fmt.Sprintf("%s?%s=%s", addr, httpconst.Rule, ctx.ServerRule.Name)
				req, err := http.NewRequest(http.MethodPost, url, file) //nolint:noctx // this is a test
				So(err, ShouldBeNil)

				req.SetBasicAuth(ctx.LocAccount.Login, "")
				_, err = client.Do(req)
				SoMsg("Then it should return an error", err, ShouldNotBeNil)
			})
		})
	})
}
