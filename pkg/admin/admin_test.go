package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStart(t *testing.T) {

	Convey("Given a correct configuration", t, func() {
		config := &conf.ServerConfig{}
		config.Admin.Address = "localhost:0"
		config.Admin.TLSCert = "test-cert/cert.pem"
		config.Admin.TLSKey = "test-cert/key.pem"
		rest := &Server{Conf: config, Services: make(map[string]service.Service)}

		Convey("When starting the service", func() {
			err := rest.Start()

			Convey("Then it should return no error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the service should be running", func() {
				code, reason := rest.State().Get()

				So(code, ShouldEqual, service.Running)
				So(reason, ShouldBeEmpty)
			})

			Convey("When starting the service a second time", func() {
				err := rest.Start()

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then the service should still be running", func() {
					code, reason := rest.State().Get()

					So(code, ShouldEqual, service.Running)
					So(reason, ShouldBeEmpty)
				})
			})
		})
	})

	Convey("Given an invalid address", t, func() {
		config := &conf.ServerConfig{}
		config.Admin.Address = "invalid_address"
		rest := &Server{Conf: config, Services: make(map[string]service.Service)}

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
		rest := &Server{Conf: config, Services: make(map[string]service.Service)}

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
		rest := &Server{Conf: config, Services: make(map[string]service.Service)}

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
		rest := &Server{Conf: config, Services: make(map[string]service.Service)}

		Convey("When starting the service", func() {
			err := rest.Start()

			Convey("Then it should produce an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestStop(t *testing.T) {
	Convey("Given a running REST service", t, func() {
		config := &conf.ServerConfig{}
		config.Admin.Address = "localhost:0"
		rest := &Server{Conf: config, Services: make(map[string]service.Service)}

		err := rest.Start()
		So(err, ShouldBeNil)

		Convey("When the service is stopped", func() {
			addr := rest.server.Addr

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			err := rest.Stop(ctx)

			Reset(cancel)

			Convey("Then it should return no error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the service should no longer respond to requests", func() {
				client := new(http.Client)
				response, err := client.Get(addr)

				So(err, ShouldBeError)
				So(response, ShouldBeNil)
				if response != nil {
					_ = response.Body.Close()
				}
			})

			Convey("When the service is stopped a 2nd time", func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				err := rest.Stop(ctx)

				Reset(cancel)

				Convey("Then it should not do anything", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}

func TestAuthentication(t *testing.T) {
	authLogger := log.NewLogger("rest_auth_test")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	db := database.GetTestDatabase()

	SkipConvey("Given an authentication handler", t, func() {
		auth := Authentication(authLogger, db).Middleware(handler)

		Convey("Given an incoming request", func() {
			w := httptest.NewRecorder()
			r, err := http.NewRequest(http.MethodGet, "/api", nil)

			So(err, ShouldBeNil)

			Convey("Given valid credentials", func() {
				r.SetBasicAuth("admin", "admin_password")

				Convey("When sending the request", func() {
					auth.ServeHTTP(w, r)

					Convey("Then the handler should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
				})
			})

			Convey("Given an invalid login", func() {
				r.SetBasicAuth("not_admin", "admin_password")

				Convey("When sending the request", func() {
					auth.ServeHTTP(w, r)

					Convey("Then the handler should reply 'Unauthorized'", func() {
						So(w.Code, ShouldEqual, http.StatusUnauthorized)
					})
				})
			})

			Convey("Given an incorrect password", func() {
				r.SetBasicAuth("admin", "not_admin_password")

				Convey("When sending the request", func() {
					auth.ServeHTTP(w, r)

					Convey("Then the handler should reply 'Unauthorized'", func() {
						So(w.Code, ShouldEqual, http.StatusUnauthorized)
					})
				})
			})
		})
	})
}

func checkValidUpdate(db *database.Db, w *httptest.ResponseRecorder, method,
	path, id, parameter string, body []byte, handler http.Handler, old, expected interface{}) {

	Convey("When sending the request to the handler", func() {
		r, err := http.NewRequest(method, "", bytes.NewReader(body))
		So(err, ShouldBeNil)
		r = mux.SetURLVars(r, map[string]string{parameter: id})

		handler.ServeHTTP(w, r)

		Convey("Then it should reply 'Created'", func() {
			So(w.Code, ShouldEqual, http.StatusCreated)
		})

		Convey("Then the 'Location' header should contain "+
			"the URI of the updated "+parameter, func() {

			location := w.Header().Get("Location")
			So(location, ShouldEqual, path+id)
		})

		Convey("Then the response body should be empty", func() {
			So(w.Body.String(), ShouldBeEmpty)
		})

		Convey("Then the updated "+parameter+" should exist in the database", func() {
			exist, err := db.Exists(expected)

			So(err, ShouldBeNil)
			So(exist, ShouldBeTrue)
		})

		Convey("Then the old "+parameter+" should no longer exist", func() {
			exist, err := db.Exists(old)

			So(err, ShouldBeNil)
			So(exist, ShouldBeFalse)
		})
	})
}

func checkInvalidUpdate(db *database.Db, handler http.Handler, w *httptest.ResponseRecorder,
	body []byte, path, id, parameter string, old interface{}, errorMsg string) {

	Convey("When sending the request to the handler", func() {
		r, err := http.NewRequest(http.MethodPatch, path+id, bytes.NewReader(body))
		So(err, ShouldBeNil)
		r = mux.SetURLVars(r, map[string]string{parameter: id})

		handler.ServeHTTP(w, r)

		Convey("Then it should reply with a 'Bad Request' error", func() {
			So(w.Code, ShouldEqual, http.StatusBadRequest)
		})

		Convey("Then the response body should contain a message stating "+
			"the error", func() {

			So(w.Body.String(), ShouldEqual, errorMsg)
		})

		Convey("Then the old "+parameter+" should stay unchanged", func() {
			exist, err := db.Exists(old)
			So(err, ShouldBeNil)
			So(exist, ShouldBeTrue)
		})
	})
}
