package r66

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	"code.waarp.fr/waarp-r66/r66"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTLS(t *testing.T) {
	logger := log.NewLogger("test_r66_tls")

	Convey("Given a TLS R66 server", t, func(c C) {
		db := database.GetTestDatabase()

		addr := fmt.Sprintf("localhost:%d", testhelpers.GetFreePort(c))
		server := &model.LocalAgent{
			Name:        "r66_tls",
			Protocol:    "r66",
			ProtoConfig: json.RawMessage(`{"serverPassword":"c2VzYW1l"}`),
			Address:     addr,
		}
		So(db.Create(server), ShouldBeNil)

		servCert := &model.Cert{
			OwnerType:   server.TableName(),
			OwnerID:     server.ID,
			Name:        "server cert",
			PrivateKey:  []byte(testKey),
			Certificate: []byte(testCrt),
		}
		So(db.Create(servCert), ShouldBeNil)

		service := NewService(db, server, logger)
		So(service.Start(), ShouldBeNil)
		Reset(func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			So(service.Stop(ctx), ShouldBeNil)
		})

		Convey("Given a TLS R66 client", func() {
			client := &client{
				conf:      config.R66ProtoConfig{IsTLS: true},
				r66Client: r66.NewClient("toto", []byte("sesame")),
				info: model.OutTransferInfo{
					Agent: &model.RemoteAgent{
						Address: addr,
					},
					ClientCerts: []model.Cert{{
						PrivateKey:  []byte(testKey),
						Certificate: []byte(testCrt),
					}},
					ServerCerts: []model.Cert{{
						PrivateKey:  []byte(testKey),
						Certificate: []byte(testCrt),
					}},
				},
			}
			var err error
			client.tlsConf, err = makeClientTLSConfig(&client.info)
			So(err, ShouldBeNil)

			Convey("When connecting to the server", func() {
				err := client.Connect()

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
					Reset(client.remote.Close)

					Convey("Then the connection should be opened", func() {
						So(client.remote, ShouldNotBeNil)
						So(client.Authenticate(), ShouldNotBeNil)
					})
				})
			})
		})

		Convey("Given that the server certificate is unknown", func() {
			client := &client{
				conf:      config.R66ProtoConfig{IsTLS: true},
				r66Client: r66.NewClient("toto", []byte("sesame")),
				info: model.OutTransferInfo{
					Agent: &model.RemoteAgent{
						Address: addr,
					},
					ClientCerts: []model.Cert{{
						PrivateKey:  []byte(testKey),
						Certificate: []byte(testCrt),
					}},
				},
			}
			var err error
			client.tlsConf, err = makeClientTLSConfig(&client.info)
			So(err, ShouldBeNil)

			Convey("When connecting to the server", func() {
				err := client.Connect()

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, model.TransferError{Code: model.TeConnection,
						Details: "x509: certificate signed by unknown authority"})
				})
			})
		})

		Convey("Given that the client does not provide a certificate", func() {
			client := &client{
				conf:      config.R66ProtoConfig{IsTLS: true},
				r66Client: r66.NewClient("toto", []byte("sesame")),
				info: model.OutTransferInfo{
					Agent: &model.RemoteAgent{
						Address: addr,
					},
					ServerCerts: []model.Cert{{
						PrivateKey:  []byte(testKey),
						Certificate: []byte(testCrt),
					}},
				},
			}
			var err error
			client.tlsConf, err = makeClientTLSConfig(&client.info)
			So(err, ShouldBeNil)

			Convey("When connecting to the server", func() {
				So(client.Connect(), ShouldBeNil)
				Reset(client.remote.Close)
				err := client.Authenticate()

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}
