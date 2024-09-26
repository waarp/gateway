package backup

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/filesystems"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestExportClouds(t *testing.T) {
	const testFsType = "test-fs-type"

	logger := testhelpers.GetTestLogger(t)

	filesystems.FileSystems.Store(testFsType,
		func(string, string, map[string]any) (fs.FS, error) {
			return nil, nil //nolint:nilnil //doesn't matter, we don't use it
		})
	t.Cleanup(func() {
		filesystems.FileSystems.Delete(testFsType)
	})

	db := dbtest.TestDatabase(t)

	dbCloud := model.CloudInstance{
		Name:    "remote-fs",
		Type:    testFsType,
		Key:     "access-key",
		Secret:  "access-secret",
		Options: map[string]any{"key1": "val1", "key2": true},
	}
	require.NoError(t, db.Insert(&dbCloud).Run())

	res, err := exportClouds(logger, db)
	require.NoError(t, err)

	require.Len(t, res, 1)

	require.Equal(t, dbCloud.Name, res[0].Name)
	require.Equal(t, dbCloud.Type, res[0].Type)
	require.Equal(t, dbCloud.Key, res[0].Key)
	require.Equal(t, dbCloud.Secret.String(), res[0].Secret)
	require.Equal(t, dbCloud.Options, res[0].Options)
}
