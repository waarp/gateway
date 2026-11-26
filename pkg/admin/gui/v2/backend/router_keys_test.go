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
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// TestKeysRouter checks the right each route of the "Cryptographic keys"
// screen requires: the rights on the administration, as the REST API of the
// keys.
func TestKeysRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)

	key := &model.CryptoKey{Name: "hmac-key", Type: model.CryptoKeyTypeHMAC, Key: "c2VjcmV0"}
	require.NoError(t, db.Insert(key).Run())

	router := mux.NewRouter()
	MakeKeysRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(perms model.PermsMask, method, target string) int {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "tester", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec.Code
	}

	id := fmt.Sprintf("/keys/%d", key.ID)
	read := model.PermAdminRead
	write := read | model.PermAdminWrite
	del := read | model.PermAdminDelete

	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, "/keys"))
	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, id))
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/keys/999"))
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/keys/abc"), "an identifier is a number")
	assert.Equal(t, http.StatusMethodNotAllowed, serve(del, http.MethodDelete, id), "every action is a POST")

	// The deletion comes last, once the other routes have answered.
	for _, route := range []struct {
		method, target string
		perm           model.PermsMask
		lacking        model.PermsMask
	}{
		{http.MethodGet, "/keys", read, 0},
		{http.MethodGet, id, read, 0},
		{http.MethodGet, "/keys/new", write, del},
		{http.MethodPost, "/keys/new", write, del},
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
