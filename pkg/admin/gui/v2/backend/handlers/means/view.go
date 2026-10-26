package means

import (
	"strings"
	"time"

	"code.waarp.fr/lib/gui"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/certinfo"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66/r66auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

// Families of means: the block of fields of the form and the shape of the
// summary.
const (
	familyPassword = "password" // one secret
	familyFile     = "file"     // one file: a certificate, a public key or a private key
	familyTwoFile  = "twofile"  // a certificate and its private key
	familyPair     = "pair"     // a login and a password: the PeSIT pre-connection
	familyNone     = "none"     // nothing to give: the kind itself is the information
)

// families are the kinds of means the pages handle, by the keys the model
// registers.
//
//nolint:gochecknoglobals // reference data of the pages
var families = map[string]string{
	auth.Password:                 familyPassword,
	auth.TLSCertificate:           familyTwoFile,
	auth.TLSTrustedCertificate:    familyFile,
	sftp.AuthSSHPrivateKey:        familyFile,
	sftp.AuthSSHPublicKey:         familyFile,
	pesit.PreConnectionAuth:       familyPair,
	r66auth.AuthLegacyCertificate: familyNone,
}

// FamilyOf is the family of a kind of means: the block of fields of its form.
func FamilyOf(typ string) string { return families[typ] }

// view is a means as the pages show it: never the secret, only what
// identifies it (the summary of its certificate or key, shared with the
// authorities screen).
type view struct {
	ID   int64
	Name string
	Type string
	certinfo.Info
	// Chain is every certificate of a chain, the certificate first; Info
	// sums up the first one.
	Chain []certinfo.Info
	// The PeSIT pre-connection: its login (not a secret) and whether it
	// holds a password (never the password).
	Login       string
	HasPassword bool
}

func viewOf(c *model.Credential) view {
	v := view{ID: c.ID, Name: c.Name, Type: c.Type}

	switch c.Type {
	case auth.TLSCertificate, auth.TLSTrustedCertificate:
		// One that cannot be read leaves nothing: the means is summarised
		// as "set".
		if chain := certinfo.Chain(c.Value); len(chain) > 0 {
			v.Info, v.Chain = chain[0], chain
		}
	case sftp.AuthSSHPrivateKey:
		if signer, err := ssh.ParsePrivateKey([]byte(c.Value)); err == nil {
			v.Info = certinfo.Key(signer.PublicKey())
		}
	case sftp.AuthSSHPublicKey:
		if key, err := sftp.ParseAuthorizedKey(c.Value); err == nil {
			v.Info = certinfo.Key(key)
			v.Public = strings.TrimSpace(c.Value)
		}
	case pesit.PreConnectionAuth:
		v.Login, v.HasPassword = c.Value, c.Value2 != ""
	}

	return v
}

func (v *view) isCert() bool {
	return v.Type == auth.TLSCertificate || v.Type == auth.TLSTrustedCertificate
}

// expiry is the validity state of a certificate (certinfo); nothing for the
// other kinds.
func (v *view) expiry(translate gui.Translator, now time.Time) gui.Status {
	if !v.isCert() {
		return gui.Status{}
	}

	// A chain is valid as long as every certificate of it is.
	nearest := certinfo.Nearest(v.Chain)

	return nearest.Expiry(translate, now)
}

// chainText says how many certificates a chain holds, after the summary;
// nothing for a certificate alone.
func (v *view) chainText(translate gui.Translator) string {
	if len(v.Chain) <= 1 {
		return ""
	}

	return " · " + translate("auth.content.chain", "n", len(v.Chain))
}

// summary is the non-secret content shown in the list.
func (v *view) summary(translate gui.Translator) string {
	switch {
	case v.Type == auth.Password:
		return translate("auth.content.secret")
	case v.Type == auth.TLSCertificate && v.Subject != "":
		return translate("auth.content.cert_key", "subject", v.Subject) + v.chainText(translate)
	case v.Type == auth.TLSTrustedCertificate && v.Subject != "":
		return v.Subject + v.chainText(translate)
	case v.Type == sftp.AuthSSHPrivateKey && v.KeyType != "":
		return translate("auth.content.private_key", "type", v.KeyType)
	case v.Type == sftp.AuthSSHPublicKey && v.KeyType != "":
		return translate("auth.content.key", "type", v.KeyType, "fingerprint", v.Fingerprint)
	case v.Type == pesit.PreConnectionAuth:
		return v.preConnection(translate)
	default:
		return translate("auth.content.set")
	}
}

// preConnection sums up a PeSIT pre-connection: its login, whether it holds
// a password; both are optional.
func (v *view) preConnection(translate gui.Translator) string {
	login := translate("auth.content.pre_no_login")
	if v.Login != "" {
		login = translate("auth.content.pre_login", "login", v.Login)
	}

	password := translate("auth.content.pre_no_password")
	if v.HasPassword {
		password = translate("auth.content.pre_password")
	}

	return login + " · " + password
}

// Summary is the non-secret content of a means, as the lists show it.
func Summary(translate gui.Translator, c *model.Credential) string {
	v := viewOf(c)

	return v.summary(translate)
}
