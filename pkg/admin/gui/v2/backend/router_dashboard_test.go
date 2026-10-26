package backend

import (
	"context"
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

// TestDashboardRouter checks that the dashboard needs no particular right
// (the blocks about the transfers follow the rights inside the page) and
// serves GET only.
func TestDashboardRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	router := mux.NewRouter()
	MakeDashboardRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(perms model.PermsMask, method, target string) *httptest.ResponseRecorder {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "reader", Owner: "gw-test", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec
	}

	t.Run("Given a GET request from a user without any permission", func(t *testing.T) {
		t.Parallel()

		rec := serve(0, http.MethodGet, "/dashboard")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `id="tdb-body"`)
		assert.Contains(t, rec.Body.String(), "Services of node gw-test")
		assert.NotContains(t, rec.Body.String(), `id="tdb-transfers"`, "no transfers without the right to read them")
	})

	t.Run("Given a GET request from a user who reads the transfers", func(t *testing.T) {
		t.Parallel()

		rec := serve(model.PermTransfersRead, http.MethodGet, "/dashboard")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `id="tdb-transfers"`)
	})

	t.Run("Given a request for the body fragment", func(t *testing.T) {
		t.Parallel()

		rec := serve(model.PermTransfersRead, http.MethodGet, "/dashboard?fragment=body")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), "<html")
		assert.Contains(t, rec.Body.String(), `id="tdb-body"`)
	})

	t.Run("Given a POST request", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, http.StatusMethodNotAllowed, serve(model.PermAll, http.MethodPost, "/dashboard").Code,
			"the screen has no action of its own")
	})
}
