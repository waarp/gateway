package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCloudImport(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	fsType := fstest.MakeDummyBackend(t)

	existing := &model.CloudInstance{
		Name:    "existing-fs",
		Type:    fsType,
		Key:     "other-access-key",
		Secret:  "other-access-secret",
		Options: map[string]string{"key1": "val1"},
	}

	newCloud := file.Cloud{
		Name:    "remote-fs",
		Type:    fsType,
		Key:     "access-key",
		Secret:  "access-secret",
		Options: map[string]string{"key1": "val1", "key2": "val2"},
	}

	t.Run("Imported cloud does not exist", func(t *testing.T) {
		t.Run("With reset", func(t *testing.T) {
			db := dbtest.TestDatabase(t)
			require.NoError(t, db.Insert(existing).Run())

			impErr := importCloud(logger, db, []file.Cloud{newCloud}, true)
			require.NoError(t, impErr)

			var dbClouds model.CloudInstances
			require.NoError(t, db.Select(&dbClouds).OrderBy("id", true).Run())
			assert.Len(t, dbClouds, 1)

			assert.Equal(t, newCloud.Name, dbClouds[0].Name)
			assert.Equal(t, newCloud.Type, dbClouds[0].Type)
			assert.Equal(t, newCloud.Key, dbClouds[0].Key)
			assert.Equal(t, newCloud.Secret, dbClouds[0].Secret.String())
			assert.Equal(t, newCloud.Options, dbClouds[0].Options)
		})

		t.Run("Without reset", func(t *testing.T) {
			db := dbtest.TestDatabase(t)
			require.NoError(t, db.Insert(existing).Run())

			impErr := importCloud(logger, db, []file.Cloud{newCloud}, false)
			require.NoError(t, impErr)

			var dbClouds model.CloudInstances
			require.NoError(t, db.Select(&dbClouds).OrderBy("id", true).Run())
			assert.Len(t, dbClouds, 2)

			assert.Equal(t, existing, dbClouds[0])

			assert.Equal(t, newCloud.Name, dbClouds[1].Name)
			assert.Equal(t, newCloud.Type, dbClouds[1].Type)
			assert.Equal(t, newCloud.Key, dbClouds[1].Key)
			assert.Equal(t, newCloud.Secret, dbClouds[1].Secret.String())
			assert.Equal(t, newCloud.Options, dbClouds[1].Options)
		})
	})

	t.Run("Imported cloud already exists", func(t *testing.T) {
		db := dbtest.TestDatabase(t)
		require.NoError(t, db.Insert(existing).Run())

		oldCloud := &model.CloudInstance{
			Name:    newCloud.Name,
			Type:    fsType,
			Key:     "old-access-key",
			Secret:  "old-access-secret",
			Options: map[string]string{"key1": "val1"},
		}
		require.NoError(t, db.Insert(oldCloud).Run())

		impErr := importCloud(logger, db, []file.Cloud{newCloud}, false)
		require.NoError(t, impErr)

		var dbClouds model.CloudInstances
		require.NoError(t, db.Select(&dbClouds).OrderBy("id", true).Run())
		assert.Len(t, dbClouds, 2)

		assert.Equal(t, existing, dbClouds[0])

		assert.Equal(t, newCloud.Name, dbClouds[1].Name)
		assert.Equal(t, newCloud.Type, dbClouds[1].Type)
		assert.Equal(t, newCloud.Key, dbClouds[1].Key)
		assert.Equal(t, newCloud.Secret, dbClouds[1].Secret.String())
		assert.Equal(t, newCloud.Options, dbClouds[1].Options)
	})
}
