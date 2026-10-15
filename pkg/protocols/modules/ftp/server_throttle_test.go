package ftp

import (
	"net"
	"testing"
	"time"

	ftplib "github.com/fclairamb/ftpserverlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

type throttleClientContext struct {
	ftplib.ClientContext

	addr  net.Addr
	extra any
}

func (c *throttleClientContext) RemoteAddr() net.Addr { return c.addr }
func (c *throttleClientContext) Extra() any           { return c.extra }

func TestAuthUserThrottle(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	db.Config.Security = conf.SecurityConfig{
		MaxAuthFailures: 3, AuthFailureWindow: 5 * time.Minute,
		AuthLockDuration: time.Minute, AuthMaxLockDuration: 30 * time.Minute,
	}

	agent := &model.LocalAgent{
		Name: "ftp_throttle_server", Protocol: FTP, Address: types.Addr("localhost", 0),
	}
	require.NoError(t, db.Insert(agent).Run())

	account := &model.LocalAccount{LocalAgentID: agent.ID, Login: "ftp_throttle_user"}
	require.NoError(t, db.Insert(account).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: account.NullableID(), Type: auth.Password, Value: "sesame",
	}).Run())

	h := &handler{db: db, logger: testhelpers.GetTestLogger(t), dbServer: agent}

	authenticate := func(ip, pswd string) error {
		cc := &throttleClientContext{
			addr:  &net.TCPAddr{IP: net.ParseIP(ip), Port: 6666},
			extra: account,
		}

		_, err := h.AuthUser(cc, account.Login, pswd)

		return err
	}

	for range 3 {
		require.Error(t, authenticate("10.0.0.1", "wrong"))
	}

	assert.Error(t, authenticate("10.0.0.1", "sesame"),
		"valid password refused while the address is locked")
	assert.NoError(t, authenticate("10.0.0.2", "sesame"),
		"valid password accepted from another address")
}
