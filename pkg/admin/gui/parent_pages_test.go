package gui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// newParentPagesTestRouter returns the WebUI pages router, as reached by an
// authenticated user holding all permissions.
func newParentPagesTestRouter(t *testing.T, db *database.DB) http.Handler {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("webui_password"), bcrypt.MinCost)
	require.NoError(t, err)

	user := &model.User{Username: "webui_admin", PasswordHash: string(hash), Permissions: model.PermAll}
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

	RouterPages(secureRouter, db, testhelpers.GetTestLogger(t))

	return router
}

func TestParentPagesWithoutParent(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	router := newParentPagesTestRouter(t, db)

	pages := map[string]string{
		"/partner_authentication":        "/partner_management",
		"/remote_account_management":     "/partner_management",
		"/remote_account_authentication": "/partner_management",
		"/server_authentication":         "/server_management",
		"/local_account_management":      "/server_management",
		"/local_account_authentication":  "/server_management",
		"/management_usage_rights_rules": "/transfer_rules_management",
	}

	queries := map[string]string{
		"missing ID": "",
		"invalid ID": "?partnerID=abc&serverID=abc&ruleID=abc&accountID=abc",
		"unknown ID": "?partnerID=999&serverID=999&ruleID=999&accountID=999",
	}

	for page, parent := range pages {
		for name, query := range queries {
			t.Run(page+" with "+name, func(t *testing.T) {
				t.Parallel()

				rec := httptest.NewRecorder()

				require.NotPanics(t, func() {
					router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, page+query, nil))
				})

				assert.Equal(t, http.StatusSeeOther, rec.Code)
				assert.Equal(t, parent, rec.Header().Get("Location"))
			})
		}
	}
}

func TestUsageRightsPageWithRule(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	router := newParentPagesTestRouter(t, db)

	rule := &model.Rule{Name: "rule", IsSend: true, Path: "rule"}
	require.NoError(t, db.Insert(rule).Run())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/management_usage_rights_rules?ruleID="+utils.FormatInt(rule.ID), nil))

	assert.Equal(t, http.StatusOK, rec.Code)
}
