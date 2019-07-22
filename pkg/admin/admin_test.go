package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/smartystreets/assertions"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	. "github.com/smartystreets/goconvey/convey"
)

var testLogger = log.NewLogger("admin-test")

func TestStart(t *testing.T) {
	Convey("Given a correct configuration", t, func() {
		config := &conf.ServerConfig{}
		config.Admin.Address = "localhost:0"
		config.Admin.TLSCert = "test-cert/cert.pem"
		config.Admin.TLSKey = "test-cert/key.pem"
		rest := &Server{Conf: config, Services: make(map[string]service.Servicer)}

		Convey("When starting the service, even multiple times", func() {
			err1 := rest.Start()
			err2 := rest.Start()

			Convey("Then it should start without errors", func() {
				So(err1, ShouldBeNil)
				So(err2, ShouldBeNil)
			})
		})
	})

	Convey("Given an invalid address", t, func() {
		config := &conf.ServerConfig{}
		config.Admin.Address = "invalid_address"
		rest := &Server{Conf: config, Services: make(map[string]service.Servicer)}

		Convey("When starting the service", func() {
			err := rest.Start()

			Convey("Then it should produce an error", func() {
				So(err, ShouldBeError)
			})
		})
	})

	Convey("Given an incorrect host", t, func() {
		config := &conf.ServerConfig{}
		config.Admin.Address = "invalid_host:0"
		rest := &Server{Conf: config, Services: make(map[string]service.Servicer)}

		Convey("When starting the service", func() {
			err := rest.Start()

			Convey("Then it should produce an error", func() {
				So(err, ShouldBeError)
			})
		})
	})

	Convey("Given an incorrect port number", t, func() {
		config := &conf.ServerConfig{}
		config.Admin.Address = ":999999"
		rest := &Server{Conf: config, Services: make(map[string]service.Servicer)}

		Convey("When starting the service", func() {
			err := rest.Start()

			Convey("Then it should produce an error", func() {
				So(err, ShouldBeError)
			})
		})
	})

	Convey("Given an incorrect certificate", t, func() {
		config := &conf.ServerConfig{}
		config.Admin.Address = ":0"
		config.Admin.TLSCert = "not_a_cert"
		config.Admin.TLSKey = "not_a_key"
		rest := &Server{Conf: config, Services: make(map[string]service.Servicer)}

		Convey("When starting the service", func() {
			err := rest.Start()

			Convey("Then it should produce an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestStop(t *testing.T) {
	Convey("Given a REST service", t, func() {
		config := &conf.ServerConfig{}
		config.Admin.Address = "localhost:0"
		rest := &Server{Conf: config, Services: make(map[string]service.Servicer)}

		err := rest.Start()
		if err != nil {
			t.Fatal(err)
		}

		Convey("When the service is stopped, even multiple times", func() {
			addr := rest.server.Addr

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			err1 := rest.Stop(ctx)
			// FIXME: Should be `defer cancel()`?
			cancel()

			ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
			err2 := rest.Stop(ctx)
			// FIXME: Should be `defer cancel()`?
			cancel()

			So(err1, ShouldBeNil)
			So(err2, ShouldBeNil)

			Convey("Then the service should no longer respond to requests", func() {
				client := new(http.Client)
				response, err := client.Get(addr)

				urlError := new(url.Error)
				So(err, ShouldHaveSameTypeAs, urlError)
				So(response, ShouldBeNil)
				if response != nil {
					_ = response.Body.Close()
				}
			})
		})
	})
}

func TestAuthentication(t *testing.T) {
	logger := log.NewLogger(ServiceName)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	db := database.GetTestDatabase()
	defer func() {
		_ = db.Stop(context.Background())
	}()

	Convey("Given valid credentials", t, func() {
		w := httptest.NewRecorder()
		r, err := http.NewRequest(http.MethodGet, "/api", nil)
		if err != nil {
			t.Fatal(err)
		}
		r.SetBasicAuth("admin", "admin_password")

		Convey("The function should reply OK", func() {
			Authentication(logger, db).Middleware(handler).ServeHTTP(w, r)

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

		Convey("The function should reply '401 - Unauthorized'", func() {
			Authentication(logger, db).Middleware(handler).ServeHTTP(w, r)

			So(w.Code, ShouldEqual, http.StatusUnauthorized)
		})
	})
}

func TestStatus(t *testing.T) {
	Convey("Given a status handling function", t, func() {
		var services = make(map[string]service.Servicer)
		services["Admin"] = &Server{}

		Convey("When a request is passed to it", func() {
			r, err := http.NewRequest(http.MethodGet, "/api/status", nil)
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()

			Convey("Then the function should reply OK with a JSON body", func() {
				getStatus(testLogger, services).ServeHTTP(w, r)
				contentType := w.Header().Get("Content-Type")

				So(w.Code, ShouldEqual, http.StatusOK)
				So(contentType, ShouldEqual, "application/json")
				So(json.Valid(w.Body.Bytes()), ShouldBeTrue)

				code, reason := services["Admin"].State().Get()
				admin := Status{
					State:  code.Name(),
					Reason: reason,
				}
				statuses := map[string]Status{"Admin": admin}
				expected, err := json.Marshal(statuses)
				So(err, ShouldBeNil)

				So(w.Body.String(), assertions.ShouldEqualTrimSpace, string(expected))
			})
		})
	})
}
