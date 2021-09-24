package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/http/httpconst"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/pipelinetest"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHTTPSClient(t *testing.T) {
	Convey("Given an external HTTPS server", t, func(c C) {
		src := testhelpers.NewTestReader(c)
		serv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				_, _ = io.Copy(w, src)
			case http.MethodPost:
				_, _ = ioutil.ReadAll(r.Body)
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

		addr := strings.TrimLeft(serv.URL, "https://")

		Convey("Given a new HTTPS push transfer", func(c C) {
			ctx := pipelinetest.InitClientPush(c, "https", &config.HTTPProtoConfig{})

			ctx.Partner.Address = addr
			So(ctx.DB.Update(ctx.Partner).Cols("address").Run(), ShouldBeNil)
			cert := model.Crypto{
				OwnerType:   ctx.Partner.TableName(),
				OwnerID:     ctx.Partner.ID,
				Name:        "partner_cert",
				Certificate: testhelpers.LocalhostCert,
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
			ctx := pipelinetest.InitClientPull(c, "https", src.Content(), &config.HTTPProtoConfig{})

			ctx.Partner.Address = addr
			So(ctx.DB.Update(ctx.Partner).Cols("address").Run(), ShouldBeNil)
			cert := model.Crypto{
				OwnerType:   ctx.Partner.TableName(),
				OwnerID:     ctx.Partner.ID,
				Name:        "partner_cert",
				Certificate: testhelpers.LocalhostCert,
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
		ctx := pipelinetest.InitServerPush(c, "https", &config.HTTPProtoConfig{})
		serverCert := model.Crypto{
			OwnerType:   ctx.Server.TableName(),
			OwnerID:     ctx.Server.ID,
			Name:        "server_cert",
			PrivateKey:  testhelpers.LocalhostKey,
			Certificate: testhelpers.LocalhostCert,
		}
		clientCert := model.Crypto{
			OwnerType:   ctx.LocAccount.TableName(),
			OwnerID:     ctx.LocAccount.ID,
			Name:        "client_cert",
			Certificate: testhelpers.ClientFooCert,
		}
		ctx.AddCryptos(c, serverCert, clientCert)

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
			addr := "https://" + ctx.Server.Address + "/" + ctx.Filename()

			Convey("When executing the transfer", func(c C) {
				file := testhelpers.NewTestReader(c)

				url := fmt.Sprintf("%s?%s=%s", addr, httpconst.Rule, ctx.ServerRule.Name)
				req, err := http.NewRequest(http.MethodPost, url, file)
				So(err, ShouldBeNil)

				req.SetBasicAuth(ctx.LocAccount.Login, "")
				resp, err := client.Do(req)
				So(err, ShouldBeNil)
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
			client := &http.Client{Transport: transport, Timeout: time.Second}
			addr := "https://" + ctx.Server.Address + "/" + ctx.Filename()

			Convey("When executing the transfer", func(c C) {
				file := testhelpers.NewTestReader(c)

				url := fmt.Sprintf("%s?%s=%s", addr, httpconst.Rule, ctx.ServerRule.Name)
				req, err := http.NewRequest(http.MethodPost, url, file)
				So(err, ShouldBeNil)

				req.SetBasicAuth(ctx.LocAccount.Login, "")
				_, err = client.Do(req)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})
			})
		})
	})
}
