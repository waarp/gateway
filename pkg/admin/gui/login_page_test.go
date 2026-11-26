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

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	loginTestUser     = "login_page_user"
	loginTestPassword = "sesame"
)

// loginSuite serves the sign-in page and the sign-out, as the router mounts
// them, for a database holding one user.
type loginSuite struct {
	db     *database.DB
	router *mux.Router
}

func newLoginSuite(t *testing.T) *loginSuite {
	t.Helper()

	db := dbtest.TestDatabase(t)

	hash, err := utils.HashPassword(loginTestPassword)
	require.NoError(t, err)
	require.NoError(t, db.Insert(&model.User{
		Username: loginTestUser, PasswordHash: hash, Permissions: model.PermAll,
	}).Run())

	logger := testhelpers.GetTestLogger(t)
	router := mux.NewRouter()
	sub := router.PathPrefix(Prefix).Subrouter()
	sub.HandleFunc("/login", loginPage(logger, db)).Methods("GET", "POST")
	sub.HandleFunc("/logout", logout()).Methods("GET", "POST")

	secure := sub.PathPrefix("/").Subrouter()
	secure.Use(AuthenticationMiddleware(logger, db))
	secure.HandleFunc("/home", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	// A page of the reworked interface lives under a sub-path: the guard must
	// lead it to the same sign-in page.
	secure.HandleFunc("/transfers/new", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	return &loginSuite{db: db, router: router}
}

// do sends a request in the given language, with a form for a POST and the
// given cookie when set.
func (s *loginSuite) do(t *testing.T, lang, method, target string, form url.Values, cookie string) *httptest.ResponseRecorder {
	t.Helper()

	ctx := context.WithValue(context.Background(), ContextLanguageKey, lang)

	var req *http.Request
	if form != nil {
		req = httptest.NewRequestWithContext(ctx, method, target, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequestWithContext(ctx, method, target, nil)
	}

	req.RemoteAddr = "10.0.0.9:1234"

	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: tokenCookieName, Value: cookie})
	}

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

func credentials(user, password string) url.Values {
	return url.Values{"username": {user}, "password": {password}}
}

// sessionCookie returns the session cookie set by a response, empty without one.
func sessionCookie(rec *httptest.ResponseRecorder) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == tokenCookieName {
			return c.Value
		}
	}

	return ""
}

// The subtests share the suite and sign in as the same user, whose sessions
// a new sign-in ends: they run one after the other.
func TestLoginPage(t *testing.T) {
	t.Parallel()

	s := newLoginSuite(t)

	t.Run("the page in French and in English", func(t *testing.T) {
		rec := s.do(t, "fr", http.MethodGet, "/webui/login", nil, "")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		for _, want := range []string{
			`<html lang="fr">`, `<title>Se connecter · Waarp Gateway</title>`, `<body class="w-plain"`,
			`<main class="w-plain__main" id="contenu" tabindex="-1">`, `<h1 class="w-plain__title">Se connecter</h1>`,
			`<label class="w-field__label" for="username">Identifiant</label>`,
			`id="username" name="username" value="" required autofocus autocomplete="username"`,
			`<label class="w-field__label" for="password">Mot de passe</label>`,
			`type="password" id="password" name="password" autocomplete="current-password" required`,
			`data-password-toggle`, `aria-label="Afficher le mot de passe"`,
			`<button class="w-btn w-btn--primary w-plain__submit" type="submit">Se connecter</button>`,
			`/webui/static/gui/waarp-banner.png`, `/webui/static/gui/kit.css`,
			`<a href="/webui/login?language=fr" lang="fr" aria-current="true">Français</a>`,
			`<a href="/webui/login?language=en" lang="en">English</a>`,
		} {
			assert.Contains(t, body, want)
		}

		for _, forbidden := range []string{`class="w-skip"`, `<header`, `<aside`, `role="alert"`, `role="status"`, `leave.js`} {
			assert.NotContains(t, body, forbidden)
		}

		body = s.do(t, "en", http.MethodGet, "/webui/login", nil, "").Body.String()
		assert.Contains(t, body, `<html lang="en">`)
		assert.Contains(t, body, `<title>Sign in · Waarp Gateway</title>`)
		assert.Contains(t, body, `for="username">Username</label>`)
	})

	t.Run("the language switcher keeps the page asked for", func(t *testing.T) {
		body := s.do(t, "fr", http.MethodGet, "/webui/login?redirect=%2Fwebui%2Fpartners%3Fpage%3D2", nil, "").Body.String()
		assert.Contains(t, body, `<a href="/webui/login?language=en&amp;redirect=%2Fwebui%2Fpartners%3Fpage%3D2" lang="en">English</a>`)
	})

	t.Run("the reason the user is on the page", func(t *testing.T) {
		body := s.do(t, "fr", http.MethodGet, "/webui/login?reason=expired", nil, "").Body.String()
		assert.Contains(t, body, `role="status"`)
		assert.Contains(t, body, "Votre session a pris fin. Reconnectez-vous pour continuer.")

		body = s.do(t, "en", http.MethodGet, "/webui/login?reason=signed_out", nil, "").Body.String()
		assert.Contains(t, body, "You are signed out.")

		body = s.do(t, "fr", http.MethodGet, "/webui/login?reason=whatever", nil, "").Body.String()
		assert.NotContains(t, body, `role="status"`, "an unknown reason shows nothing")
	})

	t.Run("a sign-in opens the session and leads to the home page", func(t *testing.T) {
		rec := s.do(t, "fr", http.MethodPost, "/webui/login", credentials(loginTestUser, loginTestPassword), "")
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/home", rec.Header().Get("Location"))

		cookie := sessionCookie(rec)
		require.NotEmpty(t, cookie, "the session cookie is set")

		assert.Equal(t, http.StatusOK, s.do(t, "fr", http.MethodGet, "/webui/home", nil, cookie).Code,
			"the session cookie opens the interface")
	})

	t.Run("a sign-in leads to the page asked for, within the interface only", func(t *testing.T) {
		// The browser gets the address resolved against the page: a path.
		for target, want := range map[string]string{
			"/webui/partners?page=2":        "/webui/partners?page=2",
			"/webui/":                       "/webui/home",
			"/webui/login?reason=expired":   "/webui/home",
			"/webui/logout":                 "/webui/home",
			"/webui//logout/":               "/webui/home",
			"https://example.com/webui/x":   "/webui/home",
			"//example.com/webui/x":         "/webui/home",
			"/api/users":                    "/webui/home",
			"/webuiX/partners":              "/webui/home",
			"":                              "/webui/home",
			"/webui//example.com":           "/webui/example.com",
			"/webui/transfers/12#tab-tasks": "/webui/transfers/12#tab-tasks",
		} {
			query := url.Values{redirectParam: {target}}
			rec := s.do(t, "fr", http.MethodPost, "/webui/login?"+query.Encode(),
				credentials(loginTestUser, loginTestPassword), "")
			require.Equal(t, http.StatusSeeOther, rec.Code, target)
			assert.Equal(t, want, rec.Header().Get("Location"), "redirect=%q", target)
		}
	})

	t.Run("a refused sign-in shows the form again", func(t *testing.T) {
		rec := s.do(t, "fr", http.MethodPost, "/webui/login", credentials(loginTestUser, "wrong"), "")
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Empty(t, sessionCookie(rec), "no session")

		body := rec.Body.String()
		assert.Contains(t, body, `role="alert"`)
		assert.Contains(t, body, "Identifiant ou mot de passe incorrect. Vérifiez votre saisie, puis réessayez.")
		assert.Contains(t, body, `name="username" value="login_page_user"`, "the username is kept")
		assert.NotContains(t, body, "wrong", "the password never comes back")

		body = s.do(t, "en", http.MethodPost, "/webui/login", credentials("nobody", "wrong"), "").Body.String()
		assert.Contains(t, body, "Incorrect username or password. Check what you typed, then try again.",
			"the same sentence for an unknown account")
	})

	t.Run("the sign-out ends the session and says so", func(t *testing.T) {
		cookie := sessionCookie(s.do(t, "fr", http.MethodPost, "/webui/login",
			credentials(loginTestUser, loginTestPassword), ""))
		require.NotEmpty(t, cookie)

		rec := s.do(t, "fr", http.MethodPost, "/webui/logout", url.Values{}, cookie)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/login?reason=signed_out", rec.Header().Get("Location"))

		var cleared bool
		for _, c := range rec.Result().Cookies() {
			if c.Name == tokenCookieName && c.MaxAge < 0 {
				cleared = true
			}
		}

		assert.True(t, cleared, "the session cookie is cleared")

		rec = s.do(t, "fr", http.MethodGet, "/webui/home", nil, cookie)
		require.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/webui/login?reason=expired&redirect=%2Fwebui%2Fhome", rec.Header().Get("Location"),
			"the old cookie is refused and said so")
	})

	t.Run("without any cookie, the interface leads to the sign-in page without a reason", func(t *testing.T) {
		rec := s.do(t, "fr", http.MethodGet, "/webui/home", nil, "")
		require.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/webui/login?redirect=%2Fwebui%2Fhome", rec.Header().Get("Location"))
	})

	t.Run("a page under a sub-path leads to the sign-in page of the interface, not to one of its own", func(t *testing.T) {
		rec := s.do(t, "fr", http.MethodGet, "/webui/transfers/new", nil, "")
		require.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/webui/login?redirect=%2Fwebui%2Ftransfers%2Fnew", rec.Header().Get("Location"),
			"the sign-in page is at the root of the interface, whatever the directory of the page asked for")

		rec = s.do(t, "fr", http.MethodGet, "/webui/transfers/new", nil, "stale-token")
		require.Equal(t, http.StatusFound, rec.Code)
		assert.Equal(t, "/webui/login?reason=expired&redirect=%2Fwebui%2Ftransfers%2Fnew", rec.Header().Get("Location"))
	})
}
