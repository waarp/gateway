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

		identityInfo = `╰─ Certificate "` + authorityName + `"
      ├─ Subject
      │  ├─ Common Name: 
      │  ╰─ Organization: Acme Co
      ├─ Issuer
      │  ├─ Common Name: 
      │  ╰─ Organization: Acme Co
      ├─ Validity
      │  ├─ Not Before: Thu, 01 Jan 1970 00:00:00 UTC
      │  ╰─ Not After: Sat, 29 Jan 2084 16:00:00 UTC
      ├─ Subject Alt Names
      │  ├─ DNS Name: localhost
      │  ├─ IP Address: 127.0.0.1
      │  ╰─ IP Address: ::1
      ├─ Public Key Info
      │  ├─ Algorithm: RSA
      │  ╰─ Public Value:  30 81 89 02 81 81 00 B8 0E 75 4C 68 64 BA 37 9D DC A2 9C 57 BA F7 AC 96 7F 1B 22
      │     52 E5 CF B5 23 F0 03 46 DC 24 73 3A ED 6E 9F F7 39 D6 47 6C 24 40 4D 2E 7B FC D3 D6 3E 35 D9 B6
      │     6D 6A 46 50 7B D4 72 EC 6C C5 37 E5 01 5D 51 94 FA 4B 5C 0A BB 97 0C D2 9E A9 00 B1 9A 32 D6 81
      │     0F 90 68 97 B1 BA 7D 72 4C E8 48 90 CF A3 45 81 2D 72 F8 6F B5 5D 13 EF 95 88 7B 95 F6 9B 27 BB
      │     95 0D 6F 37 37 C4 55 CE F8 91 5E 47 02 03 01 00 01
      ├─ Fingerprints
      │  ├─ SHA-256: 78 20 F0 9E 64 61 2E 4B 67 76 28 80 EE 36 B4 84 5C F0 CF AB 69 C6 DB 38 6F E1 79 D3 48 49 FB 14
      │  ╰─ SHA-1: 11 AF D3 5F 24 B4 42 BE 09 2B 37 49 41 2B 40 03 C1 33 C5 2C
      ├─ Signature
      │  ├─ Algorithm: SHA256-RSA
      │  ╰─ Public Value:  78 94 3D 08 4B 8D 94 58 DF AE 85 DC 5F F4 2E 80 A5 A8 A1 C9 C8 E1 2E D4 73 CB 08
      │     C1 6C 1D 04 92 CB 58 1E F2 34 15 2F 0E E3 99 3E EF E4 39 CD DB D6 F2 4B C1 01 0D EE 0C B2 78 BA
      │     AE D8 34 E2 58 CB 0F 3E 73 68 DF D0 2A 7A 97 FA 9F 2A 3A 7A 3B 7B 47 78 E0 36 E3 1C 3D DB B5 6F
      │     E8 2C EC 39 9D F7 71 77 5B 54 C6 1F 14 E3 87 63 0F CF 74 8E 4D 35 DE C6 9B CD A4 25 BA E1 30 D3
      │     C8 E5 E2 39 9D
      ├─ Serial Number: B0 55 E8 BB DB BA AE 98 EE 65 0E CA 76 B1 6F 33
      ├─ Key Usages: Digital Signature, Key Encipherment, Certificate Sign
      ╰─ Extended Key Usages: Server Auth`
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
					fmt.Sprintf("── Authority %q\n", authorityName)+
						fmt.Sprintf("   ├─ Type: %s\n", authorityType)+
						fmt.Sprintf("   ├─ Valid Hosts: %s, %s\n", authorityHost1, authorityHost2)+
						fmt.Sprintf("   %s\n", identityInfo),
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

		auth1Info = `╰─ Certificate "` + auth1Name + `"
│     ├─ Subject
│     │  ├─ Common Name: 
│     │  ╰─ Organization: Acme Co
│     ├─ Issuer
│     │  ├─ Common Name: 
│     │  ╰─ Organization: Acme Co
│     ├─ Validity
│     │  ├─ Not Before: Thu, 01 Jan 1970 00:00:00 UTC
│     │  ╰─ Not After: Sat, 29 Jan 2084 16:00:00 UTC
│     ├─ Subject Alt Names
│     │  ├─ DNS Name: localhost
│     │  ├─ IP Address: 127.0.0.1
│     │  ╰─ IP Address: ::1
│     ├─ Public Key Info
│     │  ├─ Algorithm: RSA
│     │  ╰─ Public Value:  30 81 89 02 81 81 00 B8 0E 75 4C 68 64 BA 37 9D DC A2 9C 57 BA F7 AC 96 7F 1B 22
│     │     52 E5 CF B5 23 F0 03 46 DC 24 73 3A ED 6E 9F F7 39 D6 47 6C 24 40 4D 2E 7B FC D3 D6 3E 35 D9 B6
│     │     6D 6A 46 50 7B D4 72 EC 6C C5 37 E5 01 5D 51 94 FA 4B 5C 0A BB 97 0C D2 9E A9 00 B1 9A 32 D6 81
│     │     0F 90 68 97 B1 BA 7D 72 4C E8 48 90 CF A3 45 81 2D 72 F8 6F B5 5D 13 EF 95 88 7B 95 F6 9B 27 BB
│     │     95 0D 6F 37 37 C4 55 CE F8 91 5E 47 02 03 01 00 01
│     ├─ Fingerprints
│     │  ├─ SHA-256: 78 20 F0 9E 64 61 2E 4B 67 76 28 80 EE 36 B4 84 5C F0 CF AB 69 C6 DB 38 6F E1 79 D3 48 49 FB 14
│     │  ╰─ SHA-1: 11 AF D3 5F 24 B4 42 BE 09 2B 37 49 41 2B 40 03 C1 33 C5 2C
│     ├─ Signature
│     │  ├─ Algorithm: SHA256-RSA
│     │  ╰─ Public Value:  78 94 3D 08 4B 8D 94 58 DF AE 85 DC 5F F4 2E 80 A5 A8 A1 C9 C8 E1 2E D4 73 CB 08
│     │     C1 6C 1D 04 92 CB 58 1E F2 34 15 2F 0E E3 99 3E EF E4 39 CD DB D6 F2 4B C1 01 0D EE 0C B2 78 BA
│     │     AE D8 34 E2 58 CB 0F 3E 73 68 DF D0 2A 7A 97 FA 9F 2A 3A 7A 3B 7B 47 78 E0 36 E3 1C 3D DB B5 6F
│     │     E8 2C EC 39 9D F7 71 77 5B 54 C6 1F 14 E3 87 63 0F CF 74 8E 4D 35 DE C6 9B CD A4 25 BA E1 30 D3
│     │     C8 E5 E2 39 9D
│     ├─ Serial Number: B0 55 E8 BB DB BA AE 98 EE 65 0E CA 76 B1 6F 33
│     ├─ Key Usages: Digital Signature, Key Encipherment, Certificate Sign
│     ╰─ Extended Key Usages: Server Auth`

		auth2Info = `╰─ Certificate "` + auth2Name + `"
      ├─ Subject
      │  ├─ Common Name: foo
      │  ╰─ Organization: Acme Co
      ├─ Issuer
      │  ├─ Common Name: foo
      │  ╰─ Organization: Acme Co
      ├─ Validity
      │  ├─ Not Before: Thu, 01 Jan 1970 00:00:00 UTC
      │  ╰─ Not After: Sat, 29 Jan 2084 16:00:00 UTC
      ├─ Subject Alt Names
      │  ╰─ Email Address: foo
      ├─ Public Key Info
      │  ├─ Algorithm: RSA
      │  ╰─ Public Value:  30 81 89 02 81 81 00 D9 07 4F 33 5F 5C 0C 45 DB F3 8A BB 21 F5 14 E1 C7 EA 05 CF
      │     34 EA A9 08 72 55 10 57 10 63 47 8F AE 86 7D 16 B5 17 A6 ED 6B 18 28 AF 98 3A 76 4D C3 61 C6 82
      │     A9 DA B0 84 26 90 A4 CF 5D 5B 96 12 22 3F FE 41 68 05 6E C6 FF B2 A4 B5 6C 16 3C 88 2C 5A 68 7C
      │     FC 59 2E 25 4A 57 BB 7A 61 7A 2B 4A 1D C5 26 3F D4 FF 8E E2 FB D8 37 72 DB 58 97 98 24 64 57 FA
      │     82 A3 46 97 70 C7 4E 70 09 7E FC 53 02 03 01 00 01
      ├─ Fingerprints
      │  ├─ SHA-256: C4 68 01 A1 2B D2 1D E3 D0 35 36 6B AE AB 72 92 A7 FA A4 11 AD C6 92 54 1C 01 08 51 25 AC 87 6E
      │  ╰─ SHA-1: F4 44 D2 A5 45 26 8B CC C9 8C 88 83 56 3C FF EB 7D D7 11 96
      ├─ Signature
      │  ├─ Algorithm: SHA256-RSA
      │  ╰─ Public Value:  3D B9 C8 78 45 A2 D9 D3 81 3D AF 82 86 F7 34 DF B9 5D 5D A2 FB 0E 26 34 98 97 A7
      │     57 A7 A0 BF AA 29 4C 39 10 74 DF 97 16 54 3C AE 39 38 BF FA 09 5C C3 B2 26 D5 78 4F 8F 0A B0 5B
      │     61 A8 5A 5E E9 53 0A A2 04 39 0D 09 54 7B DF 4D 0E 34 42 D4 34 88 31 B9 C1 AB 2D 85 67 62 DE F2
      │     25 6B 0B 0C C1 89 A0 2F B4 41 64 94 2F F3 0F 77 B6 D2 CB 16 93 98 59 01 7C 84 8B 25 94 E2 9F BF
      │     CF 86 09 ED E9
      ├─ Serial Number: 77 6C B3 E2 F7 35 00 83 56 4F D9 5F 8E D0 54 78
      ├─ Key Usages: Digital Signature, Key Encipherment, Certificate Sign
      ╰─ Extended Key Usages: Client Auth`
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

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"authorities": []map[string]any{{
					"name":           auth1Name,
					"type":           auth1Type,
					"publicIdentity": auth1Identity,
					"validHosts":     []string{auth1Host1, auth1Host2},
				}, {
					"name":           auth2Name,
					"type":           auth2Type,
					"publicIdentity": auth2Identity,
					"validHosts":     []string{auth2Host1, auth2Host2},
				}},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort))

				assert.Equal(t, "Authentication authorities:\n"+
					fmt.Sprintf("╭─ Authority %q\n", auth1Name)+
					fmt.Sprintf("│  ├─ Type: %s\n", auth1Type)+
					fmt.Sprintf("│  ├─ Valid Hosts: %s, %s\n", auth1Host1, auth1Host2)+
					fmt.Sprintf("│  %s\n", auth1Info)+
					fmt.Sprintf("╰─ Authority %q\n", auth2Name)+
					fmt.Sprintf("   ├─ Type: %s\n", auth2Type)+
					fmt.Sprintf("   ├─ Valid Hosts: %s, %s\n", auth2Host1, auth2Host2)+
					fmt.Sprintf("   %s\n", auth2Info),
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
