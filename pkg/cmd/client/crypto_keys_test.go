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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCryptoKeysGet(t *testing.T) {
	const (
		keyName = "foobar"
		keyType = model.CryptoKeyTypePGPPrivate
		key     = testhelpers.TestPGPPrivateKey

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
				"name": keyName,
				"type": keyType,
				"key":  key,
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
						`‣PGP private key "{{.name}}"`,
						`  •Entity:`,
						`    ⁃Waarp (Waarp PGP test key) <info@waarp.org>`,
						`  •Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
					),
					w.String(),
					"Then it should display the key's details",
				)
			})
		})
	})
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
		key2Key  = testhelpers.TestPGPPrivateKey
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
			"name": key1Name,
			"type": key1Type,
			"key":  key1Key,
		}, {
			"name": key2Name,
			"type": key2Type,
			"key":  key2Key,
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
						`‣PGP public key "{{.name}}"`,
						`  •Entity:`,
						`    ⁃Waarp (Waarp PGP test key) <info@waarp.org>`,
						`  •Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
						`{{- end }}`,
						`{{- with index . 1 }}`,
						`‣PGP private key "{{.name}}"`,
						`  •Entity:`,
						`    ⁃Waarp (Waarp PGP test key) <info@waarp.org>`,
						`  •Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
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
