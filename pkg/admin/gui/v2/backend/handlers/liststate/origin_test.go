package liststate

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
)

// The origin of a list (?origin=) is part of the state it carries to its
// sheets and action pages, of its toolbar, but never of the state kept for
// the session.
func TestQueryOrigin(t *testing.T) {
	t.Parallel()

	spec := &Spec{Path: "/transfers", Cookie: "transfers", Sorts: []string{"start"}, Sizes: []int{10}}

	query := spec.Parse(url.Values{OriginParam: {"/webui/dashboard"}, "page": {"2"}})
	assert.Equal(t, "/webui/dashboard", query.Origin)
	assert.Equal(t, "/webui/dashboard", query.FilterValues().Get(OriginParam), "the base of the sort links")
	assert.Equal(t, "/webui/dashboard", query.ToolbarHidden()[OriginParam], "the search form keeps it")
	assert.Equal(t, "origin=%2Fwebui%2Fdashboard&page=2", query.Encode(query.Page), "the carried state")
	assert.Equal(t, "/webui/transfers?origin=%2Fwebui%2Fdashboard&page=2", spec.ListURL(query.Encode(query.Page)))

	assert.Empty(t, spec.Parse(url.Values{OriginParam: {"dashboard"}}).Origin, "not an address")
	assert.Empty(t, spec.Parse(url.Values{OriginParam: {"/" + strings.Repeat("x", common.MaxOriginLength)}}).Origin, "too long")

	w := httptest.NewRecorder()
	spec.KeepState(w, query)
	cookie := w.Result().Cookies()[0]
	assert.Equal(t, url.QueryEscape("page=2"), cookie.Value, "the session keeps the state without the origin")
}

// A request that carries only an origin is restored to the kept state, the
// origin travelling with it.
func TestRestoreKeepsOrigin(t *testing.T) {
	t.Parallel()

	spec := &Spec{Path: "/transfers", Cookie: "transfers", Sorts: []string{"start", "rule"}, Sizes: []int{10}}

	r := httptest.NewRequest(http.MethodGet, "/webui/transfers?origin=%2Fwebui%2Fdashboard", nil)
	r.AddCookie(&http.Cookie{Name: "transfers", Value: url.QueryEscape("sort=rule&dir=asc")})
	w := httptest.NewRecorder()
	require.True(t, spec.Restore(w, r))
	assert.Equal(t, "/webui/transfers?dir=asc&origin=%2Fwebui%2Fdashboard&sort=rule", w.Result().Header.Get("Location"))
}

// The origin a page gives its links is its address without what belongs to
// the moment, bounded by dropping the older origins first.
func TestOrigin(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/webui/transfers?state=error&fragment=list&done=created&name=x&language=fr", nil)
	assert.Equal(t, "/webui/transfers?state=error", Origin(r))

	r = httptest.NewRequest(http.MethodGet, "/webui/dashboard", nil)
	assert.Equal(t, "/webui/dashboard", Origin(r))
	assert.Equal(t, "/webui/services?group=core&origin=%2Fwebui%2Fdashboard", WithOrigin("/webui/services?group=core", Origin(r)))
	assert.Equal(t, "/webui/services", WithOrigin("/webui/services", ""))

	long := "/webui/transfers?origin=" + url.QueryEscape("/webui/dashboard?x="+strings.Repeat("y", common.MaxOriginLength))
	r = httptest.NewRequest(http.MethodGet, long+"&state=error", nil)
	assert.Equal(t, "/webui/transfers?state=error", Origin(r), "the older origin is dropped first")

	r = httptest.NewRequest(http.MethodGet, "/webui/transfers?q="+strings.Repeat("z", common.MaxOriginLength), nil)
	assert.Empty(t, Origin(r), "an address too long on its own is no origin")
}
