package means

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"
	"golang.org/x/crypto/ssh"

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

	// The validity state of a certificate: the triangle of the kit for one
	// that expires soon.
	stateOK       = "ok"
	stateKO       = "ko"
	stateWarn     = "warn"
	expiryWarning = 30 * 24 * time.Hour
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
// identifies it.
type view struct {
	ID          int64
	Name        string
	Type        string
	Subject     string
	Issuer      string
	SelfSigned  bool
	Fingerprint string
	KeyType     string
	NotBefore   time.Time
	NotAfter    time.Time
	Public      string // the public part, shareable: a certificate PEM, a public key line
	// The PeSIT pre-connection: its login (not a secret) and whether it
	// holds a password (never the password).
	Login       string
	HasPassword bool
}

func viewOf(c *model.Credential) view {
	v := view{ID: c.ID, Name: c.Name, Type: c.Type}

	switch c.Type {
	case auth.TLSCertificate, auth.TLSTrustedCertificate:
		v.readCertificate(c.Value)
	case sftp.AuthSSHPrivateKey:
		if signer, err := ssh.ParsePrivateKey([]byte(c.Value)); err == nil {
			v.readKey(signer.PublicKey())
		}
	case sftp.AuthSSHPublicKey:
		if key, err := sftp.ParseAuthorizedKey(c.Value); err == nil {
			v.readKey(key)
			v.Public = strings.TrimSpace(c.Value)
		}
	case pesit.PreConnectionAuth:
		v.Login, v.HasPassword = c.Value, c.Value2 != ""
	}

	return v
}

// readCertificate keeps the subject, the issuer, the validity and the
// SHA-256 fingerprint of a certificate; one that cannot be read leaves
// nothing, and the means is summarised as "set".
func (v *view) readCertificate(pemText string) {
	cert, err := auth.ParseCertPEM(pemText)
	if err != nil {
		return
	}

	sum := sha256.Sum256(cert.Raw)
	v.Subject = subjectOf(cert)
	v.Issuer = cert.Issuer.CommonName
	v.SelfSigned = bytes.Equal(cert.RawIssuer, cert.RawSubject)
	v.NotBefore, v.NotAfter = cert.NotBefore, cert.NotAfter
	v.Fingerprint = colons(sum[:])
	v.Public = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
}

// subjectOf names a certificate: its common name, or the first name it is
// issued for, or its whole subject.
func subjectOf(cert *x509.Certificate) string {
	switch {
	case cert.Subject.CommonName != "":
		return cert.Subject.CommonName
	case len(cert.DNSNames) > 0:
		return cert.DNSNames[0]
	case len(cert.IPAddresses) > 0:
		return cert.IPAddresses[0].String()
	default:
		return cert.Subject.String()
	}
}

// colons writes a fingerprint as the certificate tools do: upper-case hex
// pairs separated by colons.
func colons(sum []byte) string {
	const pair = 2 // hex digits per byte

	h := strings.ToUpper(hex.EncodeToString(sum))
	pairs := make([]string, 0, len(h)/pair)

	for i := 0; i+1 < len(h); i += pair {
		pairs = append(pairs, h[i:i+pair])
	}

	return strings.Join(pairs, ":")
}

func (v *view) readKey(key ssh.PublicKey) {
	v.KeyType = key.Type()
	v.Fingerprint = ssh.FingerprintSHA256(key)
}

func (v *view) isCert() bool {
	return v.Type == auth.TLSCertificate || v.Type == auth.TLSTrustedCertificate
}

// expiry is the validity state of a certificate: expired, expiring within
// thirty days, or valid; nothing for the other kinds.
func (v *view) expiry(translate gui.Translator, now time.Time) gui.Status {
	if !v.isCert() || v.NotAfter.IsZero() {
		return gui.Status{}
	}

	date := v.NotAfter.Format(translate("auth.date"))

	switch {
	case v.NotAfter.Before(now):
		return gui.Status{State: stateKO, Label: translate("auth.content.expired", "date", date)}
	case v.NotAfter.Before(now.Add(expiryWarning)):
		return gui.Status{State: stateWarn, Label: translate("auth.content.expires", "date", date)}
	default:
		return gui.Status{State: stateOK, Label: translate("auth.content.valid", "date", date)}
	}
}

// summary is the non-secret content shown in the list.
func (v *view) summary(translate gui.Translator) string {
	switch {
	case v.Type == auth.Password:
		return translate("auth.content.secret")
	case v.Type == auth.TLSCertificate && v.Subject != "":
		return translate("auth.content.cert_key", "subject", v.Subject)
	case v.Type == auth.TLSTrustedCertificate && v.Subject != "":
		return v.Subject
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
