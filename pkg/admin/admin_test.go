package admin

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestStart(t *testing.T) {
	db := &database.DB{Config: &conf.ServerConfig{}}

	Convey("Given an admin service", t, func(c C) {
		certFile := testhelpers.TempFile(c, "rest_cert_*.pem")
		keyFile := testhelpers.TempFile(c, "rest_key_*.pem")

		So(os.WriteFile(certFile, []byte(cert), 0o600), ShouldBeNil)
		So(os.WriteFile(keyFile, []byte(key), 0o600), ShouldBeNil)

		db.Config.Admin = conf.AdminConfig{
			Host:    "localhost",
			Port:    0,
			TLSCert: certFile,
			TLSKey:  keyFile,
		}
		server := &Server{DB: db}

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
			})
		})

		Convey("Given a key file with a passphrase", func() {
			keyFilePass := testhelpers.TempFile(c, "rest_key_passphrase_*.pem")
			So(os.WriteFile(keyFilePass, []byte(keyWithPassphrase), 0o600), ShouldBeNil)

			db.Config.Admin.TLSKey = keyFilePass
			db.Config.Admin.TLSPassphrase = keyPassphrase

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
			db.Config.Admin.Host = "invalid_host"
			db.Config.Admin.Port = 0
			rest := &Server{DB: db}

			Convey("When starting the service", func() {
				err := rest.Start()

				Convey("Then it should produce an error", func() {
					So(err, ShouldBeError)
				})
			})
		})

		Convey("Given an incorrect certificate", func() {
			db.Config.Admin.Host = "localhost"
			db.Config.Admin.Port = 0
			db.Config.Admin.TLSCert = "not_a_cert"
			db.Config.Admin.TLSKey = "not_a_key"
			rest := &Server{DB: db}

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
	db := &database.DB{Config: &conf.ServerConfig{}}

	Convey("Given a running REST service", t, func() {
		db.Config.Admin = conf.AdminConfig{Host: "localhost"}
		rest := &Server{DB: db}

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
