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

// TestTransfersRouter checks the right each route of the "Transfer
// monitoring" screen requires: reading the transfers for the list and the
// sheet, writing them for every action and its page, as the REST API.
func TestTransfersRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)

	hist := &model.HistoryEntry{
		Identifier: model.ID(77), RemoteTransferID: "r-77", IsSend: true, Rule: "envoi", Account: "prod",
		Agent: "banque-demo", Client: "cli-sftp", Protocol: "sftp", SrcFilename: "a.txt", DestFilename: "a.txt",
		Start: time.Date(2026, 10, 21, 9, 0, 0, 0, time.Local), Stop: time.Date(2026, 10, 21, 9, 0, 1, 0, time.Local),
		Status: types.StatusDone,
	}
	require.NoError(t, db.Insert(hist).Run())

	router := mux.NewRouter()
	MakeTransfersRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(perms model.PermsMask, method, target string) int {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "tester", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec.Code
	}

	id := fmt.Sprintf("/transfers/%d", hist.ID)
	read := model.PermTransfersRead
	write := read | model.PermTransfersWrite
	others := model.PermServersRead | model.PermPartnersRead | model.PermRulesRead | model.PermAdminRead

	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, "/transfers"))
	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, id))
	assert.Equal(t, http.StatusForbidden, serve(others, http.MethodGet, "/transfers"),
		"the rights on the other objects do not open it")
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/transfers/999"))
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/transfers/abc"), "an identifier is a number")
	assert.Equal(t, http.StatusMethodNotAllowed, serve(write, http.MethodGet, id+"/pause"), "every action is a POST")

	for _, route := range []struct {
		method, target string
		perm           model.PermsMask
	}{
		{http.MethodGet, "/transfers", read},
		{http.MethodGet, id, read},
		{http.MethodGet, "/transfers/new", write},
		{http.MethodPost, "/transfers/new", write},
		{http.MethodGet, "/transfers/preregister", write},
		{http.MethodPost, "/transfers/preregister", write},
		{http.MethodPost, id + "/pause", write},
		{http.MethodPost, id + "/resume", write},
		{http.MethodGet, id + "/cancel", write},
		{http.MethodPost, id + "/cancel", write},
		{http.MethodGet, id + "/retry", write},
		{http.MethodPost, id + "/retry", write},
	} {
		lacking := model.PermsMask(0)
		if route.perm == write {
			lacking = read | others
		}

		assert.Equalf(t, http.StatusForbidden, serve(lacking, route.method, route.target),
			"%s %s must be refused without the right", route.method, route.target)
		assert.NotEqualf(t, http.StatusForbidden, serve(route.perm, route.method, route.target),
			"%s %s must be served with the right", route.method, route.target)
	}
}
