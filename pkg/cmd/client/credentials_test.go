package wg

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCredentialAdd(t *testing.T) {
	const (
		serverName = "test-server"

		credName = "server_cert"
		credType = auth.TLSCertificate
		credVal1 = testhelpers.LocalhostCert
		credVal2 = testhelpers.LocalhostCert

		path     = "/api/servers/" + serverName + "/credentials"
		location = path + "/" + credName
	)

	t.Run(`Given the credentials "add" command`, func(t *testing.T) {
		command := &CredentialAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":   credName,
				"type":   credType,
				"value":  credVal1,
				"value2": credVal2,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			Server = serverName
			defer resetVars()

			testServer(t, expected, result)

			t.Run("When executing the command with values", func(t *testing.T) {
				w := newTestOutput()

				require.NoError(t, executeCommand(t, w, command,
					"--name", credName,
					"--type", credType,
					"--value", credVal1,
					"--secondary-value", credVal2,
				), "Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The %q credential was successfully added.\n", credName),
					w.String(),
					"Then it should display a message saying the credential was added",
				)
			})

			t.Run("When executing the command with file paths", func(t *testing.T) {
				w := newTestOutput()
				file1 := writeFile(t, "val1.file", credVal1)
				file2 := writeFile(t, "val2.file", credVal2)

				require.NoError(t, executeCommand(t, w, command,
					"--name", credName,
					"--type", credType,
					"--value", file1,
					"--secondary-value", file2,
				), "Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The %q credential was successfully added.\n", credName),
					w.String(),
					"Then it should display a message saying the credential was added",
				)
			})
		})
	})
}

func TestCredentialGet(t *testing.T) {
	var command *CredentialGet

	const (
		server     = "toto"
		partner    = "tata"
		locAccount = "titi"
		remAccount = "tutu"

		pswdName  = "pswd"
		pswdType  = auth.Password
		pswdValue = "sesame"

		certName  = "tls-cert"
		certType  = auth.TLSCertificate
		certValue = testhelpers.LocalhostCert

		sshKeyName    = "ssh-key"
		sshKeyType    = sftp.AuthSSHPublicKey
		sshKeyContent = testhelpers.SSHPbk

		pkeyName    = "pkey"
		pkeyType    = sftp.AuthSSHPrivateKey
		pkeyContent = testhelpers.RSAPk

		certInfo = `── Certificate "` + certName + `"
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
   ╰─ Extended Key Usages: Server Auth
`

		sshKeyInfo = `── SSH Public Key "` + sshKeyName + `"
   ├─ Type: ssh-rsa
   ├─ SHA-256 Fingerprint: SHA256:WvBFQfpPo+17qsVgYnyBPY0j2KTDjMLvqRFnjjYLFNk
   ╰─ MD5 Fingerprint: 5b:22:c5:76:75:d0:60:8f:f8:8f:7e:e9:b7:46:c6:16
`

		pKeyInfo = `── Private Key "` + pkeyName + `"
   ├─ Type: ssh-rsa
   ├─ SHA-256 Fingerprint: SHA256:WvBFQfpPo+17qsVgYnyBPY0j2KTDjMLvqRFnjjYLFNk
   ╰─ MD5 Fingerprint: 5b:22:c5:76:75:d0:60:8f:f8:8f:7e:e9:b7:46:c6:16
`
	)

	t.Run(`Testing the credential "get" command with a server`, func(t *testing.T) {
		const path = "/api/servers/" + server + "/credentials/" + certName

		Server = server
		defer resetVars()

		w := newTestOutput()
		command = &CredentialGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":  certName,
				"type":  certType,
				"value": certValue,
			},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, certName))

				assert.Equal(t, certInfo, w.String(),
					"Then it should display the credential")
			})
		})
	})

	t.Run(`Testing the credential "get" command with a partner`, func(t *testing.T) {
		const path = "/api/partners/" + partner + "/credentials/" + pswdName

		Partner = partner
		defer resetVars()

		w := newTestOutput()
		command = &CredentialGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":  pswdName,
				"type":  pswdType,
				"value": pswdValue,
			},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, pswdName))

				assert.Equal(t,
					fmt.Sprintf("── Password %q: %s\n", pswdName, pswdValue),
					w.String(),
					"Then it should display the public key")
			})
		})
	})

	t.Run(`Testing the credential "get" command with a local account`, func(t *testing.T) {
		const path = "/api/servers/" + server + "/accounts/" + locAccount + "/credentials/" + sshKeyName

		Server = server
		LocalAccount = locAccount
		defer resetVars()

		w := newTestOutput()
		command = &CredentialGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":  sshKeyName,
				"type":  sshKeyType,
				"value": sshKeyContent,
			},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, sshKeyName))

				assert.Equal(t, sshKeyInfo, w.String(),
					"Then it should display the public key")
			})
		})
	})

	t.Run(`Testing the credential "get" command with a remote account`, func(t *testing.T) {
		const path = "/api/partners/" + partner + "/accounts/" + remAccount + "/credentials/" + pkeyName

		Partner = partner
		RemoteAccount = remAccount
		defer resetVars()

		w := newTestOutput()
		command = &CredentialGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":  pkeyName,
				"type":  pkeyType,
				"value": pkeyContent,
			},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, pkeyName))

				assert.Equal(t, pKeyInfo, w.String(),
					"Then it should display the private key")
			})
		})
	})
}

func TestCredentialDelete(t *testing.T) {
	const (
		serverName = "test-server"

		credName = "server_cert"
		path     = "/api/servers/" + serverName + "/credentials/" + credName
	)

	t.Run(`Given the credential "delete"" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &CredentialDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			Server = serverName
			defer resetVars()

			testServer(t, expected, result)

			t.Run("When executing the command with file paths", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, credName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The %q credential was successfully removed.\n", credName),
					w.String(),
					"Then it should display a message saying the credential was deleted",
				)
			})
		})
	})
}
