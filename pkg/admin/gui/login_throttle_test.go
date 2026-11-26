package gui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

func TestLoginThrottle(t *testing.T) {
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
		Username: "webui_throttle_user", PasswordHash: string(hash), Permissions: model.PermAll,
	}).Run())

	handler := loginPage(testhelpers.GetTestLogger(t), db)

	login := func(addr, pswd string) int {
		form := url.Values{"username": {"webui_throttle_user"}, "password": {pswd}}
		req := httptest.NewRequestWithContext(
			context.WithValue(context.Background(), ContextLanguageKey, "en"),
			http.MethodPost, "/login", strings.NewReader(form.Encode()),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = addr

		rec := httptest.NewRecorder()
		handler(rec, req)

		return rec.Code
	}

	for range 3 {
		assert.Equal(t, http.StatusUnprocessableEntity, login("10.0.0.1:1234", "wrong"), "login page shown again")
	}

	assert.Equal(t, http.StatusUnprocessableEntity, login("10.0.0.1:1234", password),
		"valid credentials refused while the address is locked")
	assert.Equal(t, http.StatusSeeOther, login("10.0.0.2:1234", password),
		"valid credentials accepted from another address")
}
