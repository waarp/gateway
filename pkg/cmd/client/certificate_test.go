package wg

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCertificateGet(t *testing.T) {
	var command *CertGet

	const (
		server     = "toto"
		partner    = "tata"
		locAccount = "titi"
		remAccount = "tutu"

		certName    = "tls-cert"
		certContent = testhelpers.LocalhostCert

		sshKeyName    = "ssh-key"
		sshKeyContent = testhelpers.SSHPbk

		pkeyName    = "pkey"
		pkeyContent = testhelpers.RSAPk
	)

	t.Run(`Testing the certificate "get" command with a server`, func(t *testing.T) {
		const path = "/api/servers/" + server + "/certificates/" + certName

		Server = server
		defer resetVars()

		w := newTestOutput()
		command = &CertGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":        certName,
				"certificate": certContent,
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
					"Then it should display the certificate")
			})
		})
	})

	t.Run(`Testing the certificate "get" command with a partner`, func(t *testing.T) {
		const path = "/api/partners/" + partner + "/certificates/" + sshKeyName

		Partner = partner
		defer resetVars()

		w := newTestOutput()
		command = &CertGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":      sshKeyName,
				"publicKey": sshKeyContent,
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

	t.Run(`Testing the certificate "get" command with a local account`, func(t *testing.T) {
		const path = "/api/servers/" + server + "/accounts/" + locAccount + "/certificates/" + sshKeyName

		Server = server
		LocalAccount = locAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":      sshKeyName,
				"publicKey": sshKeyContent,
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

	t.Run(`Testing the certificate "get" command with a remote account`, func(t *testing.T) {
		const path = "/api/partners/" + partner + "/accounts/" + remAccount + "/certificates/" + pkeyName

		Partner = partner
		RemoteAccount = remAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":       pkeyName,
				"privateKey": pkeyContent,
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

func TestCertificateAdd(t *testing.T) {
	var command *CertAdd

	const (
		server     = "toto"
		partner    = "tata"
		locAccount = "titi"
		remAccount = "tutu"

		name        = "new_crypto"
		certContent = testhelpers.LocalhostCert
		pkey        = testhelpers.LocalhostKey
		sshKey      = testhelpers.SSHPbk
	)

	certFile := writeFile(t, "cert.pem", certContent)
	pkeyFile := writeFile(t, "key.pem", pkey)
	sshKeyFile := writeFile(t, "id_rsa", sshKey)

	t.Run(`Testing the cert "add" command with a server`, func(t *testing.T) {
		const (
			path     = "/api/servers/" + server + "/certificates"
			location = path + "/" + name
		)

		Server = server
		defer resetVars()

		w := newTestOutput()
		command = &CertAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":        name,
				"certificate": certContent,
				"privateKey":  pkey,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", name,
					"--certificate", certFile,
					"--private_key", pkeyFile))

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully added.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was added")
			})
		})
	})

	t.Run(`Testing the cert "add" command with a partner`, func(t *testing.T) {
		const (
			path     = "/api/partners/" + partner + "/certificates"
			location = path + "/" + name
		)

		Partner = partner
		defer resetVars()

		w := newTestOutput()
		command = &CertAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":      name,
				"publicKey": sshKey,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", name,
					"--public_key", sshKeyFile))

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully added.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was added")
			})
		})
	})

	t.Run(`Testing the cert "add" command with a local account`, func(t *testing.T) {
		const (
			path     = "/api/servers/" + server + "/accounts/" + locAccount + "/certificates"
			location = path + "/" + name
		)

		Server = server
		LocalAccount = locAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":      name,
				"publicKey": sshKey,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", name,
					"--public_key", sshKeyFile))

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully added.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was added")
			})
		})
	})

	t.Run(`Testing the cert "add" command with a remote account`, func(t *testing.T) {
		const (
			path     = "/api/partners/" + partner + "/accounts/" + remAccount + "/certificates"
			location = path + "/" + name
		)

		Partner = partner
		RemoteAccount = remAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":        name,
				"certificate": certContent,
				"privateKey":  pkey,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", name,
					"--certificate", certFile,
					"--private_key", pkeyFile))

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully added.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was added")
			})
		})
	})
}

func TestCertificateDelete(t *testing.T) {
	var command *CertDelete

	const (
		server     = "toto"
		partner    = "tata"
		locAccount = "titi"
		remAccount = "tutu"

		name = "crypto"
	)

	t.Run(`Testing the certificate "delete" command with a server`, func(t *testing.T) {
		const path = "/api/servers/" + server + "/certificates/" + name

		Server = server
		defer resetVars()

		w := newTestOutput()
		command = &CertDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name))

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully deleted.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was deleted")
			})
		})
	})

	t.Run(`Testing the certificate "delete" command with a partner`, func(t *testing.T) {
		const path = "/api/partners/" + partner + "/certificates/" + name

		Partner = partner
		defer resetVars()

		w := newTestOutput()
		command = &CertDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name))

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully deleted.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was deleted")
			})
		})
	})

	t.Run(`Testing the certificate "delete" command with a local account`, func(t *testing.T) {
		const path = "/api/servers/" + server + "/accounts/" + locAccount + "/certificates/" + name

		Server = server
		LocalAccount = locAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name))

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully deleted.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was deleted")
			})
		})
	})

	t.Run(`Testing the certificate "delete" command with a remote account`, func(t *testing.T) {
		const path = "/api/partners/" + partner + "/accounts/" + remAccount + "/certificates/" + name

		Partner = partner
		RemoteAccount = remAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, name))

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully deleted.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was deleted")
			})
		})
	})
}

func TestCertificateList(t *testing.T) {
	var command *CertList

	const (
		sort   = "name+"
		limit  = "10"
		offset = "5"

		server     = "toto"
		partner    = "tata"
		locAccount = "titi"
		remAccount = "tutu"

		pkey1Name = "pkey1"
		pkey1     = testhelpers.RSAPk
		pkey2Name = "pkey2"
		pkey2     = testhelpers.LocalhostKey
	)

	keyList := map[string]any{
		"certificates": []map[string]any{{
			"name":       pkey1Name,
			"privateKey": pkey1,
		}, {
			"name":       pkey2Name,
			"privateKey": pkey2,
		}},
	}

	listOutput := expectedOutput(t, keyList["certificates"],
		`=== Certificates ===`,
		`{{- with (index . 0) }}`,
		`‣Private Key "{{.name}}":`,
		`  •Type: ssh-rsa`,
		`  •SHA-256 Fingerprint: SHA256:WvBFQfpPo+17qsVgYnyBPY0j2KTDjMLvqRFnjjYLFNk`,
		`  •MD5 Fingerprint: 5b:22:c5:76:75:d0:60:8f:f8:8f:7e:e9:b7:46:c6:16`,
		`{{- end }}`,
		`{{- with (index . 1) }}`,
		`‣Private Key "{{.name}}":`,
		`  •Type: ssh-rsa`,
		`  •SHA-256 Fingerprint: SHA256:Dej683aqkWUH0xTngdOTovoPfEfYjVIFB0yDNBxck6g`,
		`  •MD5 Fingerprint: 19:db:ad:45:34:32:15:d8:b5:f1:0e:b2:9a:8a:a2:f5`,
		`{{- end }}`,
	)

	t.Run(`Testing the certificate "list" command with a server`, func(t *testing.T) {
		const path = "/api/servers/" + server + "/certificates"

		Server = server
		defer resetVars()

		w := newTestOutput()
		command = &CertList{}

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
			body:   keyList,
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort))

				assert.Equal(t,
					listOutput,
					w.String(),
					"Then it should display the certificate")
			})
		})
	})

	t.Run(`Testing the certificate "list" command with a partner`, func(t *testing.T) {
		const path = "/api/partners/" + partner + "/certificates"

		Partner = partner
		defer resetVars()

		w := newTestOutput()
		command = &CertList{}

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
			body:   keyList,
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort))

				assert.Equal(t,
					listOutput,
					w.String(),
					"Then it should display the certificate")
			})
		})
	})

	t.Run(`Testing the certificate "list" command with a local account`, func(t *testing.T) {
		const path = "/api/servers/" + server + "/accounts/" + locAccount + "/certificates"

		Server = server
		LocalAccount = locAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertList{}

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
			body:   keyList,
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort))

				assert.Equal(t,
					listOutput,
					w.String(),
					"Then it should display the certificate")
			})
		})
	})

	t.Run(`Testing the certificate "list" command with a remote account`, func(t *testing.T) {
		const path = "/api/partners/" + partner + "/accounts/" + remAccount + "/certificates"

		Partner = partner
		RemoteAccount = remAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertList{}

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
			body:   keyList,
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort))

				assert.Equal(t,
					listOutput,
					w.String(),
					"Then it should display the certificate")
			})
		})
	})
}

func TestCertificateUpdate(t *testing.T) {
	var command *CertUpdate

	const (
		server     = "toto"
		partner    = "tata"
		locAccount = "titi"
		remAccount = "tutu"

		oldName     = "old_crypto"
		name        = "new_crypto"
		certContent = testhelpers.LocalhostCert
		pkey        = testhelpers.LocalhostKey
		sshKey      = testhelpers.SSHPbk
	)

	certFile := writeFile(t, "cert.pem", certContent)
	pkeyFile := writeFile(t, "key.pem", pkey)
	sshKeyFile := writeFile(t, "id_rsa", sshKey)

	t.Run(`Testing the cert "update" command with a server`, func(t *testing.T) {
		const (
			path     = "/api/servers/" + server + "/certificates/" + oldName
			location = "/api/servers/" + server + "/certificates/" + name
		)

		Server = server
		defer resetVars()

		w := newTestOutput()
		command = &CertUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":        name,
				"certificate": certContent,
				"privateKey":  pkey,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", name,
					"--certificate", certFile,
					"--private_key", pkeyFile,
					oldName,
				),
					"Then is should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully updated.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was updated")
			})
		})
	})

	t.Run(`Testing the cert "update" command with a partner`, func(t *testing.T) {
		const (
			path     = "/api/partners/" + partner + "/certificates/" + oldName
			location = "/api/partners/" + partner + "/certificates/" + name
		)

		Partner = partner
		defer resetVars()

		w := newTestOutput()
		command = &CertUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":      name,
				"publicKey": sshKey,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", name,
					"--public_key", sshKeyFile,
					oldName,
				),
					"Then is should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully updated.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was updated")
			})
		})
	})

	t.Run(`Testing the cert "update" command with a local account`, func(t *testing.T) {
		const (
			path     = "/api/servers/" + server + "/accounts/" + locAccount + "/certificates/" + oldName
			location = "/api/servers/" + server + "/accounts/" + locAccount + "/certificates/" + name
		)

		Server = server
		LocalAccount = locAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":      name,
				"publicKey": sshKey,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", name,
					"--public_key", sshKeyFile,
					oldName,
				),
					"Then is should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully updated.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was updated")
			})
		})
	})

	t.Run(`Testing the cert "update" command with a remote account`, func(t *testing.T) {
		const (
			path     = "/api/partners/" + partner + "/accounts/" + remAccount + "/certificates/" + oldName
			location = "/api/partners/" + partner + "/accounts/" + remAccount + "/certificates/" + name
		)

		Partner = partner
		RemoteAccount = remAccount
		defer resetVars()

		w := newTestOutput()
		command = &CertUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":        name,
				"certificate": certContent,
				"privateKey":  pkey,
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", name,
					"--certificate", certFile,
					"--private_key", pkeyFile,
					oldName,
				),
					"Then is should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The certificate %q was successfully updated.\n", name),
					w.String(),
					"Then it should display a message saying the certificate was updated")
			})
		})
	})
}
