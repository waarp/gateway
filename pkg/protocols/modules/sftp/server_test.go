package sftp

import (
	"net"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestAuth(t *testing.T) {
	const (
		login    = "foobar"
		password = "sesame"
	)

	userKey, keyErr := ParseAuthorizedKey(SSHPbk)
	require.NoError(t, keyErr)

	Convey("Given a test SFTP server", t, func(c C) {
		db := database.TestDatabase(c)
		logger := testhelpers.TestLogger(c, "test_sftp_auth")

		locAgent := &model.LocalAgent{
			Name:     "sftp_test_auth",
			Address:  types.Addr("localhost", 0),
			Protocol: SFTP,
		}
		So(db.Insert(locAgent).Run(), ShouldBeNil)

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
		So(db.Insert(&model.Credential{
			LocalAccountID: utils.NewNullInt64(locAccount.ID),
			Type:           AuthSSHPublicKey,
			Value:          SSHPbk,
		}).Run(), ShouldBeNil)

		type authCallback func(ssh.ConnMetadata) (*ssh.Permissions, error)

		for authType, callback := range map[string]authCallback{
			"password": func(metadata ssh.ConnMetadata) (*ssh.Permissions, error) {
				return passwordCallback(db, logger, locAgent)(metadata, []byte(password))
			},
			"publickey": func(metadata ssh.ConnMetadata) (*ssh.Permissions, error) {
				return userKeyCallback(db, logger, locAgent)(metadata, userKey)
			},
		} {
			Convey("When logging in via "+authType, func() {
				Convey("Given a normal account", func() {
					Convey("When logging in", func() {
						clientAddr, addrErr := net.ResolveTCPAddr("tcp", "127.0.0.1:22")
						require.NoError(t, addrErr)

						connMetadata := testConnMetadata{
							user:       login,
							remoteAddr: clientAddr,
						}

						Convey("Then it should succeed", func() {
							_, err := callback(connMetadata)
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given an IP-restricted account", func() {
					locAccount.IPAddresses = []string{"127.0.0.1"}
					So(db.Update(locAccount).Run(), ShouldBeNil)

					Convey("When logging in from the correct IP", func() {
						clientAddr, addrErr := net.ResolveTCPAddr("tcp", "127.0.0.1:22")
						require.NoError(t, addrErr)

						connMetadata := testConnMetadata{
							user:       login,
							remoteAddr: clientAddr,
						}

						Convey("Then it should succeed", func() {
							_, err := callback(connMetadata)
							So(err, ShouldBeNil)
						})
					})

					Convey("When logging in from an unauthorized IP", func() {
						clientAddr, addrErr := net.ResolveTCPAddr("tcp", "1.2.3.4:22")
						require.NoError(t, addrErr)

						connMetadata := testConnMetadata{
							user:       login,
							remoteAddr: clientAddr,
						}

						Convey("Then it should fail", func() {
							_, err := callback(connMetadata)
							So(err, ShouldBeError, ErrUnauthorizedIP)
						})
					})
				})
			})
		}
	})
}
