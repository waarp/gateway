package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"unsafe"

	"github.com/goccy/go-yaml"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/clouds"
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

const (
	// The option of a R66 configuration that carries, in its former form,
	// the password of the server.
	serverPasswordOption = "serverPassword"
	// The name of the password in a marker, without a credential to name it.
	defaultPasswordName = "password"
)

// markers are the three, for the guard of the import.
//
//nolint:gochecknoglobals // reference data of the screen
var markers = []string{markerPassword, markerKey, markerPEM}

// carriesMarkers says whether a configuration file still carries a marker:
// such a file is not to be imported, a marker would become a password. The
// file is read in the format its name tells, as the import reads it, and
// nothing more: the import itself hashes the passwords it reads, a marker
// among them. Every key and every value is looked at, those the import does
// not know too; a comment is not one (the migration tooling writes comments
// that cite the markers).
func carriesMarkers(content []byte, name string) (bool, error) {
	var (
		found bool
		err   error
	)

	if slices.Contains(yamlTypes, filepath.Ext(name)) {
		found, err = yamlCarriesMarkers(content)
	} else {
		found, err = jsonCarriesMarkers(content)
	}

	if err != nil {
		return false, fmt.Errorf("failed to read the values of the file %q: %w", name, err)
	}

	return found, nil
}

// jsonCarriesMarkers reads the first JSON value of the file, the one the
// import reads, token by token: each key and each text as the file writes
// them, a key written twice both times (the import merges the two).
func jsonCarriesMarkers(content []byte) (bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	for depth := 0; ; {
		token, err := decoder.Token()
		if err != nil {
			return false, err //nolint:wrapcheck // wrapped by the caller, with the name of the file
		}

		switch token := token.(type) {
		case json.Delim:
			if token == '{' || token == '[' {
				depth++
			} else {
				depth--
			}
		case string:
			if hasMarker(token) {
				return true, nil
			}
		default:
			// A number, a boolean, a null: nothing that holds a marker.
		}

		if depth == 0 {
			return false, nil
		}
	}
}

// yamlCarriesMarkers decodes the file and walks its keys and its values.
func yamlCarriesMarkers(content []byte) (bool, error) {
	var values any
	if err := yaml.Unmarshal(content, &values); err != nil {
		return false, err //nolint:wrapcheck // wrapped by the caller, with the name of the file
	}

	return (&markerWalk{lists: map[part]struct{}{}, texts: map[text]struct{}{}}).marked(reflect.ValueOf(values)), nil
}

// markerWalk walks a decoded value for a marker, each part of it once: the
// aliases of a YAML file share their value, and a few nested ones, or many
// to a long text, would otherwise stand for a value of any size.
type markerWalk struct {
	lists map[part]struct{} // the maps and the lists already walked
	texts map[text]struct{} // the texts and the bytes already read
}

// part names a map or a list, text a text or bytes, by where they are and
// how long they are: never read through, only told apart.
type (
	part struct {
		at     uintptr
		length int
	}
	text struct {
		at     *byte
		length int
	}
)

func (w *markerWalk) marked(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		return !value.IsNil() && w.marked(value.Elem())
	case reflect.String:
		return w.markedText(value.String())
	case reflect.Map:
		if w.walked(value) {
			return false
		}

		for entry := value.MapRange(); entry.Next(); {
			if w.marked(entry.Key()) || w.marked(entry.Value()) {
				return true
			}
		}
	case reflect.Slice:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return w.markedBytes(value.Bytes())
		}

		if w.walked(value) {
			return false
		}

		for i := range value.Len() {
			if w.marked(value.Index(i)) {
				return true
			}
		}
	default:
		// A number, a boolean, a date: nothing that holds a marker.
	}

	return false
}

// walked notes a map or a list as walked, and says whether it was.
func (w *markerWalk) walked(value reflect.Value) bool {
	if value.IsNil() || value.Len() == 0 {
		return false
	}

	key := part{value.Pointer(), value.Len()}
	if _, ok := w.lists[key]; ok {
		return true
	}

	w.lists[key] = struct{}{}

	return false
}

// markedText reads a text once: the same text met again, by an alias, was
// read already.
func (w *markerWalk) markedText(value string) bool {
	return w.markedBytes(unsafe.Slice(unsafe.StringData(value), len(value)))
}

// markedBytes reads bytes once, as they are, without a copy.
func (w *markerWalk) markedBytes(value []byte) bool {
	if len(value) < len(markerPassword) {
		return false
	}

	key := text{unsafe.SliceData(value), len(value)}
	if _, ok := w.texts[key]; ok {
		return false
	}

	w.texts[key] = struct{}{}

	for _, m := range markers {
		if bytes.Contains(value, []byte(m)) {
			return true
		}
	}

	return false
}

func hasMarker(text string) bool {
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}

	return false
}

// redactSecrets replaces, in place, every secret and every certificate of the
// data by a marker: the passwords of the accounts, the credentials, the
// certificates and keys of the servers, the partners and their accounts, the
// secrets of the cloud instances and those of their options the screen never
// shows in clear, the server password a R66 configuration still carries in
// its former form, the community strings and passphrases of
// SNMP, the material of the cryptographic keys, the certificates of the
// authorities, the passwords of SMTP. The hashes of the users' passwords stay:
// they are not secrets in clear, and the import knows nothing else of a user;
// the hash of a local account goes with the password credential it doubles.
func redactSecrets(data *file.Data) {
	for i := range data.Locals {
		server := &data.Locals[i]
		owner := "servers/" + segment(server.Name)
		redactServerPassword(owner, server.Configuration, server.Credentials)
		redactCredentials(owner, server.Credentials)
		redactCertificates(owner, server.Certificates)

		for j := range server.Accounts {
			account := &server.Accounts[j]
			acc := owner + "/" + segment(account.Login)
			replace(&account.Password.Value, password(acc, defaultPasswordName))
			// The former field holds the hash the password credential holds.
			replace(&account.PasswordHash, passwordOf(acc, account.Credentials))
			redactCredentials(acc, account.Credentials)
			redactCertificates(acc, account.Certificates)
		}
	}

	for i := range data.Remotes {
		partner := &data.Remotes[i]
		owner := "partners/" + segment(partner.Name)
		redactServerPassword(owner, partner.Configuration, partner.Credentials)
		redactCredentials(owner, partner.Credentials)
		redactCertificates(owner, partner.Certificates)

		for j := range partner.Accounts {
			account := &partner.Accounts[j]
			acc := owner + "/" + segment(account.Login)
			replace(&account.Password.Value, password(acc, "password"))
			redactCredentials(acc, account.Credentials)
			redactCertificates(acc, account.Certificates)
		}
	}

	for i := range data.Clouds {
		cloud := &data.Clouds[i]
		owner := "clouds/" + segment(cloud.Name)
		replace(&cloud.Secret, password(owner, "secret"))

		for _, option := range clouds.SecretOptions(cloud.Type, cloud.Options) {
			if cloud.Options[option] != "" {
				cloud.Options[option] = password(owner, segment(option))
			}
		}
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

// redactServerPassword replaces the password a R66 configuration still
// carries in its former form. The same secret is held by the password
// credential of the same owner: both take the same marker, so that it is
// filled once.
func redactServerPassword(owner string, configuration map[string]any, creds []file.Credential) {
	// The protocol reads the option whatever its case: so does this.
	for option, value := range configuration {
		if text, isText := value.(string); isText && text != "" && strings.EqualFold(option, serverPasswordOption) {
			configuration[option] = passwordOf(owner, creds)
		}
	}
}

// passwordOf is the marker of the password of an owner: the one of its
// password credential, which its former fields double.
func passwordOf(owner string, creds []file.Credential) string {
	for i := range creds {
		if creds[i].Type == auth.Password {
			return password(owner, segment(creds[i].Name))
		}
	}

	return password(owner, defaultPasswordName)
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
