package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// The host the requests of the test are sent to, and another origin.
const (
	originHost  = "gateway.test:8080"
	originOwn   = "http://gateway.test:8080"
	originOther = "http://other.test"
)

// writeRoute is a route of the web interface that answers a method which may
// change state, with a path that matches it.
type writeRoute struct{ method, path string }

// webInterfaceWrites walks the routes of the administration service and
// returns, for every route under the prefix of the web interface, each method
// that is not safe (a route without methods answers them all: POST stands
// for them), with a path that matches the route.
func webInterfaceWrites(t *testing.T, router *mux.Router) []writeRoute {
	t.Helper()

	variable := regexp.MustCompile(`\{[^{}]*\}`)
	safe := []string{http.MethodGet, http.MethodHead, http.MethodOptions}

	var writes []writeRoute

	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		// A route without a handler is the prefix of a group of routes.
		template, err := route.GetPathTemplate()
		if err != nil || route.GetHandler() == nil || !strings.HasPrefix(template, gui.Prefix) {
			return nil //nolint:nilerr // a route without a path is not one of the web interface
		}

		methods, err := route.GetMethods()
		if err != nil {
			methods = []string{http.MethodPost}
		}

		path := variable.ReplaceAllString(template, "1")

		for _, method := range methods {
			if write := (writeRoute{method, path}); !slices.Contains(safe, method) && !slices.Contains(writes, write) {
				writes = append(writes, write)
			}
		}

		return nil
	}))

	return writes
}

// sharedRoutes returns the routes of the web interface that both a safe
// method and another one reach: those without methods, and those that name
// both kinds.
func sharedRoutes(t *testing.T, router *mux.Router) []string {
	t.Helper()

	safe := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	isSafe := func(method string) bool { return slices.Contains(safe, method) }

	var shared []string

	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		template, err := route.GetPathTemplate()
		if err != nil || route.GetHandler() == nil || !strings.HasPrefix(template, gui.Prefix) {
			return nil //nolint:nilerr // a route without a path is not one of the web interface
		}

		methods, err := route.GetMethods()
		isWrite := func(method string) bool { return !isSafe(method) }
		if err != nil || (slices.ContainsFunc(methods, isSafe) && slices.ContainsFunc(methods, isWrite)) {
			shared = append(shared, template)
		}

		return nil
	}))

	return shared
}

// The origin of a request that may change state is checked once, for every
// route of the web interface: a request from another origin is refused,
// whichever of the two ways the browser says so; a request of the interface
// itself, and one that says nothing (a script, a test), go through. The REST
// API is left alone.
func TestWebInterfaceOriginCheck(t *testing.T) {
	var logged bytes.Buffer

	const (
		username = "origin_check_user"
		password = "sesame"
	)

	db := dbtest.TestDatabase(t)

	hash, err := utils.HashPassword(password)
	require.NoError(t, err)
	require.NoError(t, db.Insert(&model.User{Username: username, PasswordHash: hash, Permissions: model.PermAll}).Run())

	backend := &log.Backend{Output: &logged, Level: log.LevelDebug}
	router := makeRouter(backend.NewLogger("admin"), db)

	serve := func(method, path string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://"+originHost+path, strings.NewReader("a=b"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		for name, value := range headers {
			req.Header.Set(name, value)
		}

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		return rec
	}

	writes := webInterfaceWrites(t, router)
	require.Greater(t, len(writes), 80, "the write routes of the web interface")
	require.Contains(t, writes, writeRoute{http.MethodPost, gui.Prefix + "/login"})
	require.Contains(t, writes, writeRoute{http.MethodPost, gui.Prefix + "/logout"})
	require.Contains(t, writes, writeRoute{http.MethodPost, gui.Prefix + "/servers/1/delete"})

	for _, write := range writes {
		name := write.method + " " + write.path

		assert.Equalf(t, http.StatusForbidden, serve(write.method, write.path,
			map[string]string{"Sec-Fetch-Site": "cross-site"}).Code, "%s, said cross-site by the browser", name)
		assert.Equalf(t, http.StatusForbidden, serve(write.method, write.path,
			map[string]string{"Sec-Fetch-Site": "same-site"}).Code, "%s, from another host of the same site", name)
		assert.Equalf(t, http.StatusForbidden, serve(write.method, write.path,
			map[string]string{"Origin": originOther}).Code, "%s, an origin that is not the host", name)

		assert.NotEqualf(t, http.StatusForbidden, serve(write.method, write.path,
			map[string]string{"Sec-Fetch-Site": "same-origin"}).Code, "%s, said same-origin by the browser", name)
		assert.NotEqualf(t, http.StatusForbidden, serve(write.method, write.path,
			map[string]string{"Origin": originOwn}).Code, "%s, the origin of the host", name)
		assert.NotEqualf(t, http.StatusForbidden, serve(write.method, write.path, nil).Code,
			"%s, neither header", name)
	}

	// No route changes state on a safe method. The routes a safe method and
	// another one both reach are known: the sign-in page, whose GET shows
	// the form; the routes without methods, which redirect (the root, the
	// former addresses) or serve files. They are read with a session, past
	// the redirection to the sign-in page that any screen gets without one.
	signIn := httptest.NewRequest(http.MethodPost, "http://"+originHost+gui.Prefix+"/login",
		strings.NewReader("username="+username+"&password="+password))
	signIn.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	signedIn := httptest.NewRecorder()
	router.ServeHTTP(signedIn, signIn)

	sessionCookies := signedIn.Result().Cookies()
	require.NotEmpty(t, sessionCookies, "the user is signed in")

	session := map[string]string{"Cookie": sessionCookies[0].String()}
	require.Equal(t, http.StatusOK, serve(http.MethodGet, gui.Prefix+"/home", session).Code)

	for _, shared := range sharedRoutes(t, router) {
		switch {
		case shared == gui.Prefix+"/login":
		case strings.HasPrefix(shared, gui.Prefix+"/static"):
		default:
			rec := serve(http.MethodGet, shared, session)
			assert.Truef(t, rec.Code == http.StatusFound || rec.Code == http.StatusMovedPermanently,
				"GET %s is reached by every method: it must only redirect, and answers %d", shared, rec.Code)
			assert.NotContainsf(t, rec.Header().Get("Location"), "/login",
				"GET %s is answered by its handler, not by the sign-in guard", shared)
		}
	}

	// A safe method is never refused: a link from another site opens a page.
	assert.NotEqual(t, http.StatusForbidden,
		serve(http.MethodGet, gui.Prefix+"/login", map[string]string{"Sec-Fetch-Site": "cross-site"}).Code)

	// The REST API has its own authentication, and no session cookie: the
	// check leaves it alone.
	assert.Equal(t, http.StatusUnauthorized,
		serve(http.MethodPost, rest.Prefix+"/servers", map[string]string{"Sec-Fetch-Site": "cross-site"}).Code)

	// A refusal says why, in the language of the request, and leaves a
	// warning in the log with what was received.
	for lang, text := range map[string]string{"en": "reverse proxy", "fr": "mandataire"} {
		rec := serve(http.MethodPost, gui.Prefix+"/login?language="+lang, map[string]string{"Origin": originOther})
		require.Equal(t, http.StatusForbidden, rec.Code)
		assert.Containsf(t, rec.Body.String(), text, "the refusal names the likely cause (%s)", lang)
		assert.Containsf(t, rec.Body.String(), "Host", "the refusal names the header (%s)", lang)
	}

	assert.Contains(t, logged.String(), "WARNING")
	assert.Contains(t, logged.String(), originOther, "the origin received")
	assert.Contains(t, logged.String(), originHost, "the host received")

	// What a refused request sent is written to the log within a bound: a
	// request that needs no session does not fill the log.
	logged.Reset()

	long := strings.Repeat("a", 20000)
	rec := serve(http.MethodPost, gui.Prefix+"/servers/"+strings.Repeat("1", len(long))+"/delete", map[string]string{
		"Origin": "http://" + long + ".test", "Sec-Fetch-Site": long,
	})
	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.Less(t, logged.Len(), 2000, "the line of a refusal is bounded")
	assert.Contains(t, logged.String(), "http://aaaa", "the start of the origin received")
}
