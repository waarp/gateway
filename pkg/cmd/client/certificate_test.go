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

func resetVars() {
	Server = ""
	Partner = ""
	LocalAccount = ""
	RemoteAccount = ""
}

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

				assert.Equal(t, certInfo, w.String(),
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

				assert.Equal(t, sshKeyInfo, w.String(),
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

				assert.Equal(t, sshKeyInfo, w.String(),
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

				assert.Equal(t, pKeyInfo, w.String(),
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

		listOutput = `Certificates:
╭─ Private Key "` + pkey1Name + `"
│  ├─ Type: ssh-rsa
│  ├─ SHA-256 Fingerprint: SHA256:WvBFQfpPo+17qsVgYnyBPY0j2KTDjMLvqRFnjjYLFNk
│  ╰─ MD5 Fingerprint: 5b:22:c5:76:75:d0:60:8f:f8:8f:7e:e9:b7:46:c6:16
╰─ Private Key "` + pkey2Name + `"
   ├─ Type: ssh-rsa
   ├─ SHA-256 Fingerprint: SHA256:qRP7bkkwOS3DjaMYPvxA6h63wv/+gqcqHban855E3/I
   ╰─ MD5 Fingerprint: f0:63:df:e5:bb:ab:9c:c7:56:29:65:12:0b:c3:00:db
`
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
