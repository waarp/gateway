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

				assert.Equal(t,
					expectedOutput(t, result.body,
						`‣Certificate "{{.name}}":`,
						`  •Subject`,
						`    ⁃Common Name: `,
						`    ⁃Organization: Acme Co`,
						`  •Issuer`,
						`    ⁃Common Name: `,
						`    ⁃Organization: Acme Co`,
						`  •Validity`,
						`    ⁃Not Before: Thu, 01 Jan 1970 00:00:00 UTC`,
						`    ⁃Not After: Sat, 29 Jan 2084 16:00:00 UTC`,
						`  •Subject Alt Names`,
						`    ⁃DNS Names: localhost`,
						`    ⁃IP Addresses: 127.0.0.1, ::1`,
						`  •Public Key Info`,
						`    ⁃Algorithm: RSA`,
						`    ⁃Public Value: 30 82 01 0A 02 82 01 01 00 B7 C7 3F ED... (270 bytes total)`,
						`  •Fingerprints`,
						`    ⁃SHA-256: FB A0 34 52 2F BD CC CD AE D2 16 99 05 C9 11 71`,
						`              AB 31 55 DB 7C 6F 8B BB 6F 0D 49 0E EA 91 1D A6`,
						`    ⁃SHA-1: E1 60 A8 18 84 47 5D 47 7A E7 77 83 DE CC A2 34 E8 61 74 3A`,
						`  •Signature`,
						`    ⁃Algorithm: SHA256-RSA`,
						`    ⁃Public Value: A7 B8 06 56 98 75 F6 FA 5D 33 A6 45 CE... (256 bytes total)`,
						`  •Serial Number: 55 4F 78 10 5C 39 1F 0E 17 42 A3 05 65 9C BA D6`,
						`  •Key Usages: Digital Signature, Key Encipherment, Certificate Sign`,
						`  •Extended Key Usages: Server Auth`,
					),
					w.String(),
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
					fmt.Sprintf("‣Password %q: %s\n", pswdName, pswdValue),
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

				assert.Equal(t,
					expectedOutput(t, result.body,
						`‣SSH Public Key "{{.name}}":`,
						`  •Type: ssh-rsa`,
						`  •SHA-256 Fingerprint: SHA256:WvBFQfpPo+17qsVgYnyBPY0j2KTDjMLvqRFnjjYLFNk`,
						`  •MD5 Fingerprint: 5b:22:c5:76:75:d0:60:8f:f8:8f:7e:e9:b7:46:c6:16`,
					),
					w.String(),
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

				assert.Equal(t,
					expectedOutput(t, result.body,
						`‣Private Key "{{.name}}":`,
						`  •Type: ssh-rsa`,
						`  •SHA-256 Fingerprint: SHA256:WvBFQfpPo+17qsVgYnyBPY0j2KTDjMLvqRFnjjYLFNk`,
						`  •MD5 Fingerprint: 5b:22:c5:76:75:d0:60:8f:f8:8f:7e:e9:b7:46:c6:16`,
					),
					w.String(),
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
