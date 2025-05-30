package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestStart(t *testing.T) {
	Convey("Given an admin service", t, func(c C) {
		certFile := testhelpers.TempFile(c, "rest_cert_*.pem")
		keyFile := testhelpers.TempFile(c, "rest_key_*.pem")

		So(os.WriteFile(certFile, []byte(cert), 0o600), ShouldBeNil)
		So(os.WriteFile(keyFile, []byte(key), 0o600), ShouldBeNil)

		conf.GlobalConfig.Admin = conf.AdminConfig{
			Host:    "localhost",
			Port:    0,
			TLSCert: certFile,
			TLSKey:  keyFile,
		}
		server := &Server{}

		Reset(func() { _ = server.server.Close() })

		Convey("Given a correct configuration", func() {
			Convey("When starting the service", func() {
				So(server.Start(), ShouldBeNil)

				Convey("Then it should have started a TLS listener", func() {
					So(server.server.TLSConfig, ShouldNotBeNil)
				})

				Convey("Then the service should be running", func() {
					code, reason := server.State()

					So(code, ShouldEqual, utils.StateRunning)
					So(reason, ShouldBeEmpty)
				})

				Convey("When starting the service a second time", func() {
					So(server.Start(), ShouldBeError, utils.ErrAlreadyRunning)

					Convey("Then the service should still be running", func() {
						code, reason := server.State()

						So(code, ShouldEqual, utils.StateRunning)
						So(reason, ShouldBeEmpty)
					})
				})
			})
		})

		Convey("Given a key file with a passphrase", func() {
			keyFilePass := testhelpers.TempFile(c, "rest_key_passphrase_*.pem")
			So(os.WriteFile(keyFilePass, []byte(keyWithPassphrase), 0o600), ShouldBeNil)

			conf.GlobalConfig.Admin.TLSKey = keyFilePass
			conf.GlobalConfig.Admin.TLSPassphrase = keyPassphrase

			So(server.Start(), ShouldBeNil)

			Convey("Then it should have started a TLS listener", func() {
				So(server.server.TLSConfig, ShouldNotBeNil)
			})

			Convey("Then the service should be running", func() {
				code, reason := server.State()

				So(code, ShouldEqual, utils.StateRunning)
				So(reason, ShouldBeEmpty)
			})
		})

		Convey("Given an incorrect host", func() {
			conf.GlobalConfig.Admin.Host = "invalid_host"
			conf.GlobalConfig.Admin.Port = 0
			rest := &Server{}

			Convey("When starting the service", func() {
				err := rest.Start()

				Convey("Then it should produce an error", func() {
					So(err, ShouldBeError)
				})
			})
		})

		Convey("Given an incorrect certificate", func() {
			conf.GlobalConfig.Admin.Host = "localhost"
			conf.GlobalConfig.Admin.Port = 0
			conf.GlobalConfig.Admin.TLSCert = "not_a_cert"
			conf.GlobalConfig.Admin.TLSKey = "not_a_key"
			rest := &Server{}

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
		conf.GlobalConfig.Admin = conf.AdminConfig{Host: "localhost"}
		rest := &Server{}

		err := rest.Start()
		So(err, ShouldBeNil)

		Convey("When the service is stopped", func() {
			addr := rest.server.Addr

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			So(rest.Stop(ctx), ShouldBeNil)

			Reset(cancel)

			Convey("Then the service should no longer respond to requests", func() {
				client := new(http.Client)
				response, err := client.Get(addr) //nolint:noctx // this is a test

				So(err, ShouldBeError)
				So(response, ShouldBeNil)

				if response != nil {
					_ = response.Body.Close()
				}
			})

			Convey("When the service is stopped a 2nd time", func() {
				ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second*10)
				So(rest.Stop(ctx2), ShouldBeError, utils.ErrNotRunning)

				Reset(cancel2)

				Convey("Then it should not do anything", func() {
					code, reason := rest.State()

					So(code, ShouldEqual, utils.StateOffline)
					So(reason, ShouldBeEmpty)
				})
			})
		})
	})
}

func TestAuthentication(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	Convey("Given an authentication handler", t, func(c C) {
		logger := testhelpers.TestLogger(c, "rest_auth_test")
		db := database.TestDatabase(c)
		authHandler := AuthenticationMiddleware(logger, db).Middleware(handler)

		Convey("Given an incoming request", func() {
			w := httptest.NewRecorder()
			r, err := http.NewRequest(http.MethodGet, "/api", nil)

			So(err, ShouldBeNil)

			Convey("Given valid credentials", func() {
				r.SetBasicAuth("admin", "admin_password")

				Convey("When sending the request", func() {
					authHandler.ServeHTTP(w, r)

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
					authHandler.ServeHTTP(w, r)

					Convey("Then the handler should reply 'Unauthorized'", func() {
						So(w.Code, ShouldEqual, http.StatusUnauthorized)
					})
				})
			})

			Convey("Given an incorrect password", func() {
				r.SetBasicAuth("admin", "not_admin_password")

				Convey("When sending the request", func() {
					authHandler.ServeHTTP(w, r)

					Convey("Then the handler should reply 'Unauthorized'", func() {
						So(w.Code, ShouldEqual, http.StatusUnauthorized)
					})
				})
			})
		})
	})
}
