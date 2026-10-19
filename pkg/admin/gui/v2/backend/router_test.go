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

func TestServicesRouter(t *testing.T) {
	t.Parallel()

	router := mux.NewRouter()
	MakeServicesRouter(router, testhelpers.GetTestLogger(t))

	serve := func(method, target string) *httptest.ResponseRecorder {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "reader", Permissions: 0})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec
	}

	t.Run("Given a GET request from a user without any permission", func(t *testing.T) {
		t.Parallel()

		rec := serve(http.MethodGet, "/services")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `id="etat"`)
	})

	t.Run("Given a request for the state fragment", func(t *testing.T) {
		t.Parallel()

		rec := serve(http.MethodGet, "/services?fragment=state")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), "<html")
	})

	t.Run("Given a POST request", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, http.StatusMethodNotAllowed, serve(http.MethodPost, "/services").Code,
			"the screen has no action")
	})
}

// TestHomeRouter checks that the home page and the "About" page need no
// particular right and serve GET only.
func TestHomeRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	user := &model.User{Username: "reader", PasswordHash: "$2a$04$notahashbutnotemptyeither", Permissions: 0}
	require.NoError(t, db.Insert(user).Run())

	router := mux.NewRouter()
	MakeHomeRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(method, target string) *httptest.ResponseRecorder {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec
	}

	rec := serve(http.MethodGet, "/home")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `<h1 class="w-page-header__title">Home</h1>`)
	assert.Equal(t, http.StatusOK, serve(http.MethodGet, "/about").Code)
	assert.Equal(t, http.StatusOK, serve(http.MethodGet, "/account").Code)
	assert.Equal(t, http.StatusMethodNotAllowed, serve(http.MethodPost, "/home").Code, "the pages have no action")
	assert.Equal(t, http.StatusMethodNotAllowed, serve(http.MethodGet, "/account/password").Code, "the change is a POST")
	assert.Equal(t, http.StatusUnprocessableEntity, serve(http.MethodPost, "/account/password").Code,
		"an empty form is refused, not served")
}

// TestServersRouter checks the right each route of the "Local servers"
// screen requires: reading for the list and the details, writing for the
// creation, the edition, the start and the stop, deleting for the deletion.
func TestServersRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	server := &model.LocalAgent{Name: "srv-sftp-01", Protocol: "sftp", Address: types.Addr("0.0.0.0", 10022)}
	require.NoError(t, db.Insert(server).Run())

	router := mux.NewRouter()
	MakeServersRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(perms model.PermsMask, method, target string) int {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "tester", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec.Code
	}

	account := &model.LocalAccount{LocalAgentID: server.ID, Login: "partner"}
	require.NoError(t, db.Insert(account).Run())
	cred := &model.Credential{LocalAccountID: account.NullableID(), Name: "mdp", Type: "password", Value: "pa55"}
	require.NoError(t, db.Insert(cred).Run())

	id := fmt.Sprintf("/servers/%d", server.ID)
	accountID := fmt.Sprintf("%s/accounts/%d", id, account.ID)
	credID := fmt.Sprintf("%s/auth/%d", accountID, cred.ID)
	read := model.PermServersRead
	write := read | model.PermServersWrite
	del := read | model.PermServersDelete

	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, "/servers"))
	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, id))
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/servers/999"))
	assert.Equal(t, http.StatusMethodNotAllowed, serve(del, http.MethodDelete, id), "every action is a POST")
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/servers/abc"), "an identifier is a number")

	// The actions run for real against the gateway's service lists: the
	// deletion comes last, once the other routes have answered.
	for _, route := range []struct {
		method, target string
		perm           model.PermsMask
		lacking        model.PermsMask
	}{
		{http.MethodGet, "/servers", read, 0},
		{http.MethodGet, id, read, 0},
		{http.MethodGet, "/servers/new", write, del},
		{http.MethodPost, "/servers/new", write, del},
		{http.MethodGet, id + "/edit", write, del},
		{http.MethodPost, id + "/edit", write, del},
		{http.MethodPost, id + "/start", write, del},
		{http.MethodPost, id + "/stop", write, del},
		{http.MethodGet, id + "/delete", del, write},
		{http.MethodPost, id + "/delete", del, write},

		// The sheet: its tabs and the pages in reading behind the read
		// right, the forms behind the write right, the deletions behind
		// the delete right (hardening decided on 2026-10-18).
		{http.MethodGet, id + "/auth", read, 0},
		{http.MethodGet, id + "/auth/new", write, del},
		{http.MethodPost, id + "/auth/new", write, del},
		{http.MethodGet, id + "/accounts", read, 0},
		{http.MethodGet, id + "/accounts/new", write, del},
		{http.MethodPost, id + "/accounts/new", write, del},
		{http.MethodGet, accountID, read, 0},
		{http.MethodGet, accountID + "/edit", write, del},
		{http.MethodPost, accountID + "/edit", write, del},
		{http.MethodGet, accountID + "/delete", del, write},
		{http.MethodGet, accountID + "/auth", read, 0},
		{http.MethodGet, accountID + "/auth/new", write, del},
		{http.MethodGet, credID, read, 0},
		{http.MethodGet, credID + "/edit", write, del},
		{http.MethodPost, credID + "/edit", write, del},
		{http.MethodGet, credID + "/delete", del, write},
		{http.MethodPost, credID + "/delete", del, write},
		{http.MethodPost, accountID + "/delete", del, write},
	} {
		assert.Equalf(t, http.StatusForbidden, serve(route.lacking, route.method, route.target),
			"%s %s must be refused without the right", route.method, route.target)
		assert.NotEqualf(t, http.StatusForbidden, serve(route.perm, route.method, route.target),
			"%s %s must be served with the right", route.method, route.target)
	}
}

func TestKitStaticRouter(t *testing.T) {
	t.Parallel()

	router := mux.NewRouter()
	MakeKitStaticRouter(router.PathPrefix(constants.WebuiPrefix).Subrouter())

	for _, file := range []string{"tokens.css", "kit.css", "kit.js", "icons.svg", "fonts/lato-400.woff2"} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, constants.WebuiPrefix+constants.KitStaticPrefix+file, nil))

		assert.Equalf(t, http.StatusOK, rec.Code, "GET %s", file)
		assert.NotEmptyf(t, rec.Body.Bytes(), "GET %s", file)
	}
}
