package admin

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	. "github.com/smartystreets/goconvey/convey"
)

func TestStart(t *testing.T) {

	Convey("Given an admin service", t, func() {
		So(ioutil.WriteFile("cert.pem", []byte(cert), 0700), ShouldBeNil)
		So(ioutil.WriteFile("key.pem", []byte(key), 0700), ShouldBeNil)

		Reset(func() {
			_ = os.Remove("cert.pem")
			_ = os.Remove("key.pem")
		})

		conf.GlobalConfig.ServerConf.Admin = conf.AdminConfig{
			Host:    "localhost",
			Port:    0,
			TLSCert: "cert.pem",
			TLSKey:  "key.pem",
		}
		server := &Server{Services: make(map[string]service.Service)}
		Reset(func() { _ = server.server.Close() })

		Convey("Given a correct configuration", func() {

			Convey("When starting the service", func() {
				err := server.Start()

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then the service should be running", func() {
					code, reason := server.State().Get()

					So(code, ShouldEqual, service.Running)
					So(reason, ShouldBeEmpty)
				})

				Convey("When starting the service a second time", func() {
					err := server.Start()

					Convey("Then it should not return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the service should still be running", func() {
						code, reason := server.State().Get()

						So(code, ShouldEqual, service.Running)
						So(reason, ShouldBeEmpty)
					})
				})
			})
		})

		Convey("Given an incorrect host", func() {
			conf.GlobalConfig.ServerConf.Admin.Host = "invalid_host"
			conf.GlobalConfig.ServerConf.Admin.Port = 0
			rest := &Server{Services: make(map[string]service.Service)}

			Convey("When starting the service", func() {
				err := rest.Start()

				Convey("Then it should produce an error", func() {
					So(err, ShouldBeError)
				})
			})
		})

		Convey("Given an incorrect port number", func(c C) {
			port := testhelpers.GetFreePort(c)
			conf.GlobalConfig.ServerConf.Admin.Host = "localhost"
			conf.GlobalConfig.ServerConf.Admin.Port = port
			rest := &Server{Services: make(map[string]service.Service)}
			l, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
			So(err, ShouldBeNil)
			Reset(func() { _ = l.Close() })

			Convey("When starting the service", func() {
				err := rest.Start()

				Convey("Then it should produce an error", func() {
					So(err, ShouldBeError)
				})
			})
		})

		Convey("Given an incorrect certificate", func() {
			conf.GlobalConfig.ServerConf.Admin.Host = "localhost"
			conf.GlobalConfig.ServerConf.Admin.Port = 0
			conf.GlobalConfig.ServerConf.Admin.TLSCert = "not_a_cert"
			conf.GlobalConfig.ServerConf.Admin.TLSKey = "not_a_key"
			rest := &Server{Services: make(map[string]service.Service)}

			Convey("When starting the service", func() {
				err := rest.Start()

				Convey("Then it should produce an error", func() {
					So(err, ShouldBeError)
				})
			})
		})
	})
}

func TestStop(t *testing.T) {
	Convey("Given a running REST service", t, func() {
		conf.GlobalConfig.ServerConf.Admin = conf.AdminConfig{Host: "localhost"}
		rest := &Server{Services: make(map[string]service.Service)}

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

	Convey("Given an authentication handler", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		auth := authentication(authLogger, db).Middleware(handler)

		Convey("Given an incoming request", func() {
			w := httptest.NewRecorder()
			r, err := http.NewRequest(http.MethodGet, "/api", nil)

			So(err, ShouldBeNil)

			Convey("Given valid credentials", func() {
				r.SetBasicAuth("admin", "admin_password")

				Convey("When sending the request", func() {
					auth.ServeHTTP(w, r)

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeBlank)
					})

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
