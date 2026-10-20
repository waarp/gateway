package me

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

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	oldPassword = "old-pa55word"
	newPassword = "new-pa55word"
)

type suite struct {
	db     *database.DB
	router *mux.Router
	user   *model.User
}

func newSuite(t *testing.T, perms model.PermsMask) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}

	hash, err := utils.HashPassword(oldPassword)
	require.NoError(t, err)

	s.user = &model.User{Username: "tester", PasswordHash: hash, Permissions: perms}
	require.NoError(t, s.db.Insert(s.user).Run())

	h := New(s.db, testhelpers.GetTestLogger(t))
	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/account").Methods(http.MethodGet).HandlerFunc(h.Page)
	r.Path("/account/password").Methods(http.MethodPost).HandlerFunc(h.ChangePassword)

	return s
}

// do sends a request as the user of the suite, read again from the database
// (as the session middleware does).
func (s *suite) do(t *testing.T, lang, method, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	var user model.User
	require.NoError(t, s.db.Get(&user, "id=?", s.user.ID).Run())

	ctx := context.WithValue(context.Background(), constants.ContextUserKey, &user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, lang)

	var req *http.Request
	if form != nil {
		req = httptest.NewRequestWithContext(ctx, method, target, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequestWithContext(ctx, method, target, nil)
	}

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

func (s *suite) hash(t *testing.T) string {
	t.Helper()

	var user model.User
	require.NoError(t, s.db.Get(&user, "id=?", s.user.ID).Run())

	return user.PasswordHash
}

func TestPage(t *testing.T) {
	t.Parallel()

	t.Run("Given a user with the servers rights and the users read right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, model.PermServersRead|model.PermServersWrite|model.PermServersDelete|model.PermUsersRead)
		rec := s.do(t, "fr", http.MethodGet, "/webui/account", nil)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Mon compte · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<a href="/webui/home">Accueil</a>`)
		assert.Contains(t, body, `<dt>Identifiant</dt><dd><code class="w-code">tester</code></dd>`)
		assert.Contains(t, body, `<dt>Instance</dt><dd><code class="w-code">gw-test</code></dd>`)
		assert.Contains(t, body, `<td><span class="w-cell-title">Serveurs et clients locaux</span></td>
            <td>Oui</td>
            <td>Oui</td>
            <td>Oui</td>`)
		assert.Contains(t, body, `<td><span class="w-cell-title">Utilisateurs</span></td>
            <td>Oui</td>
            <td><span class="w-muted">Non</span></td>
            <td><span class="w-muted">Non</span></td>`)
		assert.Contains(t, body, `<td><span class="w-cell-title">Transferts</span></td>
            <td><span class="w-muted">Non</span></td>`)
		assert.Contains(t, body, `<form class="w-form" method="post" action="/webui/account/password" novalidate>`)
		assert.Contains(t, body, `type="password" id="current" name="current" required autocomplete="current-password"`)
		assert.Contains(t, body, `type="password" id="new" name="new" required autocomplete="new-password"`)
		assert.Contains(t, body, `href="/webui/account"`, "the account entry of the menu and the cancel link")
		assert.NotContains(t, body, "w-btn--primary\" href", "no coloured link")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a user without any right, in English, after a change", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, 0)
		body := s.do(t, "en", http.MethodGet, "/webui/account?done=password", nil).Body.String()
		assert.Contains(t, body, "<title>My account · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "Password changed")
		assert.Equal(t, 18, strings.Count(body, `<span class="w-muted">No</span>`), "six domains, no right at all")
		assert.Contains(t, body, "Your rights are set by an administrator")
	})
}

func TestChangePassword(t *testing.T) {
	t.Parallel()

	t.Run("Given the right current password and a confirmed new one", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, 0)
		before := s.hash(t)

		rec := s.do(t, "fr", http.MethodPost, "/webui/account/password",
			url.Values{"current": {oldPassword}, "new": {newPassword}, "confirm": {newPassword}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, "/webui/account?done=password", rec.Header().Get("Location"))

		after := s.hash(t)
		assert.NotEqual(t, before, after)
		assert.True(t, utils.CheckPassword(utils.BcryptRounds, after, newPassword), "the new password opens")
		assert.False(t, utils.CheckPassword(utils.BcryptRounds, after, oldPassword), "the old one no longer does")

		var user model.User
		require.NoError(t, s.db.Get(&user, "id=?", s.user.ID).Run())
		assert.Equal(t, "tester", user.Username, "nothing else changed")
		assert.Equal(t, model.PermsMask(0), user.Permissions)
	})

	t.Run("Given a wrong current password", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, 0)
		rec := s.do(t, "fr", http.MethodPost, "/webui/account/password",
			url.Values{"current": {"not-the-one"}, "new": {newPassword}, "confirm": {newPassword}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Le mot de passe actuel n&#39;est pas le bon.")
		assert.Contains(t, body, "Le mot de passe n&#39;a pas été modifié.")
		assert.NotContains(t, body, "not-the-one", "no password is rendered back")
		assert.NotContains(t, body, newPassword)
		assert.True(t, utils.CheckPassword(utils.BcryptRounds, s.hash(t), oldPassword), "the password is unchanged")
	})

	t.Run("Given a new password that is empty or not confirmed", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, 0)
		rec := s.do(t, "fr", http.MethodPost, "/webui/account/password",
			url.Values{"current": {oldPassword}, "new": {""}, "confirm": {""}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez le nouveau mot de passe.")

		rec = s.do(t, "en", http.MethodPost, "/webui/account/password",
			url.Values{"current": {oldPassword}, "new": {newPassword}, "confirm": {"other"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "The two entries of the new password differ.")
		assert.True(t, utils.CheckPassword(utils.BcryptRounds, s.hash(t), oldPassword))
	})
}
