// Package certchain assembles a certificate chain from the PEM blocks a
// form receives, in any order, and explains why the product refuses one.
//
// A certificate issued by a private PKI is signed by an intermediate
// authority, not by the root: to be verified, it must be given with its
// chain, the certificate first, then its intermediates up to the root, which
// the other side knows on its own. The product accepts and presents such a
// chain, provided the root is already registered among the authentication
// authorities; given alone, the certificate is refused with "signed by
// unknown authority", which does not say what is missing. The forms of the
// web interface take several files at once: this package puts them in the
// order TLS expects, drops a root given by mistake, refuses a certificate
// foreign to the chain, and names the issuer the gateway does not know.
package certchain

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/certinfo"
)

// Chain is what the form received, put in order: the certificate, its
// intermediates, and the root given with them, which TLS never presents.
type Chain struct {
	Certs []*x509.Certificate // the leaf first, then the intermediates
	Root  *x509.Certificate   // a self-signed root given with the chain, dropped from it
}

var (
	// ErrNoCertificate is returned when the text holds no certificate.
	ErrNoCertificate = errors.New("no certificate")
	// ErrNoLeaf is returned when every certificate signs another one: the
	// certificate itself is missing.
	ErrNoLeaf = errors.New("no end certificate")
)

// UnrelatedError names a certificate that belongs to no chain with the
// others.
type UnrelatedError struct{ Subject string }

func (e *UnrelatedError) Error() string {
	return fmt.Sprintf("the certificate %q does not belong to the chain", e.Subject)
}

// Parse reads every CERTIFICATE block of a PEM text, in the order given;
// other blocks are skipped.
func Parse(pemText string) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate

	rest := []byte(pemText)

	for {
		var block *pem.Block

		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse a certificate: %w", err)
		}

		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return nil, ErrNoCertificate
	}

	return certs, nil
}

// Assemble puts the certificates in the order TLS expects: the one that
// signs no other first, then each one's issuer. A self-signed certificate
// that ends the chain is the root: it is set apart, not kept in the chain. A
// single self-signed certificate is its own chain (the usual case of a
// certificate made for one server). A certificate that is neither in the
// chain nor its root is foreign: refused, named.
func Assemble(certs []*x509.Certificate) (*Chain, error) {
	if len(certs) == 0 {
		return nil, ErrNoCertificate
	}

	if len(certs) == 1 {
		return &Chain{Certs: certs}, nil
	}

	leaf := findLeaf(certs)
	if leaf == nil {
		return nil, ErrNoLeaf
	}

	chain := &Chain{Certs: []*x509.Certificate{leaf}}
	used := map[*x509.Certificate]bool{leaf: true}
	last := leaf

	for !selfSigned(last) {
		issuer := findIssuer(certs, last, used)
		if issuer == nil {
			break
		}

		used[issuer] = true
		last = issuer

		if selfSigned(issuer) {
			chain.Root = issuer

			break
		}

		chain.Certs = append(chain.Certs, issuer)
	}

	for _, c := range certs {
		if !used[c] {
			return nil, &UnrelatedError{Subject: certinfo.SubjectOf(c)}
		}
	}

	return chain, nil
}

// PEM writes the chain as the product stores it: the certificate, then its
// intermediates, one block each.
func (c *Chain) PEM() string {
	var b strings.Builder

	for _, cert := range c.Certs {
		b.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	}

	return b.String()
}

// Leaf is the certificate itself.
func (c *Chain) Leaf() *x509.Certificate { return c.Certs[0] }

// LeafPEM writes the certificate alone: what a trusted certificate keeps.
func (c *Chain) LeafPEM() string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Leaf().Raw}))
}

// Last is the last certificate of the chain: the one whose issuer the other
// side must know.
func (c *Chain) Last() *x509.Certificate { return c.Certs[len(c.Certs)-1] }

// Unknown explains "signed by unknown authority": the chain ends with a
// certificate whose issuer the gateway does not know. It returns the subject
// of that certificate and the name of its issuer, so that the form can say
// what to do: give the intermediate certificate of that issuer with the
// certificate, or register it among the authorities when it is the root.
func (c *Chain) Unknown() (subject, issuer string) {
	last := c.Last()

	return certinfo.SubjectOf(last), issuerName(last)
}

// findLeaf is the certificate that signs no other one of the set; nil when
// every certificate signs another (a loop, or the leaf is missing).
func findLeaf(certs []*x509.Certificate) *x509.Certificate {
	for _, candidate := range certs {
		signsAnother := false

		for _, other := range certs {
			if other != candidate && !selfSigned(other) && bytes.Equal(other.RawIssuer, candidate.RawSubject) &&
				other.CheckSignatureFrom(candidate) == nil {
				signsAnother = true

				break
			}
		}

		if !signsAnother {
			return candidate
		}
	}

	return nil
}

// findIssuer is the certificate of the set, not used yet, that signed the
// given one.
func findIssuer(certs []*x509.Certificate, of *x509.Certificate, used map[*x509.Certificate]bool) *x509.Certificate {
	for _, candidate := range certs {
		if used[candidate] || !bytes.Equal(of.RawIssuer, candidate.RawSubject) {
			continue
		}

		if of.CheckSignatureFrom(candidate) == nil {
			return candidate
		}
	}

	return nil
}

func selfSigned(c *x509.Certificate) bool {
	return bytes.Equal(c.RawIssuer, c.RawSubject) && c.CheckSignatureFrom(c) == nil
}

// issuerName names the issuer of a certificate as its subject would be
// named: its common name, else its whole name.
func issuerName(c *x509.Certificate) string {
	if c.Issuer.CommonName != "" {
		return c.Issuer.CommonName
	}

	return c.Issuer.String()
}
