// Package chaintest makes a certificate chain of three levels for the tests
// of the screens: a root, an intermediate, a leaf for a server on 127.0.0.1
// with its private key, and a stranger (another self-signed certificate).
package chaintest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// PKI is the chain made for a test.
type PKI struct {
	Root, Inter, Leaf, Stranger *x509.Certificate
	// LeafKey is the private key of the leaf, PEM, as a form posts it.
	LeafKey string
}

// New makes the chain, valid for a day.
//
//nolint:mnd // a chain made for the tests, with its own serial numbers and addresses
func New(t *testing.T) *PKI {
	t.Helper()

	rootKey := newKey(t)
	root := sign(t, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Test root"},
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}, nil, rootKey, rootKey)

	interKey := newKey(t)
	inter := sign(t, &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "Test intermediate"},
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}, root, interKey, rootKey)

	leafKey := newKey(t)
	leaf := sign(t, &x509.Certificate{
		SerialNumber: big.NewInt(3), Subject: pkix.Name{CommonName: "127.0.0.1"},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)}, DNSNames: []string{"localhost"},
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}, inter, leafKey, interKey)

	strangerKey := newKey(t)
	stranger := sign(t, &x509.Certificate{
		SerialNumber: big.NewInt(4), Subject: pkix.Name{CommonName: "Stranger"},
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}, nil, strangerKey, strangerKey)

	der, err := x509.MarshalECPrivateKey(leafKey)
	require.NoError(t, err)

	return &PKI{
		Root: root, Inter: inter, Leaf: leaf, Stranger: stranger,
		LeafKey: string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})),
	}
}

// PEM writes the certificates, one block each, in the order given.
func PEM(certs ...*x509.Certificate) string {
	var b strings.Builder
	for _, c := range certs {
		b.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Raw}))
	}

	return b.String()
}

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	return key
}

// sign issues the template, signed by the parent (itself when nil), valid
// from an hour ago for a day.
//
//nolint:mnd // a validity made for the tests
func sign(t *testing.T, tmpl, parent *x509.Certificate, key, parentKey *ecdsa.PrivateKey) *x509.Certificate {
	t.Helper()

	tmpl.NotBefore, tmpl.NotAfter = time.Now().Add(-time.Hour), time.Now().Add(24*time.Hour)
	if parent == nil {
		parent = tmpl
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, &key.PublicKey, parentKey)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return cert
}
