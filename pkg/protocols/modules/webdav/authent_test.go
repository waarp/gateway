package webdav

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestAuthCredentials(t *testing.T) {
	t.Parallel()

	const (
		login    = "webdav_user"
		password = "sesame"
	)

	db := dbtest.TestDatabase(t)

	agent := &model.LocalAgent{Name: "webdav_server", Protocol: Webdav, Address: types.Addr("localhost", 0)}
	require.NoError(t, db.Insert(agent).Run())

	account := &model.LocalAccount{LocalAgentID: agent.ID, Login: login}
	require.NoError(t, db.Insert(account).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: account.NullableID(), Type: auth.Password, Value: password,
	}).Run())

	srv := &server{db: db, logger: testhelpers.GetTestLogger(t), agent: agent}

	try := func(user, pswd string) (*model.LocalAccount, bool, int) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.SetBasicAuth(user, pswd)

		rec := httptest.NewRecorder()
		acc, ok := srv.auth(rec, req)

		return acc, ok, rec.Code
	}

	t.Run("Given valid credentials", func(t *testing.T) {
		t.Parallel()

		acc, ok, _ := try(login, password)
		require.True(t, ok)
		assert.Equal(t, account.ID, acc.ID)
	})

	t.Run("Given a wrong password", func(t *testing.T) {
		t.Parallel()

		_, ok, code := try(login, "not_sesame")
		assert.False(t, ok, "a wrong password must not authenticate the request")
		assert.Equal(t, http.StatusUnauthorized, code)
	})

	t.Run("Given an unknown account", func(t *testing.T) {
		t.Parallel()

		_, ok, code := try("unknown_user", "whatever")
		assert.False(t, ok, "an unknown account must not authenticate the request")
		assert.Equal(t, http.StatusUnauthorized, code)
	})
}
