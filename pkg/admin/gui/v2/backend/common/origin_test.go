package common

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"code.waarp.fr/lib/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func translateKey(key string, _ ...any) string { return "[" + key + "]" }

// The origin a page carries is honoured only when it is an address of the
// web interface that names a screen that may be an origin; its label comes
// from that screen, its address is kept with its query.
func TestOriginOf(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		from, label, target string
	}{
		{"/webui/dashboard", "origin.dashboard", "/webui/dashboard"},
		{"/webui/transfers?role=client&state=error", "origin.transfers", "/webui/transfers?role=client&state=error"},
		{"/webui/transfers/4100008", "origin.transfer", "/webui/transfers/4100008"},
		{"/webui/servers/3/accounts/7", "origin.account", "/webui/servers/3/accounts/7"},
		{"/webui/rules/2/tasks", "origin.rule", "/webui/rules/2/tasks"},
		{"/webui/partners/5/accounts/8/auth/13", "origin.credential", "/webui/partners/5/accounts/8/auth/13"},
		{"/webui/services?group=core#core", "origin.services", "/webui/services?group=core"},
		// A chain: the origin of the origin stays in the query.
		{"/webui/transfers?origin=%2Fwebui%2Fdashboard&state=error", "origin.transfers", "/webui/transfers?origin=%2Fwebui%2Fdashboard&state=error"},
	} {
		label, target, ok := OriginOf(tc.from)
		require.Truef(t, ok, "%q is an origin", tc.from)
		assert.Equal(t, tc.label, label, tc.from)
		assert.Equal(t, tc.target, target, tc.from)
	}

	for _, from := range []string{
		"",
		"/webui/transfers/new",     // a form is no origin
		"/webui/transfers/42/edit", // nor an action page
		"/webui/transfers/abc",     // not a screen
		"/webui/logout",            // not a screen
		"/webui",                   // the prefix alone
		"/other/dashboard",         // outside the web interface
		"dashboard",                // not an address
		"https://example.com/webui/dashboard",
		"//example.com/webui/dashboard",
		"/webui/dashboard?" + strings.Repeat("x", MaxOriginLength),
	} {
		_, _, ok := OriginOf(from)
		assert.Falsef(t, ok, "%q is no origin", from)
	}
}

// The return link of a page is the origin carried, else the default of the
// handler, else nothing.
func TestReturnOf(t *testing.T) {
	t.Parallel()

	fallback := &gui.Link{Label: "Retour à la liste", URL: "/webui/transfers?page=2"}

	r := httptest.NewRequest(http.MethodGet, "/webui/transfers/42?origin=%2Fwebui%2Fdashboard", nil)
	ret := ReturnOf(r, translateKey, fallback)
	require.NotNil(t, ret)
	assert.Equal(t, "[origin.dashboard]", ret.Label)
	assert.Equal(t, "/webui/dashboard", ret.URL)

	r = httptest.NewRequest(http.MethodGet, "/webui/transfers/42?origin=https%3A%2F%2Fexample.com%2Fwebui%2Fdashboard", nil)
	assert.Same(t, fallback, ReturnOf(r, translateKey, fallback), "a forged origin leaves the default")

	r = httptest.NewRequest(http.MethodGet, "/webui/transfers/42", nil)
	assert.Same(t, fallback, ReturnOf(r, translateKey, fallback))
	assert.Nil(t, ReturnOf(r, translateKey, nil))
}
