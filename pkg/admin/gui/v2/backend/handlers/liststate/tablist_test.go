package liststate

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The state of a tab list carries the list state of its screen (?ret=)
// through the toolbar form, the sort links and the list foot.
func TestTabList(t *testing.T) {
	t.Parallel()

	q := Parse(url.Values{"q": {" x "}, "sort": {"type"}, "dir": {"desc"}, "page": {"2"}, "size": {"25"}}, []string{"name", "type"}, DefaultSizes)

	list := NewTabList("/webui/servers/1/auth", q, "q=srv")
	assert.Equal(t, "/webui/servers/1/auth?q=x&ret=q%3Dsrv", list.Base)
	assert.Equal(t, map[string]string{"sort": "type", "dir": "desc", "size": "25", "ret": "q=srv"}, list.Hidden)
	assert.Equal(t, map[string]string{"q": "x", "sort": "type", "dir": "desc", "ret": "q=srv"}, list.PagerHidden)

	list.Foot("info", q.Bounds(60), q, "/webui/servers/1/auth", "q=srv")
	assert.Equal(t, DefaultSizes, list.Sizes)
	assert.Equal(t, "/webui/servers/1/auth?dir=desc&q=x&size=25&sort=type&ret=q%3Dsrv", list.Pager.PrevURL)
	assert.Equal(t, "/webui/servers/1/auth?dir=desc&page=3&q=x&size=25&sort=type&ret=q%3Dsrv", list.Pager.NextURL)

	list.SetNoMatch("none", "/webui/servers/1/auth?ret=q%3Dsrv", "clear")
	assert.True(t, list.NoMatch)
	assert.Equal(t, "clear", list.NoMatchLink)
}
