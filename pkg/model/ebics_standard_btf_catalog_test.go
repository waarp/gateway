package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestDatabaseSeedsEbicsStandardBTFCatalogs(t *testing.T) {
	db := dbtest.TestDatabase(t)

	var catalogs EbicsStandardBTFCatalogs
	require.NoError(t, db.Select(&catalogs).Owner().Run())
	require.Len(t, catalogs, 5)

	scopes := make([]string, 0, len(catalogs))
	for _, catalog := range catalogs {
		scopes = append(scopes, catalog.Scope)
		assert.Equal(t, "gateway-standard-btf", catalog.Name)
		assert.Equal(t, "2024-10-23-baseline-v1", catalog.CatalogVersion)
		assert.Equal(t, "ACTIVE", catalog.Status)
		assert.NotEmpty(t, catalog.SeedChecksum)
	}

	assert.ElementsMatch(t, []string{"GLB", "FR", "DE", "AT", "CH"}, scopes)

	var entries EbicsStandardBTFEntries
	require.NoError(t, db.Select(&entries).Owner().Run())
	require.GreaterOrEqual(t, len(entries), 12)
}
