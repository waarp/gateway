package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestAuthenticationThrottle(t *testing.T) {
	t.Parallel()

	const password = "sesame"

	db := dbtest.TestDatabase(t)
	db.Config.Security = conf.SecurityConfig{
		MaxAuthFailures: 3, AuthFailureWindow: 5 * time.Minute,
		AuthLockDuration: time.Minute, AuthMaxLockDuration: 30 * time.Minute,
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	require.NoError(t, db.Insert(&model.User{
		Username: "rest_throttle_user", PasswordHash: string(hash), Permissions: model.PermAll,
	}).Run())

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := AuthenticationMiddleware(testhelpers.GetTestLogger(t), db).Middleware(ok)

	call := func(addr, pswd string) int {
		req := httptest.NewRequest(http.MethodGet, "/api/about", nil)
		req.RemoteAddr = addr
		req.SetBasicAuth("rest_throttle_user", pswd)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		return rec.Code
	}

	for range 3 {
		assert.Equal(t, http.StatusUnauthorized, call("10.0.0.1:1234", "wrong"))
	}

	assert.Equal(t, http.StatusUnauthorized, call("10.0.0.1:1234", password),
		"valid credentials refused while the address is locked")
	assert.Equal(t, http.StatusOK, call("10.0.0.2:1234", password),
		"valid credentials accepted from another address")
}
