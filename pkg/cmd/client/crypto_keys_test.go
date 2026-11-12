package wg

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/mattn/go-colorable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// What the API says of the test PGP key pair.
const (
	testPGPFingerprint = "57e09fbb394eec758040cdc4b9f43dcb4c634a87"
	testPGPIdentity    = "Waarp (Waarp PGP test key) <info@waarp.org>"
)

func TestCryptoKeysGet(t *testing.T) {
	const (
		keyName = "foobar"
		keyType = model.CryptoKeyTypePGPPrivate

		path = "/api/keys/" + keyName
	)

	t.Run(`Testing the cryptographic key "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &CryptoKeysGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":        keyName,
				"type":        keyType,
				"fingerprint": testPGPFingerprint,
				"identities":  []string{testPGPIdentity},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, keyName),
					"Then it should not return an error")

				outputData := maps.Clone(result.body)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`-PGP private key "{{.name}}"`,
						`  -Entity:`,
						`    -Waarp (Waarp PGP test key) <info@waarp.org>`,
						`  -Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
						`  -Value: <set>`,
					),
					w.String(),
					"Then it should display the key's details",
				)
			})
		})
	})
}

// The API never returns the value of a secret key: the command says that it is
// set, and describes a PGP key by what the API says of it.
func TestCryptoKeysDisplay(t *testing.T) {
	const pgpDetails = "  -Entity:\n    -" + testPGPIdentity + "\n  -Fingerprint: " + testPGPFingerprint + "\n"

	for _, test := range []struct {
		key      api.GetCryptoKeyRespObject
		expected string
	}{
		{
			key:      api.GetCryptoKeyRespObject{Name: "aes-key", Type: model.CryptoKeyTypeAES},
			expected: `-"AES" key "aes-key"` + "\n  -Value: <set>\n",
		},
		{
			key:      api.GetCryptoKeyRespObject{Name: "hmac-key", Type: model.CryptoKeyTypeHMAC},
			expected: `-"HMAC" key "hmac-key"` + "\n  -Value: <set>\n",
		},
		{
			key: api.GetCryptoKeyRespObject{
				Name: "pgp-private", Type: model.CryptoKeyTypePGPPrivate,
				Fingerprint: testPGPFingerprint, Identities: []string{testPGPIdentity},
			},
			expected: `-PGP private key "pgp-private"` + "\n" + pgpDetails + "  -Value: <set>\n",
		},
		{
			key: api.GetCryptoKeyRespObject{
				Name: "pgp-public", Type: model.CryptoKeyTypePGPPublic, Key: testhelpers.TestPGPPublicKey,
				Fingerprint: testPGPFingerprint, Identities: []string{testPGPIdentity},
			},
			expected: `-PGP public key "pgp-public"` + "\n" + pgpDetails,
		},
		{
			key:      api.GetCryptoKeyRespObject{Name: "other-key", Type: "OTHER"},
			expected: `-Cryptographic key "other-key" of unknown type "OTHER"` + "\n  -Value: <set>\n",
		},
	} {
		t.Run(test.key.Type, func(t *testing.T) {
			w := newTestOutput()

			require.NoError(t, displayCryptoKey(colorable.NewNonColorable(w), &test.key))
			assert.Equal(t, test.expected, w.String())
		})
	}
}

func TestCryptoKeysAdd(t *testing.T) {
	const (
		keyName = "foobar"
		keyType = model.CryptoKeyTypePGPPrivate
		key     = testhelpers.TestPGPPublicKey

		path     = "/api/keys"
		location = path + "/" + keyName
	)

	keyFilePath := filepath.Join(t.TempDir(), "test.key")
	require.NoError(t, os.WriteFile(keyFilePath, []byte(key), 0o600))

	t.Run(`Testing the cryptographic key "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &CryptoKeysAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name": keyName,
				"type": keyType,
				"key":  key,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", keyName,
					"--type", keyType,
					"--key", keyFilePath),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The cryptographic key %q was successfully added.\n", keyName),
					w.String(),
					"Then it should display a message saying the cryptographic key was added",
				)
			})
		})
	})
}

func TestCryptoKeysList(t *testing.T) {
	const (
		path = "/api/keys"

		sort   = "name+"
		limit  = "10"
		offset = "5"

		key1Name = "foo"
		key1Type = model.CryptoKeyTypePGPPublic
		key1Key  = testhelpers.TestPGPPublicKey

		key2Name = "bar"
		key2Type = model.CryptoKeyTypePGPPrivate

		key3Name = "baz"
		key3Type = model.CryptoKeyTypeAES
	)

	t.Run(`Testing the cryptographic keys "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &CryptoKeysList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{
				"sort":   []string{sort},
				"limit":  []string{limit},
				"offset": []string{offset},
			},
		}

		keys := []map[string]any{{
			"name":        key1Name,
			"type":        key1Type,
			"key":         key1Key,
			"fingerprint": testPGPFingerprint,
			"identities":  []string{testPGPIdentity},
		}, {
			"name":        key2Name,
			"type":        key2Type,
			"fingerprint": testPGPFingerprint,
			"identities":  []string{testPGPIdentity},
		}, {
			"name": key3Name,
			"type": key3Type,
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"cryptoKeys": keys},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--sort", sort, "--limit", limit, "--offset", offset),
					"Then it should not return an error")

				outputData := slices.Clone(keys)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`=== Cryptographic keys ===`,
						`{{- with index . 0 }}`,
						`-PGP public key "{{.name}}"`,
						`  -Entity:`,
						`    -Waarp (Waarp PGP test key) <info@waarp.org>`,
						`  -Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
						`{{- end }}`,
						`{{- with index . 1 }}`,
						`-PGP private key "{{.name}}"`,
						`  -Entity:`,
						`    -Waarp (Waarp PGP test key) <info@waarp.org>`,
						`  -Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
						`  -Value: <set>`,
						`{{- end }}`,
						`{{- with index . 2 }}`,
						`-"AES" key "{{.name}}"`,
						`  -Value: <set>`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the cryptographic keys' info",
				)
			})
		})
	})
}

func TestCryptoKeysDelete(t *testing.T) {
	const (
		keyName = "foobar"
		path    = "/api/keys/" + keyName
	)

	t.Run(`Testing the cryptographic key "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &CryptoKeysDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, keyName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The cryptographic key %q was successfully deleted.\n", keyName),
					w.String(),
					"Then it should display a message saying the cryptographic key was deleted")
			})
		})
	})
}

func TestCryptoKeysUpdate(t *testing.T) {
	const (
		oldKeyName = "foo"
		newKeyName = "bar"
		newKeyType = model.CryptoKeyTypePGPPublic
		newKey     = testhelpers.TestPGPPublicKey

		path     = "/api/keys/" + oldKeyName
		location = "/api/keys/" + newKeyName
	)

	keyFilePath := filepath.Join(t.TempDir(), "test.key")
	require.NoError(t, os.WriteFile(keyFilePath, []byte(newKey), 0o600))

	t.Run(`Testing the cryptographic key "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &CryptoKeysUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name": newKeyName,
				"type": newKeyType,
				"key":  newKey,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					oldKeyName,
					"--name", newKeyName,
					"--type", newKeyType,
					"--key", keyFilePath),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The cryptographic key %q was successfully updated.\n", newKeyName),
					w.String(),
					"Then it should display a message saying the cryptographic key was updated",
				)
			})
		})
	})
}
