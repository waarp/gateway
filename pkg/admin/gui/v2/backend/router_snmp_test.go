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
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// TestSNMPRouter checks the right each route of the "SNMP" screen requires:
// the rights on the administration, as the REST API of SNMP (the agent is
// disabled with the write right, as DELETE /api/snmp/server).
func TestSNMPRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	monitor := &snmp.MonitorConfig{Name: "supervision", Version: snmp.Version2, UDPAddress: "127.0.0.1:162"}
	require.NoError(t, db.Insert(monitor).Run())
	require.NoError(t, db.Insert(&snmp.ServerConfig{LocalUDPAddress: "127.0.0.1:16161"}).Run())

	router := mux.NewRouter()
	MakeSNMPRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(perms model.PermsMask, method, target string) int {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "tester", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec.Code
	}

	id := fmt.Sprintf("/snmp/monitors/%d", monitor.ID)
	read := model.PermAdminRead
	write := read | model.PermAdminWrite
	del := read | model.PermAdminDelete

	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, "/snmp"))
	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, id))
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/snmp/monitors/999"))
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/snmp/monitors/abc"), "an identifier is a number")
	assert.Equal(t, http.StatusMethodNotAllowed, serve(write, http.MethodGet, "/snmp/test"), "the test is a POST")

	// The deletions come last, once the other routes have answered.
	for _, route := range []struct {
		method, target string
		perm           model.PermsMask
		lacking        model.PermsMask
	}{
		{http.MethodGet, "/snmp", read, 0},
		{http.MethodGet, id, read, 0},
		{http.MethodGet, "/snmp/monitors/new", write, del},
		{http.MethodPost, "/snmp/monitors/new", write, del},
		{http.MethodGet, id + "/edit", write, del},
		{http.MethodPost, id + "/edit", write, del},
		{http.MethodGet, "/snmp/agent/edit", write, del},
		{http.MethodPost, "/snmp/agent/edit", write, del},
		{http.MethodPost, "/snmp/test", write, del},
		{http.MethodGet, id + "/delete", del, write},
		{http.MethodPost, id + "/delete", del, write},
		{http.MethodGet, "/snmp/agent/delete", write, del},
		{http.MethodPost, "/snmp/agent/delete", write, del},
	} {
		assert.Equalf(t, http.StatusForbidden, serve(route.lacking, route.method, route.target),
			"%s %s must be refused without the right", route.method, route.target)
		assert.NotEqualf(t, http.StatusForbidden, serve(route.perm, route.method, route.target),
			"%s %s must be served with the right", route.method, route.target)
	}
}
