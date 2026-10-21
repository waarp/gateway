// Package certinfo sums up what a certificate or a public key says, for the
// pages that show one without any secret part: the authentication means of
// the servers, partners and accounts, and the authentication authorities.
package certinfo

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

	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
)

// The validity state of a certificate, as the statuses of the kit show it: a
// round for a valid one, the triangle for one that expires within
// ExpiryWarning, the diamond for an expired one.
const (
	StateOK       = "ok"
	StateWarn     = "warn"
	StateKO       = "ko"
	ExpiryWarning = 30 * 24 * time.Hour
)

// Info is what a certificate or a public key says: the subject, the issuer,
// the validity and the SHA-256 fingerprint of a certificate; the type and the
// fingerprint of a key; the public part in full, shareable.
type Info struct {
	Subject     string
	Issuer      string
	SelfSigned  bool
	Fingerprint string
	KeyType     string
	NotBefore   time.Time
	NotAfter    time.Time
	Public      string // a certificate PEM, a public key line
}

// Certificate reads the first certificate of a PEM text; false when it cannot
// be read.
func Certificate(pemText string) (Info, bool) {
	cert, err := auth.ParseCertPEM(pemText)
	if err != nil {
		return Info{}, false
	}

	sum := sha256.Sum256(cert.Raw)

	return Info{
		Subject:     SubjectOf(cert),
		Issuer:      cert.Issuer.CommonName,
		SelfSigned:  bytes.Equal(cert.RawIssuer, cert.RawSubject),
		Fingerprint: colons(sum[:]),
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
		Public:      string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})),
	}, true
}

// Key reads the type and the fingerprint of a public key.
func Key(key ssh.PublicKey) Info {
	return Info{KeyType: key.Type(), Fingerprint: ssh.FingerprintSHA256(key)}
}

// SubjectOf names a certificate: its common name, or the first name it is
// issued for, or its whole subject.
func SubjectOf(cert *x509.Certificate) string {
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

// Expiry is the validity state of a certificate: expired, expiring within
// ExpiryWarning, or valid, with its date; nothing when it has no validity (a
// key, an unreadable certificate).
func (i *Info) Expiry(translate gui.Translator, now time.Time) gui.Status {
	if i.NotAfter.IsZero() {
		return gui.Status{}
	}

	date := i.NotAfter.Format(translate("auth.date"))

	switch {
	case i.NotAfter.Before(now):
		return gui.Status{State: StateKO, Label: translate("auth.content.expired", "date", date)}
	case i.NotAfter.Before(now.Add(ExpiryWarning)):
		return gui.Status{State: StateWarn, Label: translate("auth.content.expires", "date", date)}
	default:
		return gui.Status{State: StateOK, Label: translate("auth.content.valid", "date", date)}
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
