package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// The facts the test PGP key pair states, and two raw keys in Base64.
const (
	testPGPFingerprint = "57e09fbb394eec758040cdc4b9f43dcb4c634a87"
	testPGPIdentity    = "Waarp (Waarp PGP test key) <info@waarp.org>"
	testAESKey         = "MDEyMzQ1Njc4OWFiY2RlZg=="
	testHMACKey        = "c2VjcmV0LWhtYWMta2V5"
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

// The value of a secret key is never returned; a PGP key, public or private,
// is described by its fingerprint and its identities.
func TestGetCryptoKey(t *testing.T) {
	t.Parallel()

	const keyName = "keyname"

	for _, test := range []struct {
		keyType, key string
		expected     map[string]any
	}{
		{
			keyType: model.CryptoKeyTypePGPPublic,
			key:     testhelpers.TestPGPPublicKey,
			expected: map[string]any{
				"name":        keyName,
				"type":        model.CryptoKeyTypePGPPublic,
				"key":         testhelpers.TestPGPPublicKey,
				"fingerprint": testPGPFingerprint,
				"identities":  []string{testPGPIdentity},
			},
		},
		{
			keyType: model.CryptoKeyTypePGPPrivate,
			key:     testhelpers.TestPGPPrivateKey,
			expected: map[string]any{
				"name":        keyName,
				"type":        model.CryptoKeyTypePGPPrivate,
				"fingerprint": testPGPFingerprint,
				"identities":  []string{testPGPIdentity},
			},
		},
		{
			keyType:  model.CryptoKeyTypeAES,
			key:      testAESKey,
			expected: map[string]any{"name": keyName, "type": model.CryptoKeyTypeAES},
		},
		{
			keyType:  model.CryptoKeyTypeHMAC,
			key:      testHMACKey,
			expected: map[string]any{"name": keyName, "type": model.CryptoKeyTypeHMAC},
		},
	} {
		t.Run(test.keyType, func(t *testing.T) {
			t.Parallel()

			testGet(t, getCryptoKey, "/keys/{crypto_key}", keyName,
				nil,
				&model.CryptoKey{Name: keyName, Type: test.keyType, Key: test.key},
				test.expected,
			)
		})
	}
}

func TestListCryptoKeys(t *testing.T) {
	t.Parallel()

	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	handle := listCryptoKeys(logger, db)

	for _, dbKey := range []*model.CryptoKey{
		{Name: "aes", Type: model.CryptoKeyTypeAES, Key: testAESKey},
		{Name: "hmac", Type: model.CryptoKeyTypeHMAC, Key: testHMACKey},
		{Name: "pgp-private", Type: model.CryptoKeyTypePGPPrivate, Key: testhelpers.TestPGPPrivateKey},
		{Name: "pgp-public", Type: model.CryptoKeyTypePGPPublic, Key: testhelpers.TestPGPPublicKey},
	} {
		require.NoError(t, db.Insert(dbKey).Run())
	}

	req := makeRequest(http.MethodGet, nil, "/keys", "")
	w := httptest.NewRecorder()
	handle.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, `Then the response code should be "200 OK"`)
	assert.JSONEq(t,
		marshal(t, map[string]any{"cryptoKeys": []map[string]any{
			{"name": "aes", "type": model.CryptoKeyTypeAES},
			{"name": "hmac", "type": model.CryptoKeyTypeHMAC},
			{
				"name":        "pgp-private",
				"type":        model.CryptoKeyTypePGPPrivate,
				"fingerprint": testPGPFingerprint,
				"identities":  []string{testPGPIdentity},
			},
			{
				"name":        "pgp-public",
				"type":        model.CryptoKeyTypePGPPublic,
				"key":         testhelpers.TestPGPPublicKey,
				"fingerprint": testPGPFingerprint,
				"identities":  []string{testPGPIdentity},
			},
		}}),
		w.Body.String(),
		"Then the keys should have been returned, the secret ones without their value")

	for _, secret := range []string{testAESKey, testHMACKey, "PGP PRIVATE KEY BLOCK"} {
		assert.NotContains(t, w.Body.String(), secret, "Then no secret value should have been returned")
	}
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

// A key read through the API comes without its value: sending it back, renamed,
// must keep the value.
func TestUpdateCryptoKeyWithoutItsValue(t *testing.T) {
	t.Parallel()

	const newKeyName = "newkeyname"

	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)
	handle := updateCryptoKey(logger, db)

	oldDBKey := model.CryptoKey{Name: "keyname", Type: model.CryptoKeyTypeAES, Key: testAESKey}
	require.NoError(t, db.Insert(&oldDBKey).Run())

	reqBody := bytes.Buffer{}
	require.NoError(t, json.NewEncoder(&reqBody).Encode(map[string]any{
		"name": newKeyName,
		"type": model.CryptoKeyTypeAES,
	}))

	expectedDBKey := oldDBKey
	expectedDBKey.Name = newKeyName

	req := makeRequest(http.MethodPatch, &reqBody, "/keys/{crypto_key}", oldDBKey.Name)
	w := httptest.NewRecorder()
	handle.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, `Then the response code should be "201 Created"`)

	var dbKey model.CryptoKey

	require.NoError(t, db.Get(&dbKey, "id=?", oldDBKey.ID).Run())
	assert.Equal(t, expectedDBKey, dbKey, "Then the key should have kept its value")
}
