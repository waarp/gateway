package r66

import (
	"encoding/json"
	"testing"
	"time"

	"code.waarp.fr/lib/r66"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestValidAuthThrottle(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	db.Config.Security = conf.SecurityConfig{
		MaxAuthFailures: 3, AuthFailureWindow: 5 * time.Minute,
		AuthLockDuration: time.Minute, AuthMaxLockDuration: 30 * time.Minute,
	}

	server := &model.LocalAgent{
		Name: "r66_throttle_server", Protocol: R66, Address: types.Addr("localhost", 0),
		ProtoConfig: map[string]any{"blockSize": json.Number("512"), "serverPassword": "c2VzYW1l"},
	}
	require.NoError(t, db.Insert(server).Run())

	account := &model.LocalAccount{LocalAgentID: server.ID, LocalAgent: *server, Login: "r66_throttle_user"}
	require.NoError(t, db.Insert(account).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: account.NullableID(), Type: auth.Password, Value: "sesame",
	}).Run())

	handler := &authHandler{service: &service{
		db: db, logger: testhelpers.GetTestLogger(t), dbAgent: server, r66Conf: &ServerConfigTLS{},
	}}

	authenticate := func(addr, pswd string) error {
		_, err := handler.ValidAuth(&r66.Authent{
			Login: account.Login, Password: r66.CryptPass([]byte(pswd)), Address: addr,
			Filesize: true, FinalHash: true, Digest: "SHA-256",
		})

		return err
	}

	for range 3 {
		require.Error(t, authenticate("10.0.0.1:6666", "wrong"))
	}

	assert.Error(t, authenticate("10.0.0.1:6666", "sesame"),
		"valid credentials refused while the address is locked")
	assert.NoError(t, authenticate("10.0.0.2:6666", "sesame"),
		"valid credentials accepted from another address")
}
