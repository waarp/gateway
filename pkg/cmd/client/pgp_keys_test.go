package wg

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestPGPKeysGet(t *testing.T) {
	const (
		keyName    = "foobar"
		publicKey  = testhelpers.TestPGPPublicKey
		privateKey = testhelpers.TestPGPPrivateKey

		path = "/api/pgp/keys/" + keyName
	)

	t.Run(`Testing the PGP key "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PGPKeysGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":       keyName,
				"publicKey":  publicKey,
				"privateKey": privateKey,
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
						`‣PGP key "{{.name}}"`,
						`  •Private key:`,
						`    ⁃Entity:`,
						`      $Waarp (Waarp PGP test key) <info@waarp.org>`,
						`    ⁃Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
						`  •Public key:`,
						`    ⁃Entity:`,
						`      $Waarp (Waarp PGP test key) <info@waarp.org>`,
						`    ⁃Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
					),
					w.String(),
					"Then it should display the key's details",
				)
			})
		})
	})
}

func TestPGPKeysAdd(t *testing.T) {
	const (
		keyName    = "foobar"
		publicKey  = testhelpers.TestPGPPublicKey
		privateKey = testhelpers.TestPGPPrivateKey

		path     = "/api/pgp/keys"
		location = path + "/" + keyName
	)

	t.Run(`Testing the PGP key "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PGPKeysAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":       keyName,
				"privateKey": privateKey,
				"publicKey":  publicKey,
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
					"--private-key", privateKey,
					"--public-key", publicKey),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The PGP key %q was successfully added.\n", keyName),
					w.String(),
					"Then it should display a message saying the PGP key was added",
				)
			})
		})
	})
}

func TestPGPKeysList(t *testing.T) {
	const (
		path = "/api/pgp/keys"

		sort   = "name+"
		limit  = "10"
		offset = "5"

		key1Name  = "foo"
		key1pbKey = testhelpers.TestPGPPublicKey
		key2Name  = "bar"
		key2pKey  = testhelpers.TestPGPPrivateKey
	)

	t.Run(`Testing the PGP keys "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PGPKeysList{}

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
			"name":      key1Name,
			"publicKey": key1pbKey,
		}, {
			"name":       key2Name,
			"privateKey": key2pKey,
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"pgpKeys": keys},
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
						`=== PGP keys ===`,
						`{{- with index . 0 }}`,
						`‣PGP key "{{.name}}"`,
						`  •Public key:`,
						`    ⁃Entity:`,
						`      $Waarp (Waarp PGP test key) <info@waarp.org>`,
						`    ⁃Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
						`{{- end }}`,
						`{{- with index . 1 }}`,
						`‣PGP key "{{.name}}"`,
						`  •Private key:`,
						`    ⁃Entity:`,
						`      $Waarp (Waarp PGP test key) <info@waarp.org>`,
						`    ⁃Fingerprint: 57e09fbb394eec758040cdc4b9f43dcb4c634a87`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the PGP keys' info",
				)
			})
		})
	})
}

func TestPGPKeysDelete(t *testing.T) {
	const (
		keyName = "foobar"
		path    = "/api/pgp/keys/" + keyName
	)

	t.Run(`Testing the PGP key "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PGPKeysDelete{}

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
					fmt.Sprintf("The PGP key %q was successfully deleted.\n", keyName),
					w.String(),
					"Then it should display a message saying the PGP key was deleted")
			})
		})
	})
}

func TestPGPKeysUpdate(t *testing.T) {
	const (
		oldKeyName = "foo"
		newKeyName = "bar"
		publicKey  = testhelpers.TestPGPPublicKey
		privateKey = testhelpers.TestPGPPrivateKey

		path     = "/api/pgp/keys/" + oldKeyName
		location = "/api/pgp/keys/" + newKeyName
	)

	t.Run(`Testing the PGP key "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &PGPKeysUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":       newKeyName,
				"privateKey": privateKey,
				"publicKey":  publicKey,
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
					"--private-key", privateKey,
					"--public-key", publicKey),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The PGP key %q was successfully updated.\n", newKeyName),
					w.String(),
					"Then it should display a message saying the PGP key was updated",
				)
			})
		})
	})
}
