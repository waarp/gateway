package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestWithPermissions(t *testing.T) {
	t.Parallel()

	serve := func(t *testing.T, perms model.PermsMask) (*httptest.ResponseRecorder, bool) {
		t.Helper()

		called := false
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			called = true

			w.WriteHeader(http.StatusOK)
		})

		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "toto", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/tasks", nil)
		rec := httptest.NewRecorder()

		withPermissions(model.PermRulesWrite, next).ServeHTTP(rec, req)

		return rec, called
	}

	t.Run("Given a user lacking the required permission", func(t *testing.T) {
		t.Parallel()

		rec, called := serve(t, model.PermRulesRead)

		assert.False(t, called, "the protected handler must not be executed")
		require.Equal(t, http.StatusForbidden, rec.Code)
		assert.Contains(t, rec.Body.String(), "You are not authorized to access this page.")
	})

	t.Run("Given a user holding the required permission", func(t *testing.T) {
		t.Parallel()

		rec, called := serve(t, model.PermRulesRead|model.PermRulesWrite)

		assert.True(t, called, "the protected handler must be executed")
		require.Equal(t, http.StatusOK, rec.Code)
	})
}
