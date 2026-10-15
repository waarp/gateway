package sftp

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

type throttleConnMetadata struct {
	ssh.ConnMetadata

	user string
	addr net.Addr
}

func (c *throttleConnMetadata) User() string         { return c.user }
func (c *throttleConnMetadata) RemoteAddr() net.Addr { return c.addr }

func TestPasswordCallbackThrottle(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	db.Config.Security = conf.SecurityConfig{
		MaxAuthFailures: 3, AuthFailureWindow: 5 * time.Minute,
		AuthLockDuration: time.Minute, AuthMaxLockDuration: 30 * time.Minute,
	}

	agent := &model.LocalAgent{
		Name: "sftp_throttle_server", Protocol: SFTP, Address: types.Addr("localhost", 0),
	}
	require.NoError(t, db.Insert(agent).Run())

	account := &model.LocalAccount{LocalAgentID: agent.ID, Login: "sftp_throttle_user"}
	require.NoError(t, db.Insert(account).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: account.NullableID(), Type: auth.Password, Value: "sesame",
	}).Run())

	logger := testhelpers.GetTestLogger(t)
	callback := passwordCallback(db, logger, agent)

	conn := func(ip string) ssh.ConnMetadata {
		return &throttleConnMetadata{
			user: account.Login,
			addr: &net.TCPAddr{IP: net.ParseIP(ip), Port: 6666},
		}
	}

	for range 3 {
		_, err := callback(conn("10.0.0.1"), []byte("wrong"))
		require.ErrorIs(t, err, ErrAuthFailed)
	}

	_, err := callback(conn("10.0.0.1"), []byte("sesame"))
	require.ErrorIs(t, err, ErrAuthFailed, "valid password refused while the address is locked")

	_, err = callback(conn("10.0.0.2"), []byte("sesame"))
	require.NoError(t, err, "valid password accepted from another address")

	pubKeyCalled := false
	pubKeyCallback := withThrottle(db, logger, agent,
		func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
			pubKeyCalled = true

			return &ssh.Permissions{}, nil
		})

	_, err = pubKeyCallback(conn("10.0.0.1"), nil)
	require.ErrorIs(t, err, ErrAuthFailed, "public key refused while the address is locked")
	assert.False(t, pubKeyCalled)

	_, err = pubKeyCallback(conn("10.0.0.2"), nil)
	require.NoError(t, err, "public key accepted from another address")
	assert.True(t, pubKeyCalled)
}
