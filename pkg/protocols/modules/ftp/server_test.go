package ftp

import (
	"context"
	"net"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestAuth(t *testing.T) {
	const (
		login    = "foobar"
		password = "sesame"
	)

	Convey("Given a test FTP server", t, func(c C) {
		db := database.TestDatabase(c)

		locAgent := &model.LocalAgent{
			Name:     "ftp_test_auth",
			Address:  types.Addr("localhost", 0),
			Protocol: FTP,
		}
		So(db.Insert(locAgent).Run(), ShouldBeNil)

		server := newServer(db, locAgent)
		So(server.Start(), ShouldBeNil)

		Reset(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			So(server.Stop(ctx), ShouldBeNil)
		})

		locAccount := &model.LocalAccount{
			LocalAgentID: locAgent.ID,
			Login:        login,
		}

		So(db.Insert(locAccount).Run(), ShouldBeNil)
		So(db.Insert(&model.Credential{
			LocalAccountID: utils.NewNullInt64(locAccount.ID),
			Type:           auth.Password,
			Value:          password,
		}).Run(), ShouldBeNil)

		Convey("Given a normal account", func() {
			Convey("When logging in", func() {
				clientAddr, addrErr := net.ResolveTCPAddr("tcp", "127.0.0.1:21")
				require.NoError(t, addrErr)

				cc := testClientContext{remoteAddr: clientAddr}

				Convey("Then it should succeed", func() {
					_, err := server.handler.AuthUser(cc, login, password)
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given an IP-restricted account", func() {
			locAccount.IPAddresses = []string{"1.2.3.4"}
			So(db.Update(locAccount).Run(), ShouldBeNil)

			Convey("When logging in from the correct IP", func() {
				clientAddr, addrErr := net.ResolveTCPAddr("tcp", "1.2.3.4:21")
				require.NoError(t, addrErr)

				cc := testClientContext{remoteAddr: clientAddr}

				Convey("Then it should succeed", func() {
					_, err := server.handler.AuthUser(cc, login, password)
					So(err, ShouldBeNil)
				})
			})

			Convey("When logging in from an unauthorized IP", func() {
				clientAddr, addrErr := net.ResolveTCPAddr("tcp", "9.8.7.6:21")
				require.NoError(t, addrErr)

				cc := testClientContext{remoteAddr: clientAddr}

				Convey("Then it should fail", func() {
					_, err := server.handler.AuthUser(cc, login, password)
					So(err, ShouldBeError, "unauthorized IP address")
				})
			})
		})
	})
}
