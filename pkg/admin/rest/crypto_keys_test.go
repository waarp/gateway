package rest

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestAddCryptoKey(t *testing.T) {
	t.Parallel()

	const (
		keyName = "keyname"
		keyType = model.CryptoKeyTypePGPPrivate
		key     = testhelpers.TestPGPPrivateKey
	)

	testAdd(t, addCryptoKey, "/keys", keyName,
		nil,
		map[string]any{
			"name": keyName,
			"type": keyType,
			"key":  key,
		},
		&model.CryptoKey{
			Identifier: model.ID(1),
			Name:       keyName,
			Type:       keyType,
			Key:        key,
		},
	)
}

func TestGetCryptoKey(t *testing.T) {
	t.Parallel()

	const (
		keyName = "keyname"
		keyType = model.CryptoKeyTypePGPPublic
		key     = testhelpers.TestPGPPublicKey
	)

	testGet(t, getCryptoKey, "/keys/{crypto_key}", keyName,
		nil,
		&model.CryptoKey{
			Name: keyName,
			Type: keyType,
			Key:  key,
		},
		map[string]any{
			"name": keyName,
			"type": keyType,
			"key":  key,
		},
	)
}

func TestDeleteCryptoKey(t *testing.T) {
	t.Parallel()

	const keyName = "keyname"

	testDelete(t, deleteCryptoKey, "/keys/{crypto_key}", keyName,
		nil,
		&model.CryptoKey{
			Name: keyName,
			Type: model.CryptoKeyTypePGPPrivate,
			Key:  testhelpers.TestPGPPrivateKey,
		},
	)
}

func TestUpdateCryptoKey(t *testing.T) {
	t.Parallel()

	const (
		oldKeyName = "keyname"

		newKeyName = "newkeyname"
		newKeyType = model.CryptoKeyTypePGPPublic
		newKey     = testhelpers.TestPGPPublicKey
	)

	testUpdate(t, updateCryptoKey, "/keys/{crypto_key}", oldKeyName, newKeyName,
		nil,
		&model.CryptoKey{
			Name: oldKeyName,
			Type: model.CryptoKeyTypePGPPrivate,
			Key:  testhelpers.TestPGPPrivateKey,
		},
		map[string]any{
			"name": newKeyName,
			"type": newKeyType,
			"key":  newKey,
		},
		&model.CryptoKey{
			Identifier: model.ID(1),
			Name:       newKeyName,
			Type:       newKeyType,
			Key:        newKey,
		},
	)
}
