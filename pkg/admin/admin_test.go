package admin

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	"code.waarp.fr/waarp/gateway-ng/pkg/conf"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStart(t *testing.T) {
	Convey("Given a correct configuration", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = ":9000"
		rest := Server{
			Config: &config,
		}
		err := rest.Start()

		Convey("Then the service should start without errors", func() {
			So(err, ShouldBeNil)
		})
	})

	Convey("Given an incorrect configuration", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = ":999999"
		config.Admin.SslCert = "not_a_cert"
		config.Admin.SslKey = "not_a_key"
		rest := Server{
			Config: &config,
		}

		Convey("When starting the service", func() {
			err := rest.Start()
			Convey("Then it should produce an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestSSL(t *testing.T) {
	Convey("Given an SSL REST service", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = "localhost:9001"
		config.Admin.SslCert = "test-cert/cert.pem"
		config.Admin.SslKey = "test-cert/key.pem"
		rest := Server{
			Config: &config,
		}
		err := rest.Start()
		So(err, ShouldBeNil)

		Convey("When a status request is made", func() {
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr}
			request := &http.Request{
				Method: http.MethodGet,
				Header: http.Header{},
				URL: &url.URL{
					Scheme: "https",
					Host:   "localhost:9001",
					Path:   "/api/status",
				},
			}
			request.SetBasicAuth("admin", "adminpassword")
			response, err := client.Do(request)

			Convey("Then the service should respond OK in SSL", func() {
				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				So(response.StatusCode, ShouldEqual, http.StatusOK)
				So(response.TLS, ShouldNotBeNil)
			})
		})
	})
}

func TestStop(t *testing.T) {
	Convey("Given a REST service", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = "127.0.0.1:9002"
		rest := Server{
			Config: &config,
		}
		err := rest.Start()
		So(err, ShouldBeNil)

		Convey("When the service is stopped", func() {
			rest.Stop()

			Convey("Then the service should no longer respond to requests", func() {
				client := new(http.Client)
				response, err := client.Get("http://localhost:9002/api/status")

				So(response, ShouldBeNil)
				urlError := new(url.Error)
				So(err, ShouldHaveSameTypeAs, urlError)
			})
		})
	})
}

func TestStatus(t *testing.T) {
	Convey("Given a REST service", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = ":9100"
		rest := Server{
			Config: &config,
		}
		err := rest.Start()
		So(err, ShouldBeNil)

		Convey("When a status request is made", func() {
			client := &http.Client{}
			request := &http.Request{
				Method: http.MethodGet,
				Header: http.Header{},
				URL: &url.URL{
					Scheme: "http",
					Host:   "localhost:9100",
					Path:   "/api/status",
				},
			}
			request.SetBasicAuth("admin", "adminpassword")
			response, err := client.Do(request)

			Convey("Then the service should respond OK", func() {
				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				So(response.StatusCode, ShouldEqual, http.StatusOK)
			})
		})
	})
}
