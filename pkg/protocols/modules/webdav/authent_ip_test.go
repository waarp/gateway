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

func TestAuthIPRestriction(t *testing.T) {
	t.Parallel()

	const (
		login    = "webdav_user"
		password = "sesame"
	)

	db := dbtest.TestDatabase(t)

	agent := &model.LocalAgent{Name: "webdav_server", Protocol: Webdav, Address: types.Addr("localhost", 0)}
	require.NoError(t, db.Insert(agent).Run())

	account := &model.LocalAccount{LocalAgentID: agent.ID, Login: login, IPAddresses: []string{"1.2.3.4"}}
	require.NoError(t, db.Insert(account).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: account.NullableID(), Type: auth.Password, Value: password,
	}).Run())

	srv := &server{db: db, logger: testhelpers.GetTestLogger(t), agent: agent}

	try := func(remoteAddr, pswd string) (bool, *httptest.ResponseRecorder) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		req.SetBasicAuth(login, pswd)

		rec := httptest.NewRecorder()
		_, ok := srv.auth(rec, req)

		return ok, rec
	}

	ok, _ := try("1.2.3.4:80", password)
	assert.True(t, ok, "allowed address with valid credentials")

	okBad, recBad := try("1.2.3.4:80", "not_sesame")
	okIP, recIP := try("9.8.7.6:80", password)

	assert.False(t, okBad)
	assert.False(t, okIP)
	assert.Equal(t, recBad.Code, recIP.Code, "same status for a disallowed address")
	assert.Equal(t, recBad.Body.String(), recIP.Body.String(), "same answer for a disallowed address")
}
