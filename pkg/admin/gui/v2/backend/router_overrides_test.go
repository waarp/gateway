package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// TestOverridesRouter checks the right each route of the "Configuration
// overrides" screen requires: the rights on the administration, as the REST
// API of the overrides.
func TestOverridesRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	db.Config.NodeID = "noeud-1"
	db.Config.Overrides = conf.NewOverride(filepath.Join(t.TempDir(), "noeud-1.ini"))
	require.NoError(t, db.Config.Overrides.AddIndirection("lb.example.org", "10.0.0.5"))

	router := mux.NewRouter()
	MakeOverridesRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(perms model.PermsMask, method, target string) int {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "tester", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec.Code
	}

	const rule = "?target=lb.example.org"

	read := model.PermAdminRead
	write := read | model.PermAdminWrite
	del := read | model.PermAdminDelete

	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, "/overrides"))
	assert.Equal(t, http.StatusNotFound, serve(write, http.MethodGet, "/overrides/edit?target=absent.example.org"))
	assert.Equal(t, http.StatusMethodNotAllowed, serve(del, http.MethodDelete, "/overrides/delete"+rule),
		"every action is a POST")

	// The deletion comes last, once the other routes have answered.
	for _, route := range []struct {
		method, target string
		perm           model.PermsMask
		lacking        model.PermsMask
	}{
		{http.MethodGet, "/overrides", read, 0},
		{http.MethodGet, "/overrides/new", write, del},
		{http.MethodPost, "/overrides/new", write, del},
		{http.MethodGet, "/overrides/edit" + rule, write, del},
		{http.MethodPost, "/overrides/edit" + rule, write, del},
		{http.MethodGet, "/overrides/delete" + rule, del, write},
		{http.MethodPost, "/overrides/delete" + rule, del, write},
	} {
		assert.Equalf(t, http.StatusForbidden, serve(route.lacking, route.method, route.target),
			"%s %s must be refused without the right", route.method, route.target)
		assert.NotEqualf(t, http.StatusForbidden, serve(route.perm, route.method, route.target),
			"%s %s must be served with the right", route.method, route.target)
	}
}
