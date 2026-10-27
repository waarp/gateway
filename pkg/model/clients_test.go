package model

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestClientBeforeWrite(t *testing.T) {
	t.Parallel()

	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a new client entry", func() {
			client := &Client{
				Name: "new_client", Protocol: testProtocol,
				LocalAddress: types.Addr("1.2.3.4", 5),
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

			Convey("Given that the client's protocol is missing", func() {
				client.Protocol = ""

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError, "the client's protocol is missing")
				})
			})

			Convey("Given that the client's protocol is invalid", func() {
				client.Protocol = "foobar"

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError, `unknown protocol "foobar"`)
				})
			})

			Convey("Given that the client's address is invalid", func() {
				client.LocalAddress.Host = "not_an_address"

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					var dnsErr *net.DNSError

					err := client.BeforeWrite(db)
					So(err, ShouldNotBeNil)
					So(errors.As(err, &dnsErr), ShouldBeTrue)
				})
			})

			Convey("Given that the client's proto config is invalid", func() {
				client.Protocol = testProtocolInvalid

				Convey("Then the 'BeforeWrite' method should return an error", func() {
					So(client.BeforeWrite(db), ShouldBeError, errInvalidProtoConfig)
				})
			})

			Convey("Given that another client with the same name already exist", func() {
				otherClient := &Client{
					Name: client.Name, Protocol: testProtocol,
					LocalAddress: types.Addr("9.8.7.6", 5),
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

func TestClientBeforeDelete(t *testing.T) {
	t.Parallel()
	db := dbtest.TestDatabase(t)

	addClient := func(tb testing.TB) *Client {
		tb.Helper()
		client := &Client{
			Name:         tb.Name(),
			Protocol:     testProtocol,
			LocalAddress: types.Addr("localhost", 0),
		}
		require.NoError(t, db.Insert(client).Run())

		return client
	}

	addTransCtx := func(tb testing.TB) (*Rule, *RemoteAccount) {
		tb.Helper()
		rule := &Rule{Name: tb.Name()}
		require.NoError(t, db.Insert(rule).Run())

		partner := &RemoteAgent{Name: tb.Name(), Protocol: testProtocol, Address: types.Addr("localhost", 0)}
		require.NoError(t, db.Insert(partner).Run())

		account := &RemoteAccount{Login: tb.Name(), RemoteAgent: *partner}
		require.NoError(t, db.Insert(account).Run())

		return rule, account
	}

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		client := addClient(t)

		require.NoError(t, client.BeforeDelete(db))
	})

	t.Run("Used in transfer", func(t *testing.T) {
		t.Parallel()
		client := addClient(t)
		rule, account := addTransCtx(t)

		transfer := &Transfer{
			RuleID:          rule.ID,
			ClientID:        client.NullableID(),
			RemoteAccountID: account.NullableID(),
			SrcFilename:     "file.test",
		}
		require.NoError(t, db.Insert(transfer).Run())

		err := client.BeforeDelete(db)
		assertIsFKErrFor(t, err, transfer)
	})

	t.Run("Used in filewatcher", func(t *testing.T) {
		t.Parallel()
		client := addClient(t)
		rule, account := addTransCtx(t)

		fw := &FileWatcher{
			Flow:          t.Name(),
			Rule:          *rule,
			Client:        *client,
			RemoteAccount: *account,
		}
		require.NoError(t, db.Insert(fw).Run())

		err := client.BeforeDelete(db)
		assertIsFKErrFor(t, err, fw)
	})
}
