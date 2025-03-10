package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestExportCryptoKeys(t *testing.T) {
	const (
		aesKey  = "0123456789abcdefhijklABCDEFHIJKL"
		hmacKey = "0987654321"
	)

	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

	dbKey1 := model.CryptoKey{
		Name: "aes-key",
		Type: model.CryptoKeyTypeAES,
		Key:  aesKey,
	}
	require.NoError(t, db.Insert(&dbKey1).Run())

	dbKey2 := model.CryptoKey{
		Name: "hmac-key",
		Type: model.CryptoKeyTypeHMAC,
		Key:  hmacKey,
	}
	require.NoError(t, db.Insert(&dbKey2).Run())

	res, err := exportCryptoKeys(logger, db)
	require.NoError(t, err)
	require.Len(t, res, 2)

	assert.Equal(t, dbKey1.Name, res[0].Name)
	assert.Equal(t, dbKey1.Type, res[0].Type)
	assert.Equal(t, dbKey1.Key.String(), res[0].Key)

	assert.Equal(t, dbKey2.Name, res[1].Name)
	assert.Equal(t, dbKey2.Type, res[1].Type)
	assert.Equal(t, dbKey2.Key.String(), res[1].Key)
}
