package wg

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestAuthorityAdd(t *testing.T) {
	const (
		authorityName     = "cert_authority"
		authorityType     = auth.AuthorityTLS
		authorityIdentity = testhelpers.LocalhostCert
		authorityHost1    = "1.2.3.4"
		authorityHost2    = "9.8.7.6"

		path     = "/api/authorities"
		location = path + "/" + authorityName
	)

	identityFile := filepath.Join(t.TempDir(), "test_authority_cert.pem")
	require.NoError(t, os.WriteFile(identityFile, []byte(authorityIdentity), 0o600))

	t.Run(`Testing the authority "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &AuthorityAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":           authorityName,
				"type":           authorityType,
				"publicIdentity": authorityIdentity,
				"validHosts":     []any{authorityHost1, authorityHost2},
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
					"--name", "cert_authority",
					"--type", auth.AuthorityTLS,
					"--identity-file", identityFile,
					"--host", authorityHost1, "--host", authorityHost2,
				), "Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The authority %q was successfully added.\n", authorityName),
					w.String(),
					"Then is should display a message saying the authority was added",
				)
			})
		})
	})
}

func TestAuthorityGet(t *testing.T) {
	const (
		authorityName     = "cert_authority"
		authorityType     = auth.AuthorityTLS
		authorityIdentity = testhelpers.LocalhostCert
		authorityHost1    = "1.2.3.4"
		authorityHost2    = "9.8.7.6"

		path = "/api/authorities/" + authorityName
	)

	t.Run(`Testing the authority "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &AuthorityGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":           authorityName,
				"type":           authorityType,
				"publicIdentity": authorityIdentity,
				"validHosts":     []string{authorityHost1, authorityHost2},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, authorityName),
					"Then it should not return an error")

				assert.Equal(t,
					expectedOutput(t, result.body,
						`‣Authority "{{.name}}"`,
						`  •Type: {{.type}}`,
						`  •Valid for hosts: {{ join .validHosts }}`,
						`  •Certificate "{{.name}}":`,
						`    ⁃Subject`,
						`      $Common Name: `,
						`      $Organization: Acme Co`,
						`    ⁃Issuer`,
						`      $Common Name: `,
						`      $Organization: Acme Co`,
						`    ⁃Validity`,
						`      $Not Before: Thu, 01 Jan 1970 00:00:00 UTC`,
						`      $Not After: Sat, 29 Jan 2084 16:00:00 UTC`,
						`    ⁃Subject Alt Names`,
						`      $DNS Names: localhost`,
						`      $IP Addresses: 127.0.0.1, ::1`,
						`    ⁃Public Key Info`,
						`      $Algorithm: RSA`,
						`      $Public Value: 30 82 01 0A 02 82 01 01 00 B7 C7 3F ED... (270 bytes total)`,
						`    ⁃Fingerprints`,
						`      $SHA-256: FB A0 34 52 2F BD CC CD AE D2 16 99 05 C9 11 71`,
						`                AB 31 55 DB 7C 6F 8B BB 6F 0D 49 0E EA 91 1D A6`,
						`      $SHA-1: E1 60 A8 18 84 47 5D 47 7A E7 77 83 DE CC A2 34 E8 61 74 3A`,
						`    ⁃Signature`,
						`      $Algorithm: SHA256-RSA`,
						`      $Public Value: A7 B8 06 56 98 75 F6 FA 5D 33 A6 45 CE... (256 bytes total)`,
						`    ⁃Serial Number: 55 4F 78 10 5C 39 1F 0E 17 42 A3 05 65 9C BA D6`,
						`    ⁃Key Usages: Digital Signature, Key Encipherment, Certificate Sign`,
						`    ⁃Extended Key Usages: Server Auth`,
					),
					w.String(),
					"Then is should display the authority's information",
				)
			})
		})
	})
}

func TestAuthorityList(t *testing.T) {
	const (
		sort   = "name+"
		limit  = "10"
		offset = "5"

		auth1Name     = "auth1"
		auth1Type     = auth.AuthorityTLS
		auth1Host1    = "1.2.3.4"
		auth1Host2    = "waarp.org"
		auth1Identity = testhelpers.LocalhostCert

		auth2Name     = "auth2"
		auth2Type     = auth.AuthorityTLS
		auth2Host1    = "9.8.7.6"
		auth2Host2    = "waarp.fr"
		auth2Identity = testhelpers.ClientFooCert

		path = "/api/authorities"
	)

	t.Run(`Testing the authority "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &AuthorityList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{
				"limit":  []string{limit},
				"offset": []string{offset},
				"sort":   []string{sort},
			},
		}

		authorities := []map[string]any{{
			"name":           auth1Name,
			"type":           auth1Type,
			"publicIdentity": auth1Identity,
			"validHosts":     []string{auth1Host1, auth1Host2},
		}, {
			"name":           auth2Name,
			"type":           auth2Type,
			"publicIdentity": auth2Identity,
			"validHosts":     []string{auth2Host1, auth2Host2},
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"authorities": authorities},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort))

				assert.Equal(t,
					expectedOutput(t, authorities,
						"=== Authentication authorities ===",
						`{{- with (index . 0) }}`,
						`‣Authority "{{.name}}"`,
						`  •Type: {{.type}}`,
						`  •Valid for hosts: {{ join .validHosts }}`,
						`  •Certificate "{{.name}}":`,
						`    ⁃Subject`,
						`      $Common Name: `,
						`      $Organization: Acme Co`,
						`    ⁃Issuer`,
						`      $Common Name: `,
						`      $Organization: Acme Co`,
						`    ⁃Validity`,
						`      $Not Before: Thu, 01 Jan 1970 00:00:00 UTC`,
						`      $Not After: Sat, 29 Jan 2084 16:00:00 UTC`,
						`    ⁃Subject Alt Names`,
						`      $DNS Names: localhost`,
						`      $IP Addresses: 127.0.0.1, ::1`,
						`    ⁃Public Key Info`,
						`      $Algorithm: RSA`,
						`      $Public Value: 30 82 01 0A 02 82 01 01 00 B7 C7 3F ED... (270 bytes total)`,
						`    ⁃Fingerprints`,
						`      $SHA-256: FB A0 34 52 2F BD CC CD AE D2 16 99 05 C9 11 71`,
						`                AB 31 55 DB 7C 6F 8B BB 6F 0D 49 0E EA 91 1D A6`,
						`      $SHA-1: E1 60 A8 18 84 47 5D 47 7A E7 77 83 DE CC A2 34 E8 61 74 3A`,
						`    ⁃Signature`,
						`      $Algorithm: SHA256-RSA`,
						`      $Public Value: A7 B8 06 56 98 75 F6 FA 5D 33 A6 45 CE... (256 bytes total)`,
						`    ⁃Serial Number: 55 4F 78 10 5C 39 1F 0E 17 42 A3 05 65 9C BA D6`,
						`    ⁃Key Usages: Digital Signature, Key Encipherment, Certificate Sign`,
						`    ⁃Extended Key Usages: Server Auth`,
						`{{- end }}`,
						`{{- with (index . 1) }}`,
						`‣Authority "{{.name}}"`,
						`  •Type: {{.type}}`,
						`  •Valid for hosts: {{ join .validHosts }}`,
						`  •Certificate "{{.name}}":`,
						`    ⁃Subject`,
						`      $Common Name: foo`,
						`      $Organization: Acme Co`,
						`    ⁃Issuer`,
						`      $Common Name: foo`,
						`      $Organization: Acme Co`,
						`    ⁃Validity`,
						`      $Not Before: Thu, 01 Jan 1970 00:00:00 UTC`,
						`      $Not After: Sat, 29 Jan 2084 16:00:00 UTC`,
						`    ⁃Subject Alt Names`,
						`      $Email Addresses: foo`,
						`    ⁃Public Key Info`,
						`      $Algorithm: RSA`,
						`      $Public Value: 30 81 89 02 81 81 00 D9 07 4F 33 5F 5C... (140 bytes total)`,
						`    ⁃Fingerprints`,
						`      $SHA-256: C4 68 01 A1 2B D2 1D E3 D0 35 36 6B AE AB 72 92`,
						`                A7 FA A4 11 AD C6 92 54 1C 01 08 51 25 AC 87 6E`,
						`      $SHA-1: F4 44 D2 A5 45 26 8B CC C9 8C 88 83 56 3C FF EB 7D D7 11 96`,
						`    ⁃Signature`,
						`      $Algorithm: SHA256-RSA`,
						`      $Public Value: 3D B9 C8 78 45 A2 D9 D3 81 3D AF 82 86... (128 bytes total)`,
						`    ⁃Serial Number: 77 6C B3 E2 F7 35 00 83 56 4F D9 5F 8E D0 54 78`,
						`    ⁃Key Usages: Digital Signature, Key Encipherment, Certificate Sign`,
						`    ⁃Extended Key Usages: Client Auth`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the authorities list",
				)
			})
		})
	})
}

func TestAuthorityUpdate(t *testing.T) {
	const (
		oldAuthorityName  = "old_cert_authority"
		authorityName     = "cert_authority"
		authorityType     = auth.AuthorityTLS
		authorityIdentity = testhelpers.LocalhostCert
		authorityHost1    = "1.2.3.4"
		authorityHost2    = "9.8.7.6"

		path     = "/api/authorities/" + oldAuthorityName
		location = "/api/authorities/" + authorityName
	)

	identityFile := filepath.Join(t.TempDir(), "test_authority_cert.pem")
	require.NoError(t, os.WriteFile(identityFile, []byte(authorityIdentity), 0o600))

	t.Run(`Testing the authority "update" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &AuthorityUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":           authorityName,
				"type":           authorityType,
				"publicIdentity": authorityIdentity,
				"validHosts":     []any{authorityHost1, authorityHost2},
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, oldAuthorityName,
					"--name", "cert_authority",
					"--type", auth.AuthorityTLS,
					"--identity-file", identityFile,
					"--host", authorityHost1, "--host", authorityHost2,
				), "Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The authority %q was successfully updated.\n", authorityName),
					w.String(),
					"Then is should display a message saying the authority was updated",
				)
			})
		})
	})
}

func TestAuthorityDelete(t *testing.T) {
	const (
		authorityName = "cert_authority"
		path          = "/api/authorities/" + authorityName
	)

	t.Run(`Testing the authority "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &AuthorityDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, authorityName),
					"Then it should not return an error",
				)
				assert.Equal(t,
					fmt.Sprintf("The authority %q was successfully deleted.\n", authorityName),
					w.String(),
					"Then it should display a message saying the authority was deleted",
				)
			})
		})
	})
}
