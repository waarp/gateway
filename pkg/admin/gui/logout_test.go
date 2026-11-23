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

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// Signing out changes state: it is a POST. An address of the sign-out saved
// or followed from elsewhere (a GET) changes nothing and leads to the home
// page; the session it carries stays open.
func TestSignOutIsAPost(t *testing.T) {
	const (
		username = "sign_out_user"
		password = "sesame"
	)

	db := dbtest.TestDatabase(t)

	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	require.NoError(t, db.Insert(&model.User{Username: username, PasswordHash: hash, Permissions: model.PermAll}).Run())

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

	signedIn := func() bool { return serve(http.MethodGet, "/webui/home", cookie, nil).Code == http.StatusOK }
	require.True(t, signedIn())

	// A GET: nothing changes, the home page.
	rec := serve(http.MethodGet, "/webui/logout", cookie, nil)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "/webui/home", rec.Header().Get("Location"))
	assert.Equal(t, cookie, sessionCookie(rec), "the session cookie is the same, kept as by any page")
	assert.True(t, signedIn(), "a GET does not end the session")

	// Without a session, the same address leads to the sign-in page, as any
	// screen does.
	rec = serve(http.MethodGet, "/webui/logout", "", nil)
	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "/webui/login")

	// The POST of the chrome ends the session.
	rec = serve(http.MethodPost, "/webui/logout", cookie, url.Values{})
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/webui/login?reason=signed_out", rec.Header().Get("Location"))
	assert.False(t, signedIn(), "the POST ends the session")
}
