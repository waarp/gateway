package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCryptoKeysImport(t *testing.T) {
	const (
		aesKey   = "0123456789abcdefhijklABCDEFHIJKL"
		hmacKey1 = "0123456789"
		hmacKey2 = "abcdefghij"
	)

	logger := testhelpers.GetTestLogger(t)

	existing := &model.CryptoKey{
		Name: "existing-aes",
		Type: model.CryptoKeyTypeAES,
		Key:  aesKey,
	}

	newKey := &file.CryptoKey{
		Name: "remote-fs",
		Type: model.CryptoKeyTypeHMAC,
		Key:  hmacKey1,
	}

	t.Run("New key", func(t *testing.T) {
		t.Run("With reset", func(t *testing.T) {
			db := dbtest.TestDatabase(t)
			require.NoError(t, db.Insert(existing).Run())

			impErr := importCryptoKeys(logger, db, []*file.CryptoKey{newKey}, true)
			require.NoError(t, impErr)

			var dbKeys model.CryptoKeys
			require.NoError(t, db.Select(&dbKeys).OrderBy("id", true).Run())
			assert.Len(t, dbKeys, 1)

			assert.Equal(t, newKey.Name, dbKeys[0].Name)
			assert.Equal(t, newKey.Type, dbKeys[0].Type)
			assert.Equal(t, newKey.Key, dbKeys[0].Key.String())
		})

		t.Run("Without reset", func(t *testing.T) {
			db := dbtest.TestDatabase(t)
			require.NoError(t, db.Insert(existing).Run())

			impErr := importCryptoKeys(logger, db, []*file.CryptoKey{newKey}, false)
			require.NoError(t, impErr)

			var dbKeys model.CryptoKeys
			require.NoError(t, db.Select(&dbKeys).OrderBy("id", true).Run())
			assert.Len(t, dbKeys, 2)

			assert.Equal(t, existing, dbKeys[0])

			assert.Equal(t, newKey.Name, dbKeys[1].Name)
			assert.Equal(t, newKey.Type, dbKeys[1].Type)
			assert.Equal(t, newKey.Key, dbKeys[1].Key.String())
		})
	})

	t.Run("Update existing", func(t *testing.T) {
		db := dbtest.TestDatabase(t)
		require.NoError(t, db.Insert(existing).Run())

		oldKey := &model.CryptoKey{
			Name: newKey.Name,
			Type: model.CryptoKeyTypeHMAC,
			Key:  hmacKey2,
		}
		require.NoError(t, db.Insert(oldKey).Run())

		impErr := importCryptoKeys(logger, db, []*file.CryptoKey{newKey}, false)
		require.NoError(t, impErr)

		var dbKeys model.CryptoKeys
		require.NoError(t, db.Select(&dbKeys).OrderBy("id", true).Run())
		assert.Len(t, dbKeys, 2)

		assert.Equal(t, existing, dbKeys[0])

		assert.Equal(t, newKey.Name, dbKeys[1].Name)
		assert.Equal(t, newKey.Type, dbKeys[1].Type)
		assert.Equal(t, newKey.Key, dbKeys[1].Key.String())
	})
}
