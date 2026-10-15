package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCheckAuthentThrottle(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	db.Config.Security = conf.SecurityConfig{
		MaxAuthFailures: 3, AuthFailureWindow: 5 * time.Minute,
		AuthLockDuration: time.Minute, AuthMaxLockDuration: 30 * time.Minute,
	}

	server := &model.LocalAgent{
		Name: "http_throttle_server", Protocol: HTTP, Address: types.Addr("localhost", 0),
	}
	require.NoError(t, db.Insert(server).Run())

	account := &model.LocalAccount{LocalAgentID: server.ID, Login: "http_throttle_user"}
	require.NoError(t, db.Insert(account).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: account.NullableID(), Type: auth.Password, Value: "sesame",
	}).Run())

	service := &httpService{db: db, agent: server, logger: testhelpers.GetTestLogger(t)}

	authenticate := func(addr, pswd string) bool {
		req := httptest.NewRequest(http.MethodGet, "/file", nil)
		req.RemoteAddr = addr
		req.SetBasicAuth(account.Login, pswd)

		_, ok := service.checkAuthent(httptest.NewRecorder(), req)

		return ok
	}

	for range 3 {
		require.False(t, authenticate("10.0.0.1:6666", "wrong"))
	}

	assert.False(t, authenticate("10.0.0.1:6666", "sesame"),
		"valid credentials refused while the address is locked")
	assert.True(t, authenticate("10.0.0.2:6666", "sesame"),
		"valid credentials accepted from another address")
}
