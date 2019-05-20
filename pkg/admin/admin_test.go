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
		config.Admin.Address = "localhost:0"
		config.Admin.TlsCert = "test-cert/cert.pem"
		config.Admin.TlsKey = "test-cert/key.pem"
		rest := Server{
			WG: gatewayd.NewWG(&config),
		}

		Convey("When starting the service, even multiple times", func() {
			err1 := rest.Start()
			err2 := rest.Start()

			Convey("Then it should start without errors", func() {
				So(err1, ShouldBeNil)
				So(err2, ShouldBeNil)
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
		config.Admin.Address = "invalid_address"
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
		config.Admin.Address = "invalid_host:0"
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
		config.Admin.Address = ":0"
		config.Admin.TlsCert = "not_a_cert"
		config.Admin.TlsKey = "not_a_key"
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
		config.Admin.Address = "localhost:0"
		rest := Server{
			WG: gatewayd.NewWG(&config),
		}
		err := rest.Start()
		So(err, ShouldBeNil)

		Convey("When the service is stopped, even multiple times", func() {
			addr := rest.server.Addr
			rest.Stop()
			rest.Stop()

			Convey("Then the service should no longer respond to requests", func() {
				client := new(http.Client)
				response, err := client.Get(addr)

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
