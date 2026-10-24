package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
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
				Login: "filiale", Password: "clear-password", PasswordHash: "$2a$hash",
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
				Login: "prod", Password: "p",
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
	assert.Equal(t, "__PWD__servers/srv-https-443/filiale:password", account.Password)
	assert.Equal(t, "$2a$hash", account.PasswordHash, "a hash is not a secret in clear")
	assert.Equal(t, "__PWD__servers/srv-https-443/filiale:password", account.Credentials[0].Value)
	assert.Equal(t, "__PEM__$GWPKIDIR/servers/srv-https-443/filiale/ssh.pem", account.Credentials[1].Value)

	partner := data.Remotes[0]
	assert.Equal(t, "__PEM__$GWPKIDIR/partners/banque-demo/legacy.pem", partner.Credentials[0].Value)
	assert.Equal(t, "__KEY__$GWPKIDIR/partners/banque-demo/legacy.key", partner.Credentials[0].Value2)

	remote := partner.Accounts[0]
	assert.Equal(t, "__PWD__partners/banque-demo/prod:password", remote.Password)
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

func TestHasMarkers(t *testing.T) {
	t.Parallel()

	assert.False(t, hasMarkers([]byte(`{"rules": [{"name": "r", "path": "r"}]}`)))
	assert.True(t, hasMarkers([]byte(`password: __PWD__partners/bank/prod:password`)))
	assert.True(t, hasMarkers([]byte(`value2: __KEY__$GWPKIDIR/servers/sftp/cert.key`)))
	assert.True(t, hasMarkers([]byte(`publicIdentity: __PEM__$GWPKIDIR/authorities/ca.pem`)))
}
