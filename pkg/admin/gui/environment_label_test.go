package gui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// instanceBlock returns the part of the header of a page that names the
// instance: its name, and its environment label when it has one.
func instanceBlock(tb testing.TB, page string) string {
	tb.Helper()

	_, after, found := strings.Cut(page, `<div class="w-instance">`)
	require.True(tb, found, "the header names the instance")

	block, _, closed := strings.Cut(after, `</div>`)
	require.True(tb, closed)

	return block
}

// The environment label of the configuration is shown as a badge next
// to the name of the instance, in the header of the pages of a session,
// whatever the language; its text is escaped. Without a label, the header is
// what it was.
func TestEnvironmentLabelInHeader(t *testing.T) {
	const (
		username = "environment_user"
		password = "sesame"
	)

	db := dbtest.TestDatabase(t)

	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	require.NoError(t, db.Insert(&model.User{
		Username: username, PasswordHash: hash, Permissions: model.PermAll,
	}).Run())

	router := mux.NewRouter()
	AddGUIRouter(router.PathPrefix(Prefix).Subrouter(), testhelpers.GetTestLogger(t), db)

	serve := func(method, target, cookie string, form url.Values) *httptest.ResponseRecorder {
		var req *http.Request
		if form != nil {
			req = httptest.NewRequest(method, target, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			req = httptest.NewRequest(method, target, nil)
		}

		req.RemoteAddr = "10.0.0.9:1234"

		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: cookie})
		}

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		return rec
	}

	cookie := sessionCookie(serve(http.MethodPost, "/webui/login", "", credentials(username, password)))
	require.NotEmpty(t, cookie, "the user is signed in")

	home := func(lang string) string {
		rec := serve(http.MethodGet, "/webui/home?language="+lang, cookie, nil)
		require.Equal(t, http.StatusOK, rec.Code)
		require.Contains(t, rec.Body.String(), `<html lang="`+lang+`"`)

		return rec.Body.String()
	}

	t.Run("Without a label", func(t *testing.T) {
		db.Config.Admin.EnvironmentLabel = ""

		block := instanceBlock(t, home("en"))
		assert.Contains(t, block, `<span class="w-instance__name">`)
		assert.NotContains(t, block, "w-badge")
	})

	t.Run("With a label", func(t *testing.T) {
		db.Config.Admin.EnvironmentLabel = conf.EnvironmentLabel("Pré-prod <2>")

		for _, lang := range []string{"en", "fr"} {
			block := instanceBlock(t, home(lang))
			assert.Contains(t, block, `<span class="w-badge w-badge--neutral">Pré-prod &lt;2&gt;</span>`, lang)
			assert.NotContains(t, block, "<2>", "the label is escaped")
		}
	})

	t.Run("On the sign-in page, which names no instance", func(t *testing.T) {
		db.Config.Admin.EnvironmentLabel = conf.EnvironmentLabel("Production")

		rec := serve(http.MethodGet, "/webui/login", "", nil)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.NotContains(t, rec.Body.String(), "Production")
	})
}
