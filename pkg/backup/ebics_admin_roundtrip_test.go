package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestImportExportEbicsAdminRoundTrip(t *testing.T) {
	t.Parallel()

	for _, fixtureName := range []string{"ebics-admin.json", "ebics-admin.yaml"} {
		t.Run(fixtureName, func(t *testing.T) {
			t.Parallel()

			db := dbtest.TestDatabase(t)
			fixturePath := filepath.Join("testdata", fixtureName)
			fixture, err := os.Open(filepath.Clean(fixturePath))
			require.NoError(t, err)
			defer fixture.Close() //nolint:errcheck // test cleanup

			require.NoError(t, ImportData(db, fixture, []string{"all"}, false, true))

			assertImportedEbicsAdminData(t, db)

			exportPath := filepath.Join(t.TempDir(), "exported-"+fixtureName)
			exportFile, err := os.Create(exportPath)
			require.NoError(t, err)

			require.NoError(t, ExportData(db, exportFile, []string{"ebics"}))
			require.NoError(t, exportFile.Close())

			exported := loadBackupData(t, exportPath)
			require.Len(t, exported.EbicsHosts, 1)
			require.Len(t, exported.EbicsSubscribers, 2)
			require.Len(t, exported.EbicsBankKeys, 2)
			require.Len(t, exported.EbicsStandardBTFCatalogs, 5)
			require.Len(t, exported.EbicsPayloadProfiles, 2)
			require.Len(t, exported.EbicsRTNProviders, 1)

			require.Equal(t, "BANKHOST", exported.EbicsHosts[0].HostID)
			require.Contains(t, exportedSubscriberNames(exported), "corp-client")
			require.Contains(t, exportedBankKeyTypes(exported), "AUTH")
			require.Contains(t, exportedStandardCatalogScopes(exported), "GLB")
			require.Contains(t, exportedStandardCatalogVersions(exported), "curated-country-pack-v1")
			require.Contains(t, exportedPayloadProfileNames(exported), "sct-upload")
			require.Equal(t, "main-rtn", exported.EbicsRTNProviders[0].Name)
		})
	}
}

func assertImportedEbicsAdminData(t *testing.T, db *database.DB) {
	t.Helper()

	var hosts model.EbicsHosts
	require.NoError(t, db.Select(&hosts).Owner().Run())
	require.Len(t, hosts, 1)
	require.Equal(t, "BANKHOST", hosts[0].HostID)

	var subscribers model.EbicsSubscribers
	require.NoError(t, db.Select(&subscribers).Owner().OrderBy("name", true).Run())
	require.Len(t, subscribers, 2)
	require.True(t, subscribers[0].RemoteAccountID.Valid)
	require.True(t, subscribers[1].LocalAccountID.Valid)

	var keys model.EbicsBankKeys
	require.NoError(t, db.Select(&keys).Owner().OrderBy("key_type", true).Run())
	require.Len(t, keys, 2)
	require.Equal(t, "validated", keys[0].State)

	var profiles model.EbicsPayloadProfiles
	require.NoError(t, db.Select(&profiles).Owner().OrderBy("name", true).Run())
	require.Len(t, profiles, 2)
	require.True(t, profiles[0].DefaultRuleID.Valid)
	require.True(t, profiles[0].StrictContractCheck)

	var catalogs model.EbicsStandardBTFCatalogs
	require.NoError(t, db.Select(&catalogs).Owner().OrderBy("scope", true).Run())
	require.Len(t, catalogs, 5)
	require.Equal(t, "gateway-standard-btf", catalogs[0].Name)

	var entries model.EbicsStandardBTFEntries
	require.NoError(t, db.Select(&entries).Owner().Run())
	require.Greater(t, len(entries), 100)

	var providers model.EbicsRTNProviders
	require.NoError(t, db.Select(&providers).Owner().Run())
	require.Len(t, providers, 1)
	require.True(t, providers[0].Enabled)
}

func loadBackupData(t *testing.T, filePath string) *file.Data {
	t.Helper()

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)

	data := &file.Data{}
	switch filepath.Ext(filePath) {
	case ".yaml", ".yml":
		require.NoError(t, yaml.Unmarshal(content, data))
	default:
		require.NoError(t, json.Unmarshal(content, data))
	}

	return data
}

func exportedSubscriberNames(data *file.Data) []string {
	names := make([]string, 0, len(data.EbicsSubscribers))
	for _, subscriber := range data.EbicsSubscribers {
		names = append(names, subscriber.Name)
	}

	return names
}

func exportedBankKeyTypes(data *file.Data) []string {
	types := make([]string, 0, len(data.EbicsBankKeys))
	for _, key := range data.EbicsBankKeys {
		types = append(types, key.KeyType)
	}

	return types
}

func exportedPayloadProfileNames(data *file.Data) []string {
	names := make([]string, 0, len(data.EbicsPayloadProfiles))
	for _, profile := range data.EbicsPayloadProfiles {
		names = append(names, profile.Name)
	}

	return names
}

func exportedStandardCatalogScopes(data *file.Data) []string {
	scopes := make([]string, 0, len(data.EbicsStandardBTFCatalogs))
	for _, catalog := range data.EbicsStandardBTFCatalogs {
		scopes = append(scopes, catalog.Scope)
	}

	return scopes
}

func exportedStandardCatalogVersions(data *file.Data) []string {
	versions := make([]string, 0, len(data.EbicsStandardBTFCatalogs))
	for _, catalog := range data.EbicsStandardBTFCatalogs {
		versions = append(versions, catalog.CatalogVersion)
	}

	return versions
}
