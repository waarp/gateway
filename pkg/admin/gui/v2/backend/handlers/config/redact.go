package config

import (
	"bytes"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

// The markers of the migration tooling (fill_passwords.py, fill_certs.py), put
// in place of the secrets and the certificates of an export: a password or a
// secret, a private key, a certificate or a public key. Each carries where the
// value belongs, so that the tooling finds it: the object and its field for a
// password ("__PWD__partners/bank/prod:password"), a file under $GWPKIDIR for
// a key or a certificate ("__KEY__$GWPKIDIR/servers/sftp/cert.key").
const (
	markerPassword = "__PWD__"
	markerKey      = "__KEY__"
	markerPEM      = "__PEM__"
	pkiDir         = "$GWPKIDIR/"
)

// markers are the three, for the guard of the import.
//
//nolint:gochecknoglobals // reference data of the screen
var markers = [][]byte{[]byte(markerPassword), []byte(markerKey), []byte(markerPEM)}

// hasMarkers says whether a file still carries a marker: such a file is not
// to be imported, a marker would become a password.
func hasMarkers(content []byte) bool {
	for _, m := range markers {
		if bytes.Contains(content, m) {
			return true
		}
	}

	return false
}

// redactSecrets replaces, in place, every secret and every certificate of the
// data by a marker: the passwords of the accounts, the credentials, the
// certificates and keys of the servers, the partners and their accounts, the
// secrets of the cloud instances, the community strings and passphrases of
// SNMP, the material of the cryptographic keys, the certificates of the
// authorities, the passwords of SMTP. The hashes of the users' passwords stay:
// they are not secrets in clear, and the import knows nothing else of a user.
func redactSecrets(data *file.Data) {
	for i := range data.Locals {
		server := &data.Locals[i]
		owner := "servers/" + segment(server.Name)
		redactCredentials(owner, server.Credentials)
		redactCertificates(owner, server.Certificates)

		for j := range server.Accounts {
			account := &server.Accounts[j]
			acc := owner + "/" + segment(account.Login)
			replace(&account.Password, password(acc, "password"))
			redactCredentials(acc, account.Credentials)
			redactCertificates(acc, account.Certificates)
		}
	}

	for i := range data.Remotes {
		partner := &data.Remotes[i]
		owner := "partners/" + segment(partner.Name)
		redactCredentials(owner, partner.Credentials)
		redactCertificates(owner, partner.Certificates)

		for j := range partner.Accounts {
			account := &partner.Accounts[j]
			acc := owner + "/" + segment(account.Login)
			replace(&account.Password, password(acc, "password"))
			redactCredentials(acc, account.Credentials)
			redactCertificates(acc, account.Certificates)
		}
	}

	for i := range data.Clouds {
		replace(&data.Clouds[i].Secret, password("clouds/"+segment(data.Clouds[i].Name), "secret"))
	}

	if snmp := data.SNMPConfig; snmp != nil {
		if server := snmp.Server; server != nil {
			replace(&server.Community, password("snmp/server", "community"))
			replace(&server.V3AuthPassphrase, password("snmp/server", "v3AuthPassphrase"))
			replace(&server.V3PrivacyPassphrase, password("snmp/server", "v3PrivacyPassphrase"))
		}

		for _, monitor := range snmp.Monitors {
			owner := "snmp/monitors/" + segment(monitor.Name)
			replace(&monitor.Community, password(owner, "community"))
			replace(&monitor.V3AuthPassphrase, password(owner, "v3AuthPassphrase"))
			replace(&monitor.V3PrivacyPassphrase, password(owner, "v3PrivacyPassphrase"))
		}
	}

	for _, authority := range data.Authorities {
		replace(&authority.PublicIdentity, certificate("authorities/"+segment(authority.Name)))
	}

	for _, key := range data.CryptoKeys {
		replace(&key.Key, privateKey("keys/"+segment(key.Name)))
	}

	if email := data.EmailConfig; email != nil {
		for _, cred := range email.Credentials {
			replace(&cred.Password, password("smtp/"+segment(cred.EmailAddress), "password"))
		}
	}
}

// redactCredentials replaces the secret parts of the credentials of an
// owner: what each kind of credential holds decides the marker.
func redactCredentials(owner string, creds []file.Credential) {
	for i := range creds {
		cred := &creds[i]
		base := owner + "/" + segment(cred.Name)

		switch cred.Type {
		case auth.TLSCertificate, r66.AuthLegacyCertificate:
			replace(&cred.Value, certificate(base))
			replace(&cred.Value2, privateKey(base))
		case auth.TLSTrustedCertificate, sftp.AuthSSHPublicKey:
			replace(&cred.Value, certificate(base))
		case sftp.AuthSSHPrivateKey:
			replace(&cred.Value, privateKey(base))
		case pesit.PreConnectionAuth:
			// The login stays, the password goes.
			replace(&cred.Value2, password(owner, segment(cred.Name)))
		default:
			// A password, or a kind this screen does not know: a secret.
			replace(&cred.Value, password(owner, segment(cred.Name)))
			replace(&cred.Value2, password(owner, segment(cred.Name)+"-2"))
		}
	}
}

// redactCertificates replaces the certificates of the former form of the
// file, which the export still writes next to the credentials for the
// importers of older versions.
//
//nolint:staticcheck // the export still writes this deprecated form
func redactCertificates(owner string, certs []file.Certificate) {
	for i := range certs {
		cert := &certs[i]
		base := owner + "/" + segment(cert.Name)
		replace(&cert.Certificate, certificate(base))
		replace(&cert.PublicKey, certificate(base+"-public"))
		replace(&cert.PrivateKey, privateKey(base))
	}
}

// replace puts the marker in place of a value, when there is one.
func replace(value *string, marker string) {
	if *value != "" {
		*value = marker
	}
}

func password(owner, field string) string { return markerPassword + owner + ":" + field }
func privateKey(path string) string       { return markerKey + pkiDir + path + ".key" }
func certificate(path string) string      { return markerPEM + pkiDir + path + ".pem" }

// segment makes a name fit in a marker: no path separator, no colon, which
// the tooling reads as its own.
func segment(name string) string {
	return strings.Trim(strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(name), "-")
}
