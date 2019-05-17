package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.waarp.fr/waarp/gateway-ng/pkg/conf"
	"code.waarp.fr/waarp/gateway-ng/pkg/gatewayd"
	"code.waarp.fr/waarp/gateway-ng/pkg/log"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStart(t *testing.T) {
	Convey("Given a correct configuration", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = ":9000"
		config.Admin.SslCert = "test-cert/cert.pem"
		config.Admin.SslKey = "test-cert/key.pem"
		rest := Server{
			WG: gatewayd.NewWG(&config),
		}

		Convey("When starting the service", func() {
			err := rest.Start()
			Convey("Then the service should start without errors", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given no configuration", t, func() {
		rest := Server{}

		Convey("When starting the service", func() {
			err := rest.Start()
			Convey("Then it should produce an error ", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given an invalid address", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = "not_an_address"
		rest := Server{
			WG: gatewayd.NewWG(&config),
		}

		Convey("When starting the service", func() {
			err := rest.Start()
			Convey("Then it should produce an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given an incorrect host", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = "not.a.valid.host:9000"
		rest := Server{
			WG: gatewayd.NewWG(&config),
		}

		Convey("When starting the service", func() {
			err := rest.Start()
			Convey("Then it should produce an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given an incorrect port number", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = ":999999"
		rest := Server{
			WG: gatewayd.NewWG(&config),
		}

		Convey("When starting the service", func() {
			err := rest.Start()
			Convey("Then it should produce an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})

	Convey("Given an incorrect certificate", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = ":9000"
		config.Admin.SslCert = "not_a_cert"
		config.Admin.SslKey = "not_a_key"
		rest := Server{
			WG: gatewayd.NewWG(&config),
		}

		Convey("When starting the service", func() {
			err := rest.Start()
			Convey("Then it should produce an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestStop(t *testing.T) {
	Convey("Given a REST service", t, func() {
		config := conf.ServerConfig{}
		config.Admin.Address = ":9002"
		rest := Server{
			WG: gatewayd.NewWG(&config),
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

func TestAuthentication(t *testing.T) {
	logger := log.NewLogger()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	Convey("Given valid credentials", t, func() {
		w := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "/api", nil)
		if err != nil {
			t.Fatal(err)
		}
		r.SetBasicAuth("admin", "adminpassword")

		Convey("The function should reply OK", func() {
			Authentication(logger).Middleware(handler).ServeHTTP(w, r)

			So(w.Code, ShouldEqual, http.StatusOK)
		})
	})

	Convey("Given invalid credentials", t, func() {
		w := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "/api", nil)
		if err != nil {
			t.Fatal(err)
		}
		r.SetBasicAuth("not_admin", "not_the_password")

		Convey("The function should reply Unauthorized", func() {
			Authentication(logger).Middleware(handler).ServeHTTP(w, r)

			So(w.Code, ShouldEqual, http.StatusUnauthorized)
		})
	})
}

func TestStatus(t *testing.T) {
	Convey("Given a status request service", t, func() {
		r, err := http.NewRequest(http.MethodGet, "/api/status", nil)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()

		Convey("Then the service should reply OK", func() {
			GetStatus(w, r)

			So(w.Code, ShouldEqual, http.StatusOK)
		})
	})
}
