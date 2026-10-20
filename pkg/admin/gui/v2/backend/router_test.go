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
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
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
