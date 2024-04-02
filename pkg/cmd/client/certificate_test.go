package wg

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
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
	)

	tlsCerts, err := utils.ParsePEMCertChain(certContent)
	require.NoError(t, err, "could not parse certificate")

	tlsCert := tlsCerts[0]

	sshKey, err := utils.ParseSSHAuthorizedKey(sshKeyContent)
	require.NoError(t, err, "could not parse SSH public key")

	pkey, err := ssh.ParsePrivateKey([]byte(pkeyContent))
	require.NoError(t, err, "could not parse private key")

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
					fmt.Sprintf("── Certificate %q\n", certName)+
						fmt.Sprintf("   ├─ Subject\n")+
						fmt.Sprintf("   │  ├─ Common name: %s\n", tlsCert.Subject.CommonName)+
						fmt.Sprintf("   │  ╰─ Organization: %s\n", tlsCert.Subject.Organization)+
						fmt.Sprintf("   ├─ Issuer\n")+
						fmt.Sprintf("   │  ├─ Common name: %s\n", tlsCert.Issuer.CommonName)+
						fmt.Sprintf("   │  ╰─ Organization: %s\n", tlsCert.Issuer.Organization)+
						fmt.Sprintf("   ├─ Validity\n")+
						fmt.Sprintf("   │  ├─ Not before: %s\n", tlsCert.NotBefore.Format(time.UnixDate))+
						fmt.Sprintf("   │  ╰─ Not after: %s\n", tlsCert.NotAfter.Format(time.UnixDate))+
						fmt.Sprintf("   ├─ Subject Alt Names\n")+
						fmt.Sprintf("   │  ╰─ DNS name: %s\n", tlsCert.DNSNames[0])+
						fmt.Sprintf("   ├─ Public Key Algorithm: %s\n", tlsCert.PublicKeyAlgorithm)+
						fmt.Sprintf("   ├─ Signature Algorithm: %s\n", tlsCert.SignatureAlgorithm)+
						fmt.Sprintf("   ├─ Signature: %s\n", fmt.Sprintf("%X", tlsCert.Signature))+
						fmt.Sprintf("   ├─ Key Usages: %s\n",
							strings.Join(utils.KeyUsageToStrings(tlsCert.KeyUsage), ", "))+
						fmt.Sprintf("   ╰─ Extended Key Usages: %s\n",
							strings.Join(utils.ExtKeyUsagesToStrings(tlsCert.ExtKeyUsage), ", ")),
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
					fmt.Sprintf("── SSH Public Key %q\n", sshKeyName)+
						fmt.Sprintf("   ├─ Type: %s\n", sshKey.Type())+
						fmt.Sprintf("   ╰─ Fingerprint: %s\n", ssh.FingerprintSHA256(sshKey)),
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
					fmt.Sprintf("── SSH Public Key %q\n", sshKeyName)+
						fmt.Sprintf("   ├─ Type: %s\n", sshKey.Type())+
						fmt.Sprintf("   ╰─ Fingerprint: %s\n", ssh.FingerprintSHA256(sshKey)),
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
					fmt.Sprintf("── Private Key %q\n", pkeyName)+
						fmt.Sprintf("   ├─ Type: %s\n", pkey.PublicKey().Type())+
						fmt.Sprintf("   ╰─ Fingerprint: %s\n", ssh.FingerprintSHA256(pkey.PublicKey())),
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

	parsedKey1, err := ssh.ParsePrivateKey([]byte(pkey1))
	require.NoError(t, err)
	parsedKey2, err := ssh.ParsePrivateKey([]byte(pkey2))
	require.NoError(t, err)

	var (
		key1Type        = parsedKey1.PublicKey().Type()
		key2Type        = parsedKey2.PublicKey().Type()
		key1FingerPrint = ssh.FingerprintSHA256(parsedKey1.PublicKey())
		key2FingerPrint = ssh.FingerprintSHA256(parsedKey2.PublicKey())
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

	listOutput := "Certificates:\n" +
		fmt.Sprintf("╭─ Private Key %q\n", pkey1Name) +
		fmt.Sprintf("│  ├─ Type: %s\n", key1Type) +
		fmt.Sprintf("│  ╰─ Fingerprint: %s\n", key1FingerPrint) +
		fmt.Sprintf("╰─ Private Key %q\n", pkey2Name) +
		fmt.Sprintf("   ├─ Type: %s\n", key2Type) +
		fmt.Sprintf("   ╰─ Fingerprint: %s\n", key2FingerPrint)

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
