package rest

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestAddPGPKey(t *testing.T) {
	const (
		keyName    = "keyname"
		privateKey = testhelpers.TestPGPPrivateKey
		publicKey  = testhelpers.TestPGPPublicKey
	)

	testAdd(t, addPGPKey, PGPKeysPath, keyName,
		map[string]any{
			"name":       keyName,
			"privateKey": privateKey,
			"publicKey":  publicKey,
		},
		&model.PGPKey{
			ID:         1,
			Name:       keyName,
			PrivateKey: privateKey,
			PublicKey:  publicKey,
		},
	)
}

func TestGetPGPKey(t *testing.T) {
	const (
		keyName    = "keyname"
		privateKey = testhelpers.TestPGPPrivateKey
		publicKey  = testhelpers.TestPGPPublicKey
	)

	testGet(t, getPGPKey, PGPKeyPath, keyName,
		&model.PGPKey{
			Name:       keyName,
			PrivateKey: privateKey,
			PublicKey:  publicKey,
		},
		map[string]any{
			"name":       keyName,
			"privateKey": privateKey,
			"publicKey":  publicKey,
		},
	)
}

func TestDeletePGPKey(t *testing.T) {
	const keyName = "keyname"

	testDelete(t, deletePGPKey, PGPKeyPath, keyName,
		&model.PGPKey{
			Name:       keyName,
			PrivateKey: testhelpers.TestPGPPrivateKey,
			PublicKey:  testhelpers.TestPGPPublicKey,
		},
	)
}

func TestUpdatePGPKey(t *testing.T) {
	const (
		oldKeyName = "keyname"

		newKeyName = "newkeyname"
		privateKey = ""
		publicKey  = testhelpers.TestPGPPublicKey
	)

	testUpdate(t, updatePGPKey, PGPKeyPath, oldKeyName, newKeyName,
		&model.PGPKey{
			Name:       oldKeyName,
			PrivateKey: testhelpers.TestPGPPrivateKey,
		},
		map[string]any{
			"name":       newKeyName,
			"privateKey": privateKey,
			"publicKey":  publicKey,
		},
		&model.PGPKey{
			ID:         1,
			Name:       newKeyName,
			PrivateKey: privateKey,
			PublicKey:  publicKey,
		},
	)
}
