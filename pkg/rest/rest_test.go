package rest

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
		config.Rest.Port = "9000"
		rest := Service{
			Config: &config,
		}
		err := rest.Start()

		Convey("Then the service should start without errors", func() {
			So(err, ShouldBeNil)
		})
	})

	Convey("Given an incorrect configuration", t, func() {
		config := conf.ServerConfig{}
		config.Rest.Port = "999999"
		rest := Service{
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
		config.Rest.Port = "9001"
		config.Rest.SslCert = "test-cert/cert.pem"
		config.Rest.SslKey = "test-cert/key.pem"
		rest := Service{
			Config: &config,
		}
		err := rest.Start()
		So(err, ShouldBeNil)

		Convey("When a status request is made", func() {
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr}
			response, err := client.Get("https://localhost:9001/status")

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
		config.Rest.Port = "9002"
		rest := Service{
			Config: &config,
		}
		err := rest.Start()
		So(err, ShouldBeNil)

		Convey("When the service is stopped", func() {
			rest.Stop()

			Convey("Then the service should no longer respond to requests", func() {
				client := new(http.Client)
				response, err := client.Get("http://localhost:9002/status")

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
		config.Rest.Port = "9100"
		rest := Service{
			Config: &config,
		}
		err := rest.Start()
		So(err, ShouldBeNil)

		Convey("When a status request is made", func() {
			client := new(http.Client)
			response, err := client.Get("http://localhost:9100/status")

			Convey("Then the service should respond OK", func() {
				So(err, ShouldBeNil)
				So(response, ShouldNotBeNil)
				So(response.StatusCode, ShouldEqual, http.StatusOK)
			})
		})
	})
}