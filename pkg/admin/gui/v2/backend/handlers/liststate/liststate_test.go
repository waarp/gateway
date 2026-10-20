package liststate

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubFilters keeps one parameter "kind" among two values.
type stubFilters struct{ Kind string }

func (f *stubFilters) Values() url.Values {
	if f.Kind == "" {
		return url.Values{}
	}

	return url.Values{"kind": {f.Kind}}
}

func (f *stubFilters) Count() int {
	if f.Kind == "" {
		return 0
	}

	return 1
}

func testSpec() *Spec {
	return &Spec{
		Path: "/things", Cookie: "webui_things_list", Sorts: []string{"name", "state"}, Sizes: []int{10, 25},
		ParseFilters: func(v url.Values) Filters {
			if kind := v.Get("kind"); kind == "a" || kind == "b" {
				return &stubFilters{Kind: kind}
			}

			return &stubFilters{}
		},
		FilterParams: []string{"kind"},
	}
}

func TestQuery(t *testing.T) {
	t.Parallel()

	spec := testSpec()

	q := spec.Parse(url.Values{
		"q": {" srv "}, "kind": {"b"}, "sort": {"state"}, "dir": {"desc"}, "page": {"3"}, "size": {"25"}, "bogus": {"1"},
	})
	assert.Equal(t, "srv", q.Q)
	assert.Equal(t, "state", q.Sort)
	assert.Equal(t, "desc", q.Dir)
	assert.Equal(t, 3, q.Page)
	assert.Equal(t, 25, q.Size)
	assert.Equal(t, 1, q.FiltersCount())
	assert.Equal(t, "dir=desc&kind=b&page=3&q=srv&size=25&sort=state", q.Encode(q.Page))
	assert.Equal(t, map[string]string{"sort": "state", "dir": "desc", "size": "25"}, q.ToolbarHidden())
	assert.Equal(t, map[string]string{"q": "srv", "kind": "b", "sort": "state", "dir": "desc"}, q.PagerHidden())
	assert.Equal(t, q.Encode(q.Page), spec.RetOf(q.Encode(q.Page)).Encode(3), "the carried state reads back")

	dflt := spec.Parse(url.Values{"sort": {"name"}, "dir": {"asc"}, "size": {"10"}, "page": {"1"}, "kind": {"zzz"}})
	assert.Empty(t, dflt.Encode(dflt.Page), "the default state is carried as nothing")
	assert.Equal(t, 0, dflt.FiltersCount())

	assert.Equal(t, "/webui/things", spec.ListURL(""))
	assert.Equal(t, "/webui/things?q=srv", spec.ListURL("q=srv&bogus=1&page=1"), "unknown parameters are dropped")
	assert.Equal(t, "/webui/things?reset=1", spec.ResetURL())
	assert.Equal(t, "/webui/things?reset=1", spec.ExplicitListURL(url.Values{}))
	assert.Equal(t, "/webui/things?kind=a", spec.ExplicitListURL(url.Values{"kind": {"a"}}))
	assert.Equal(t, "/webui/things/new?ret=q%3Dsrv", WithRet("/webui/things/new", "q=srv"))
	assert.Equal(t, "/webui/things/new", WithRet("/webui/things/new", ""))
	assert.Equal(t, "/webui/things/1?ret=page%3D2&x=1", WithQuery("/webui/things/1?ret=page%3D2", url.Values{"x": {"1"}}))

	assert.True(t, spec.HasParams(url.Values{"kind": {"a"}}))
	assert.True(t, spec.HasParams(url.Values{"reset": {"1"}}))
	assert.False(t, spec.HasParams(url.Values{"done": {"created"}}))

	b := q.Bounds(60)
	assert.Equal(t, Bounds{Page: 3, Pages: 3, From: 50, To: 60}, b)
	assert.Equal(t, Bounds{Page: 1, Pages: 1, From: 0, To: 4}, Parse(url.Values{"page": {"9"}}, spec.Sorts, spec.Sizes).Bounds(4),
		"the page is bounded by the count")

	pager := spec.Pager(q, "51 à 60 sur 60", b)
	assert.Equal(t, "/webui/things?dir=desc&kind=b&page=2&q=srv&size=25&sort=state", pager.PrevURL)
	assert.Empty(t, pager.NextURL)
}

func TestKeptState(t *testing.T) {
	t.Parallel()

	spec := testSpec()

	// A request that says something keeps its state and is not redirected.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/webui/things?kind=a&sort=state&dir=asc", nil)
	assert.False(t, spec.Restore(rec, req))
	require.Len(t, rec.Result().Cookies(), 1)
	cookie := rec.Result().Cookies()[0]
	assert.Equal(t, "webui_things_list", cookie.Name)
	assert.Equal(t, "/webui/things", cookie.Path)
	assert.True(t, cookie.HttpOnly)

	// A request that says nothing is sent to the kept state, with the
	// result of the previous action.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/webui/things?done=created&name=x", nil)
	req.AddCookie(cookie)
	assert.True(t, spec.Restore(rec, req))
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/webui/things?dir=asc&done=created&kind=a&name=x&sort=state", unescape(t, rec.Header().Get("Location")))

	// The explicit reset forgets the kept state.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/webui/things?reset=1", nil)
	req.AddCookie(cookie)
	assert.False(t, spec.Restore(rec, req))
	require.Len(t, rec.Result().Cookies(), 1)
	assert.Equal(t, -1, rec.Result().Cookies()[0].MaxAge)

	// Without a cookie, nothing to restore.
	rec = httptest.NewRecorder()
	assert.False(t, spec.Restore(rec, httptest.NewRequest(http.MethodGet, "/webui/things", nil)))
}

func unescape(t *testing.T, s string) string {
	t.Helper()

	out, err := url.QueryUnescape(s)
	require.NoError(t, err)

	return out
}

func TestCounter(t *testing.T) {
	t.Parallel()

	translate := func(key string, args ...any) string {
		if len(args) == 0 {
			return key
		}

		return key + "/" + args[0].(string) //nolint:forcetypeassert // test stub
	}

	c := Counter{"one", "many", "shown_one", "shown_many"}
	assert.Equal(t, "one", c.Text(translate, 1, 1))
	assert.Equal(t, "many/total", c.Text(translate, 3, 3))
	assert.Equal(t, "shown_one/total", c.Text(translate, 3, 1))
	assert.Equal(t, "shown_many/total", c.Text(translate, 3, 2))
}
