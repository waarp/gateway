package rest

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestRoutesRegisteredOnce(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	router := mux.NewRouter()
	MakeRESTHandler(testhelpers.GetTestLogger(t), db, router)

	seen := map[string]int{}

	require.NoError(t, router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		if name := route.GetName(); name != "" {
			seen[name]++
		}

		return nil
	}))

	for name, count := range seen {
		assert.Equal(t, 1, count, "route %q is registered %d times", name, count)
	}
}
