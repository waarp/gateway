package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// TestSMTPCredentialsRouter checks the right each route of the "SMTP
// credentials" screen requires: the rights on the rules, as the REST API of
// the credentials, not those on the administration.
func TestSMTPCredentialsRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)

	cred := &model.SMTPCredential{EmailAddress: "alerts@example.org", ServerAddress: types.Addr("smtp.example.org", 587)}
	require.NoError(t, db.Insert(cred).Run())

	router := mux.NewRouter()
	MakeSMTPCredentialsRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(perms model.PermsMask, method, target string) int {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "tester", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec.Code
	}

	id := fmt.Sprintf("/smtp/%d", cred.ID)
	read := model.PermRulesRead
	write := read | model.PermRulesWrite
	del := read | model.PermRulesDelete
	admin := model.PermAdminRead | model.PermAdminWrite | model.PermAdminDelete

	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, "/smtp"))
	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, id))
	assert.Equal(t, http.StatusForbidden, serve(admin, http.MethodGet, "/smtp"), "the rights on the administration do not open it")
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/smtp/999"))
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/smtp/abc"), "an identifier is a number")
	assert.Equal(t, http.StatusMethodNotAllowed, serve(del, http.MethodDelete, id), "every action is a POST")

	// The deletion comes last, once the other routes have answered.
	for _, route := range []struct {
		method, target string
		perm           model.PermsMask
		lacking        model.PermsMask
	}{
		{http.MethodGet, "/smtp", read, 0},
		{http.MethodGet, id, read, 0},
		{http.MethodGet, "/smtp/new", write, del},
		{http.MethodPost, "/smtp/new", write, del},
		{http.MethodGet, id + "/edit", write, del},
		{http.MethodPost, id + "/edit", write, del},
		{http.MethodGet, id + "/delete", del, write},
		{http.MethodPost, id + "/delete", del, write},
	} {
		assert.Equalf(t, http.StatusForbidden, serve(route.lacking, route.method, route.target),
			"%s %s must be refused without the right", route.method, route.target)
		assert.NotEqualf(t, http.StatusForbidden, serve(route.perm, route.method, route.target),
			"%s %s must be served with the right", route.method, route.target)
	}
}
