package config

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/fs/backends" // the library of the storages
)

// Every secret and every certificate of an export takes the marker of the
// migration tooling, with where it belongs; what is not a secret stays.
func TestRedactSecrets(t *testing.T) {
	t.Parallel()

	data := &file.Data{
		Locals: []file.LocalAgent{{
			Name: "srv/https:443", Protocol: "https",
			Credentials: []file.Credential{
				{Name: "cert", Type: "tls_certificate", Value: "CERT-PEM", Value2: "KEY-PEM"},
				{Name: "trusted", Type: "trusted_tls_certificate", Value: "CA-PEM"},
			},
			Certificates: []file.Certificate{{Name: "old", PublicKey: "PUB", PrivateKey: "PRIV", Certificate: "CRT"}},
			Accounts: []file.LocalAccount{{
				Login: "filiale", Password: api.NewNullable("clear-password"), PasswordHash: "$2a$hash",
				Credentials: []file.Credential{
					{Name: "password", Type: "password", Value: "clear-password"},
					{Name: "ssh", Type: "ssh_public_key", Value: "ssh-rsa AAAA"},
				},
			}},
		}},
		Remotes: []file.RemoteAgent{{
			Name: "banque-demo", Protocol: "pesit-tls",
			Credentials: []file.Credential{{Name: "legacy", Type: "r66_legacy_certificate", Value: "CRT", Value2: "KEY"}},
			Accounts: []file.RemoteAccount{{
				Login: "prod", Password: api.NewNullable("p"),
				Credentials: []file.Credential{
					{Name: "pre", Type: "pesit_pre-connection_auth", Value: "login-pre", Value2: "pass-pre"},
					{Name: "key", Type: "ssh_private_key", Value: "PRIVATE"},
					{Name: "odd", Type: "unknown_kind", Value: "v1", Value2: "v2"},
					{Name: "empty", Type: "password", Value: ""},
				},
			}},
		}},
		Users:  []file.User{{Username: "admin", PasswordHash: "$2a$hash"}},
		Clouds: []file.Cloud{{Name: "archive", Type: "s3", Key: "AKIA", Secret: "s3-secret"}, {Name: "none", Type: "s3"}},
		SNMPConfig: &file.SNMPConfig{
			Server:   &file.SNMPServer{Community: "public", V3AuthPassphrase: "a", V3PrivacyPassphrase: "p"},
			Monitors: []*file.SNMPMonitor{{Name: "nagios", Community: "public", V3AuthPassphrase: "a"}},
		},
		Authorities: []*file.Authority{{Name: "ca-interne", Type: "tls_authority", PublicIdentity: "CA-PEM"}},
		CryptoKeys:  []*file.CryptoKey{{Name: "aes-1", Type: "AES", Key: "base64"}},
		EmailConfig: &file.EmailConfig{Credentials: []*file.SMTPCredential{{EmailAddress: "noreply@example.org", Password: "smtp"}}},
	}

	redactSecrets(data)

	server := data.Locals[0]
	assert.Equal(t, "__PEM__$GWPKIDIR/servers/srv-https-443/cert.pem", server.Credentials[0].Value, "the name of the server made safe")
	assert.Equal(t, "__KEY__$GWPKIDIR/servers/srv-https-443/cert.key", server.Credentials[0].Value2)
	assert.Equal(t, "__PEM__$GWPKIDIR/servers/srv-https-443/trusted.pem", server.Credentials[1].Value)
	assert.Equal(t, "", server.Credentials[1].Value2, "nothing where there was nothing")
	assert.Equal(t, file.Certificate{Name: "old", PublicKey: "__PEM__$GWPKIDIR/servers/srv-https-443/old-public.pem",
		PrivateKey: "__KEY__$GWPKIDIR/servers/srv-https-443/old.key", Certificate: "__PEM__$GWPKIDIR/servers/srv-https-443/old.pem"},
		server.Certificates[0])

	account := server.Accounts[0]
	assert.Equal(t, api.NewNullable("__PWD__servers/srv-https-443/filiale:password"), account.Password)
	assert.Equal(t, "__PWD__servers/srv-https-443/filiale:password", account.PasswordHash,
		"the former field doubles the password credential: the same marker")
	assert.Equal(t, "__PWD__servers/srv-https-443/filiale:password", account.Credentials[0].Value)
	assert.Equal(t, "__PEM__$GWPKIDIR/servers/srv-https-443/filiale/ssh.pem", account.Credentials[1].Value)

	partner := data.Remotes[0]
	assert.Equal(t, "__PEM__$GWPKIDIR/partners/banque-demo/legacy.pem", partner.Credentials[0].Value)
	assert.Equal(t, "__KEY__$GWPKIDIR/partners/banque-demo/legacy.key", partner.Credentials[0].Value2)

	remote := partner.Accounts[0]
	assert.Equal(t, api.NewNullable("__PWD__partners/banque-demo/prod:password"), remote.Password)
	assert.Equal(t, "login-pre", remote.Credentials[0].Value, "the login of the pre-connection stays")
	assert.Equal(t, "__PWD__partners/banque-demo/prod:pre", remote.Credentials[0].Value2)
	assert.Equal(t, "__KEY__$GWPKIDIR/partners/banque-demo/prod/key.key", remote.Credentials[1].Value)
	assert.Equal(t, "__PWD__partners/banque-demo/prod:odd", remote.Credentials[2].Value, "an unknown kind is a secret")
	assert.Equal(t, "__PWD__partners/banque-demo/prod:odd-2", remote.Credentials[2].Value2)
	assert.Equal(t, "", remote.Credentials[3].Value, "an empty value stays empty")

	assert.Equal(t, "$2a$hash", data.Users[0].PasswordHash, "the users keep their hashes")
	assert.Equal(t, "__PWD__clouds/archive:secret", data.Clouds[0].Secret)
	assert.Equal(t, "AKIA", data.Clouds[0].Key, "the access key is not the secret")
	assert.Equal(t, "", data.Clouds[1].Secret)
	assert.Equal(t, "__PWD__snmp/server:community", data.SNMPConfig.Server.Community)
	assert.Equal(t, "__PWD__snmp/server:v3AuthPassphrase", data.SNMPConfig.Server.V3AuthPassphrase)
	assert.Equal(t, "__PWD__snmp/server:v3PrivacyPassphrase", data.SNMPConfig.Server.V3PrivacyPassphrase)
	assert.Equal(t, "__PWD__snmp/monitors/nagios:community", data.SNMPConfig.Monitors[0].Community)
	assert.Equal(t, "", data.SNMPConfig.Monitors[0].V3PrivacyPassphrase)
	assert.Equal(t, "__PEM__$GWPKIDIR/authorities/ca-interne.pem", data.Authorities[0].PublicIdentity)
	assert.Equal(t, "__KEY__$GWPKIDIR/keys/aes-1.key", data.CryptoKeys[0].Key)
	assert.Equal(t, "__PWD__smtp/noreply@example.org:password", data.EmailConfig.Credentials[0].Password)
}

// A marker is found in any key and any value of a file, as the import would
// read them, whatever the object and however the file writes it; a file
// without any is clean, a comment is not a value.
func TestCarriesMarkers(t *testing.T) {
	t.Parallel()

	for name, content := range map[string]string{
		"password.json":    `{"remotes": [{"name": "bank", "accounts": [{"login": "prod", "password": "__PWD__partners/bank/prod:password"}]}]}`,
		"hash.json":        `{"locals": [{"name": "sftp", "accounts": [{"login": "a", "passwordHash": "__PWD__servers/sftp/a:password"}]}]}`,
		"key.json":         `{"locals": [{"name": "sftp", "credentials": [{"type": "tls_certificate", "value2": "__KEY__$GWPKIDIR/servers/sftp/cert.key"}]}]}`,
		"authority.yaml":   "authorities:\n- name: ca\n  publicIdentity: __PEM__$GWPKIDIR/authorities/ca.pem\n",
		"option.yaml":      "clouds:\n- name: s3\n  options:\n    session_token: __PWD__clouds/s3:session_token\n",
		"unknown.yaml":     "notRead:\n  byTheImport: __PWD__x:y\n",
		"aKey.yaml":        "clouds:\n- options:\n    __PWD__x:y: value\n",
		"escaped.json":     `{"clouds": [{"name": "s3", "secret": "\u005f_PWD__clouds/s3:secret"}]}`,
		"twice.json":       `{"remotes": [{"name": "b", "accounts": [{"login": "p", "password": "__PWD__x:y"}]}], "remotes": []}`,
		"alias.yaml":       "secret: &s __PWD__clouds/s3:secret\nclouds:\n- name: s3\n  secret: *s\n",
		"binary.yaml":      "clouds:\n- name: s3\n  secret: !!binary X19QV0RfX2Nsb3Vkcy9zMzpzZWNyZXQ=\n",
		"folded.yaml":      "clouds:\n- name: s3\n  secret: >-\n    __PWD__clouds/s3:secret\n",
		"doubleQuoted.yml": "clouds:\n- name: s3\n  secret: \"\\x5f_PWD__clouds/s3:secret\"\n",
	} {
		marked, err := carriesMarkers([]byte(content), name)
		require.NoError(t, err, name)
		assert.True(t, marked, name)
	}

	for name, content := range map[string]string{
		"clean.json":      `{"rules": [{"name": "r", "path": "r"}]}`,
		"clean.yaml":      "rules:\n- name: r\n  path: r\n",
		"commented.yaml":  "# Markers (__PWD__, __KEY__, __PEM__) filled by the tooling.\nrules:\n- name: r # once a __PWD__ marker\n  path: r\n",
		"notANumber.yaml": "clients:\n- name: c\n  firstRetryIncrementFactor: .nan\n",
	} {
		marked, err := carriesMarkers([]byte(content), name)
		require.NoError(t, err, name)
		assert.False(t, marked, name)
	}

	_, err := carriesMarkers([]byte(`{"rules": [`), "cut.json")
	require.Error(t, err, "a file that cannot be read is not said clean")
}

// The aliases of a YAML file share their value: a few hundred bytes of
// nested ones stand for a value of any size, which the check does not unfold.
func TestCarriesMarkersNestedAliases(t *testing.T) {
	t.Parallel()

	const levels = 14

	nested := "a0: &a0 [x, x, x, x, x, x, x, x, x, x]\n"
	for i := 1; i <= levels; i++ {
		ref := fmt.Sprintf("*a%d", i-1)
		nested += fmt.Sprintf("a%d: &a%d [%s]\n", i, i, strings.TrimSuffix(strings.Repeat(ref+", ", 10), ", "))
	}

	start := time.Now()
	marked, err := carriesMarkers([]byte(nested), "nested.yaml")
	require.NoError(t, err)
	assert.False(t, marked)
	assert.Less(t, time.Since(start), 2*time.Second, "the check does not unfold the aliases")

	marked, err = carriesMarkers([]byte(nested+"last: __PWD__x:y\n"), "nested.yaml")
	require.NoError(t, err)
	assert.True(t, marked, "a marker next to the aliases is found")

	// A long text many aliases refer to is read once.
	shared := "text: &t " + strings.Repeat("x", 1<<20) + "\nlist: [" +
		strings.TrimSuffix(strings.Repeat("*t, ", 10000), ", ") + "]\n"

	start = time.Now()
	marked, err = carriesMarkers([]byte(shared), "shared.yaml")
	require.NoError(t, err)
	assert.False(t, marked)
	assert.Less(t, time.Since(start), time.Second, "the check reads a shared text once")

	// So are shared bytes.
	sharedBytes := "bytes: &b !!binary " + strings.Repeat("eHh4", 1<<18) + "\nlist: [" +
		strings.TrimSuffix(strings.Repeat("*b, ", 10000), ", ") + "]\n"

	start = time.Now()
	marked, err = carriesMarkers([]byte(sharedBytes), "shared.yaml")
	require.NoError(t, err)
	assert.False(t, marked)
	assert.Less(t, time.Since(start), time.Second, "the check reads shared bytes once")
}

// A secret also sits in the free tables of the file: the server password a
// R66 configuration still carries in its former form, the options of a cloud
// instance the screen never shows in clear. Each takes a marker; the former
// password takes the marker of the credential that holds the same secret.
func TestRedactSecretsOfFreeTables(t *testing.T) {
	t.Parallel()

	data := &file.Data{
		Locals: []file.LocalAgent{
			{
				Name: "srv-r66", Protocol: "r66",
				Configuration: map[string]any{"serverLogin": "srv", "serverPassword": "r66-clear"},
				Credentials:   []file.Credential{{Name: "srv-password", Type: "password", Value: "r66-clear"}},
			},
			{Name: "srv-r66-bare", Protocol: "r66", Configuration: map[string]any{"serverPassword": "r66-clear"}},
			{Name: "srv-r66-case", Protocol: "r66", Configuration: map[string]any{"ServerPassword": "r66-clear", "serverPassword": "enc"}},
			{Name: "srv-sftp", Protocol: "sftp", Configuration: map[string]any{"keyExchanges": []any{"curve25519-sha256"}}},
		},
		Remotes: []file.RemoteAgent{{
			Name: "part-r66", Protocol: "r66-tls",
			Configuration: map[string]any{"serverLogin": "part", "serverPassword": "$2a$hash"},
			Credentials:   []file.Credential{{Name: "password", Type: "password", Value: "$2a$hash"}},
		}},
		Clouds: []file.Cloud{{
			Name: "archive", Type: "s3", Key: "AKIA", Secret: "s3-secret",
			Options: map[string]string{"region": "eu-west-3", "session_token": "sts-token", "empty": ""},
		}},
	}

	redactSecrets(data)

	assert.Equal(t, "__PWD__servers/srv-r66:srv-password", data.Locals[0].Credentials[0].Value)
	assert.Equal(t, map[string]any{"serverLogin": "srv", "serverPassword": "__PWD__servers/srv-r66:srv-password"},
		data.Locals[0].Configuration, "one secret, one marker")
	assert.Equal(t, map[string]any{"serverPassword": "__PWD__servers/srv-r66-bare:password"}, data.Locals[1].Configuration)
	assert.Equal(t, map[string]any{
		"ServerPassword": "__PWD__servers/srv-r66-case:password", "serverPassword": "__PWD__servers/srv-r66-case:password",
	}, data.Locals[2].Configuration, "the key as the protocol reads it, whatever its case")
	assert.Equal(t, map[string]any{"keyExchanges": []any{"curve25519-sha256"}}, data.Locals[3].Configuration)

	assert.Equal(t, map[string]any{"serverLogin": "part", "serverPassword": "__PWD__partners/part-r66:password"},
		data.Remotes[0].Configuration)

	assert.Equal(t, map[string]string{
		"region": "eu-west-3", "session_token": "__PWD__clouds/archive:session_token", "empty": "",
	}, data.Clouds[0].Options)
}
