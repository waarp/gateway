package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	headerCSP           = "Content-Security-Policy"
	headerCSPReportOnly = "Content-Security-Policy-Report-Only"
	headerHSTS          = "Strict-Transport-Security"

	headersUser     = "headers"
	headersPassword = "headers_password"
)

// getHeaders asks path to the administration service and returns the status
// and the headers of its answer, redirections not followed.
func getHeaders(tb testing.TB, server *httptest.Server, path string) (int, http.Header) {
	tb.Helper()

	client := server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	req, reqErr := http.NewRequestWithContext(tb.Context(), http.MethodGet, server.URL+path, nil)
	require.NoError(tb, reqErr)
	req.SetBasicAuth(headersUser, headersPassword)

	resp, respErr := client.Do(req)
	require.NoError(tb, respErr)
	require.NoError(tb, resp.Body.Close())

	return resp.StatusCode, resp.Header
}

// The answers of the administration service: a page of the web interface, one
// of its static files, an answer of the REST API, and the answers the router
// gives on its own, which no middleware of the router sees.
func TestSecurityHeaders(t *testing.T) {
	db := dbtest.TestDatabase(t)

	hash, hashErr := utils.HashPassword(headersPassword)
	require.NoError(t, hashErr)
	require.NoError(t, db.Insert(&model.User{
		Username: headersUser, PasswordHash: hash, Permissions: model.PermAll,
	}).Run())

	answers := []struct {
		name, path string
		status     int
	}{
		{name: "a page", path: "/webui/login", status: http.StatusOK},
		{name: "a static file", path: "/webui/static/css/button.css", status: http.StatusOK},
		{name: "an API answer", path: "/api/users", status: http.StatusOK},
		{name: "an API refusal", path: "/api/nowhere", status: http.StatusNotFound},
		{name: "an unknown path", path: "/nowhere", status: http.StatusNotFound},
		{name: "a redirection", path: "/", status: http.StatusPermanentRedirect},
	}

	t.Run("Over plain HTTP, with the default settings", func(t *testing.T) {
		db.Config.Admin = conf.AdminConfig{HSTSMaxAge: time.Hour}
		server := httptest.NewServer(MakeHandler(testhelpers.GetTestLogger(t), db))
		t.Cleanup(server.Close)

		for _, answer := range answers {
			t.Run(answer.name, func(t *testing.T) {
				status, headers := getHeaders(t, server, answer.path)

				require.Equal(t, answer.status, status)
				assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
				assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
				assert.Equal(t, "same-origin", headers.Get("Referrer-Policy"))
				assert.Equal(t, contentSecurityPolicy, headers.Get(headerCSPReportOnly),
					"the policy is sent in report-only mode by default")
				assert.Empty(t, headers.Get(headerCSP), "the policy is not enforced by default")
				assert.Empty(t, headers.Get(headerHSTS), "HTTPS is not asked for over plain HTTP")
			})
		}
	})

	t.Run("Over TLS", func(t *testing.T) {
		for _, test := range []struct {
			name     string
			maxAge   time.Duration
			expected string
		}{
			{name: "without a max age", maxAge: 0, expected: ""},
			{name: "with a max age", maxAge: 365 * 24 * time.Hour, expected: "max-age=31536000"},
		} {
			t.Run(test.name, func(t *testing.T) {
				db.Config.Admin = conf.AdminConfig{HSTSMaxAge: test.maxAge}
				server := httptest.NewTLSServer(MakeHandler(testhelpers.GetTestLogger(t), db))
				t.Cleanup(server.Close)

				for _, answer := range answers {
					status, headers := getHeaders(t, server, answer.path)

					require.Equal(t, answer.status, status, answer.name)
					assert.Equal(t, test.expected, headers.Get(headerHSTS), answer.name)
					assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"), answer.name)
				}
			})
		}
	})

	t.Run("With each mode of the content security policy", func(t *testing.T) {
		for _, test := range []struct {
			mode conf.CSPMode
			sent string
		}{
			{mode: conf.CSPReportOnly, sent: headerCSPReportOnly},
			{mode: conf.CSPEnforce, sent: headerCSP},
			{mode: conf.CSPOff, sent: ""},
		} {
			t.Run(string(test.mode), func(t *testing.T) {
				db.Config.Admin = conf.AdminConfig{ContentSecurityPolicy: test.mode}
				server := httptest.NewServer(MakeHandler(testhelpers.GetTestLogger(t), db))
				t.Cleanup(server.Close)

				_, headers := getHeaders(t, server, "/webui/login")

				for _, header := range []string{headerCSPReportOnly, headerCSP} {
					if header == test.sent {
						assert.Equal(t, contentSecurityPolicy, headers.Get(header))
					} else {
						assert.Empty(t, headers.Get(header))
					}
				}
			})
		}
	})
}

// The policy, directive by directive: any change of it is a decision. It allows
// what the service serves itself, the images embedded in a stylesheet and the
// styles carried by the tags, which the preview of a mail needs. It allows no
// inline script, no inline stylesheet and no text evaluated as code.
func TestContentSecurityPolicy(t *testing.T) {
	directives := map[string]string{}

	for directive := range strings.SplitSeq(contentSecurityPolicy, ";") {
		name, sources, _ := strings.Cut(strings.TrimSpace(directive), " ")
		directives[name] = sources
	}

	assert.Equal(t, map[string]string{
		"default-src":     "'self'",
		"img-src":         "'self' data:",
		"style-src-attr":  "'unsafe-inline'",
		"object-src":      "'none'",
		"base-uri":        "'self'",
		"form-action":     "'self'",
		"frame-ancestors": "'none'",
	}, directives)
}
