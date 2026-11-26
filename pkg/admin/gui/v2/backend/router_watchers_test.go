package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// TestWatchersRouter checks the right each route of the "Partner watchers"
// screen requires: the rights on the servers, as the REST API of the
// watchers.
func TestWatchersRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)

	partner := &model.RemoteAgent{Name: "part-sftp", Protocol: "sftp", Address: types.Addr("127.0.0.1", 2222)}
	require.NoError(t, db.Insert(partner).Run())

	account := &model.RemoteAccount{RemoteAgentID: partner.ID, Login: "gw"}
	require.NoError(t, db.Insert(account).Run())

	client := &model.Client{Name: "cli-sftp", Protocol: "sftp"}
	require.NoError(t, db.Insert(client).Run())

	rule := &model.Rule{Name: "recv", IsSend: false, Path: "/recv"}
	require.NoError(t, db.Insert(rule).Run())

	watcher := &model.FileWatcher{
		Flow: "releves", Interval: time.Hour, RemoteAccount: *account, Client: *client, Rule: *rule,
	}
	require.NoError(t, db.Insert(watcher).Run())

	router := mux.NewRouter()
	MakeWatchersRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(perms model.PermsMask, method, target string) int {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "tester", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec.Code
	}

	id := fmt.Sprintf("/watchers/%d", watcher.ID)
	read := model.PermServersRead
	write := read | model.PermServersWrite
	del := read | model.PermServersDelete

	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, "/watchers"))
	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, id))
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/watchers/999"))
	assert.Equal(t, http.StatusNotFound, serve(write, http.MethodGet, "/watchers/999/edit"))
	assert.Equal(t, http.StatusNotFound, serve(write, http.MethodGet, "/watchers/abc/edit"), "an identifier is a number")
	assert.Equal(t, http.StatusMethodNotAllowed, serve(write, http.MethodGet, id+"/start"), "every action is a POST")
	assert.Equal(t, http.StatusMethodNotAllowed, serve(del, http.MethodDelete, id+"/delete"), "every action is a POST")

	// The services of the watchers are not started here: the start, the
	// stop and the run answer with the rights, whatever the outcome.
	for _, route := range []struct {
		method, target string
		perm           model.PermsMask
		lacking        model.PermsMask
	}{
		{http.MethodGet, "/watchers", read, 0},
		{http.MethodGet, id, read, 0},
		{http.MethodGet, "/watchers/new", write, del},
		{http.MethodPost, "/watchers/new", write, del},
		{http.MethodGet, id + "/edit", write, del},
		{http.MethodPost, id + "/edit", write, del},
		{http.MethodPost, id + "/stop", write, del},
		{http.MethodPost, id + "/start", write, del},
		{http.MethodGet, id + "/delete", del, write},
		{http.MethodPost, id + "/delete", del, write},
	} {
		assert.Equalf(t, http.StatusForbidden, serve(route.lacking, route.method, route.target),
			"%s %s must be refused without the right", route.method, route.target)
		assert.NotEqualf(t, http.StatusForbidden, serve(route.perm, route.method, route.target),
			"%s %s must be served with the right", route.method, route.target)
	}
}
