package gui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// makeTestPagesRouter returns the WebUI pages router, as an authenticated user
// with the given permissions would reach it.
func makeTestPagesRouter(t *testing.T, db *database.DB, perms model.PermsMask) http.Handler {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("webui_password"), bcrypt.MinCost)
	require.NoError(t, err)

	user := &model.User{Username: "webui_user", PasswordHash: string(hash), Permissions: perms}
	require.NoError(t, db.Insert(user).Run())

	router := mux.NewRouter()
	secureRouter := router.PathPrefix("/").Subrouter()
	secureRouter.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ContextUserKey, user)
			ctx = context.WithValue(ctx, ContextLanguageKey, "en")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	RouterAutocompletions(secureRouter, db)
	RouterPages(secureRouter, db, testhelpers.GetTestLogger(t))

	return router
}

func postForm(router http.Handler, target string, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	return rec
}

func TestWebUIPagesRequirePermissions(t *testing.T) {
	t.Parallel()

	serve := func(router http.Handler, method, target string) int {
		req := httptest.NewRequest(method, target, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		return rec.Code
	}

	t.Run("Given a user without any permission", func(t *testing.T) {
		t.Parallel()

		// Building the router also checks that every page has declared permissions.
		router := makeTestPagesRouter(t, dbtest.TestDatabase(t), 0)

		for path, perms := range webuiPermissions {
			if perms.read != 0 {
				assert.Equal(t, http.StatusForbidden, serve(router, http.MethodGet, path),
					"GET %s must be refused", path)
			} else {
				assert.NotEqual(t, http.StatusForbidden, serve(router, http.MethodGet, path),
					"GET %s only requires authentication", path)
			}

			if perms.write != 0 {
				assert.Equal(t, http.StatusForbidden, serve(router, http.MethodPost, path),
					"POST %s must be refused", path)
			}
		}
	})

	t.Run("Given a user with all permissions", func(t *testing.T) {
		t.Parallel()

		user := &model.User{Username: "admin", Permissions: model.PermAll}

		for path, perms := range webuiPermissions {
			for _, method := range []string{http.MethodGet, http.MethodPost} {
				called := false
				handler := withPagePermissions(perms, func(http.ResponseWriter, *http.Request) {
					called = true
				})

				ctx := context.WithValue(context.Background(), ContextUserKey, user)
				ctx = context.WithValue(ctx, ContextLanguageKey, "en")
				req := httptest.NewRequestWithContext(ctx, method, path, nil)
				handler(httptest.NewRecorder(), req)

				assert.True(t, called, "%s %s must reach the page handler", method, path)
			}
		}
	})
}

func TestUserManagementPagePermissions(t *testing.T) {
	t.Parallel()

	form := url.Values{
		"newUserUsername":           {"created_user"},
		"newUserPassword":           {"sesame"},
		"newUserPermissionsUsers[]": {"r", "w", "d"},
	}

	t.Run("Given a user without the users write permission", func(t *testing.T) {
		t.Parallel()

		db := dbtest.TestDatabase(t)
		router := makeTestPagesRouter(t, db, model.PermUsersRead)

		rec := postForm(router, "/user_management", form)

		assert.Equal(t, http.StatusForbidden, rec.Code)

		var created model.User
		err := db.Get(&created, "username=?", "created_user").Owner().Run()
		assert.True(t, database.IsNotFound(err), "no user must have been created")
	})

	t.Run("Given a user with the users write permission", func(t *testing.T) {
		t.Parallel()

		db := dbtest.TestDatabase(t)
		router := makeTestPagesRouter(t, db, model.PermUsersRead|model.PermUsersWrite)

		rec := postForm(router, "/user_management", form)

		assert.Equal(t, http.StatusSeeOther, rec.Code)

		var created model.User
		require.NoError(t, db.Get(&created, "username=?", "created_user").Owner().Run())
	})

	t.Run("Given a user without the users read permission", func(t *testing.T) {
		t.Parallel()

		db := dbtest.TestDatabase(t)
		router := makeTestPagesRouter(t, db, model.PermTransfersRead)

		req := httptest.NewRequest(http.MethodGet, "/user_management", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
