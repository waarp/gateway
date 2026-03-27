package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestImportCuratedEbicsStandardBTFCatalogs(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	fixturePath := filepath.Join("testdata", "ebics-standard-btf-curated.yaml")
	fixture, err := os.Open(filepath.Clean(fixturePath))
	require.NoError(t, err)
	defer fixture.Close() //nolint:errcheck // test cleanup

	require.NoError(t, ImportData(db, fixture, []string{"ebics"}, false, true))

	var catalogs model.EbicsStandardBTFCatalogs
	require.NoError(t, db.Select(&catalogs).Owner().Where("catalog_version=?", "curated-country-pack-v1").Run())
	require.Len(t, catalogs, 5)

	var entries model.EbicsStandardBTFEntries
	require.NoError(t, db.Select(&entries).Owner().Run())
	require.Greater(t, len(entries), 50)

	var glbEntries model.EbicsStandardBTFEntries
	require.NoError(t, db.Select(&glbEntries).Owner().Where("scope=?", "GLB").Run())
	require.NotEmpty(t, glbEntries)

	var frCatalog model.EbicsStandardBTFCatalog
	require.NoError(t, db.Get(&frCatalog, "scope=? AND catalog_version=?", "FR", "curated-country-pack-v1").Owner().Run())

	var frPackEntries model.EbicsStandardBTFEntries
	require.NoError(t, db.Select(&frPackEntries).Owner().Where("catalog_id=?", frCatalog.ID).Run())

	mixedScopes := map[string]bool{}
	for _, entry := range frPackEntries {
		mixedScopes[entry.Scope] = true
	}
	require.True(t, mixedScopes["FR"])
	require.True(t, mixedScopes["GLB"])
}
