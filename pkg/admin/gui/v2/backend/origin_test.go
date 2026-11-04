package backend

import (
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// Every screen that may be the origin of a link (#840) is a GET route of
// the web interface, registered with the very template the table names: a
// forged origin can lead nowhere else than to a screen.
func TestOriginsAreScreens(t *testing.T) {
	t.Parallel()

	router := mux.NewRouter()
	MakeRouter(router, dbtest.TestDatabase(t), testhelpers.GetTestLogger(t))

	variables := regexp.MustCompile(`\{[a-z]+(:[^}]*)?\}`)

	for _, origin := range common.Origins {
		path := variables.ReplaceAllString(origin.Path, "1")

		var match mux.RouteMatch

		req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: path}}
		require.Truef(t, router.Match(req, &match), "%s is a route of the interface", origin.Path)
		require.NoErrorf(t, match.MatchErr, "%s serves GET", origin.Path)

		template, err := match.Route.GetPathTemplate()
		require.NoError(t, err)
		assert.Equalf(t, origin.Path, template, "the origin %s names the route as registered", origin.Path)
	}
}
