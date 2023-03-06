package model

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/names"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestClientBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a new client entry", func() {
			client := &Client{
				Name:         "new_client",
				Protocol:     testProtocol,
				LocalAddress: "1.2.3.4:5",
			}

			Convey("Given that the new client is valid", func() {
				Convey("Then the 'BeforeWrite' method should not return any error", func() {
					So(client.BeforeWrite(db), ShouldBeNil)
				})
			})

			Convey("Given that the client name is missing", func() {
				client.Name = ""

				Convey("Then the 'BeforeWrite' method should not return any error", func() {
					So(client.BeforeWrite(db), ShouldBeNil)

					Convey("Then it should have used the protocol as a name", func() {
						So(client.Name, ShouldEqual, client.Protocol)
					})
				})
			})

			Convey("Given that the client's name is reserved", func() {
				client.Name = names.DatabaseServiceName

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError, fmt.Sprintf(
						`%q is a reserved service name`, client.Name))
				})
			})

			Convey("Given that the client's protocol is missing", func() {
				client.Protocol = ""

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError, "the client's protocol is missing")
				})
			})

			Convey("Given that the client's protocol is invalid", func() {
				client.Protocol = "foobar"

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError, `"foobar" is not a protocol`)
				})
			})

			Convey("Given that the client's address is invalid", func() {
				client.LocalAddress = "not_an_address"

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError,
						`"not_an_address" is not a valid client address: `+
							`address not_an_address: missing port in address`)
				})
			})

			Convey("Given that the client's proto config cannot be parsed", func() {
				client.ProtoConfig = map[string]any{"": nil}

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError,
						`invalid proto config: json: unknown field ""`)
				})
			})

			Convey("Given that the client's proto config is invalid", func() {
				client.Protocol = testProtocolInvalid

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError,
						testhelpers.ErrClientValidationFailed)
				})
			})

			Convey("Given that another client with the same name already exist", func() {
				otherClient := &Client{
					Name:         client.Name,
					Protocol:     testProtocol,
					LocalAddress: "9.8.7.6:5",
				}
				So(db.Insert(otherClient).Run(), ShouldBeNil)

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError, fmt.Sprintf(
						"a client named %q already exist", client.Name))
				})
			})
		})
	})
}
