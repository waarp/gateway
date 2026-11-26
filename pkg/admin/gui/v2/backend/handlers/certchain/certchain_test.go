package certchain

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pki is a chain of three levels made for the tests: a root, an
// intermediate, a leaf for a server, and a stranger (another self-signed
// certificate).
type pki struct {
	root, inter, leaf, stranger *x509.Certificate
}

func newPKI(t *testing.T) *pki {
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
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)}, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, inter, leafKey, interKey)

	strangerKey := newKey(t)
	stranger := sign(t, &x509.Certificate{
		SerialNumber: big.NewInt(4), Subject: pkix.Name{CommonName: "Stranger"},
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}, nil, strangerKey, strangerKey)

	return &pki{root: root, inter: inter, leaf: leaf, stranger: stranger}
}

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	return key
}

// sign issues the template, signed by the parent (itself when nil).
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

func pemOf(certs ...*x509.Certificate) string {
	var b strings.Builder
	for _, c := range certs {
		_ = pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})
	}

	return b.String()
}

func names(certs []*x509.Certificate) []string {
	out := make([]string, 0, len(certs))
	for _, c := range certs {
		out = append(out, c.Subject.CommonName)
	}

	return out
}

func TestParse(t *testing.T) {
	t.Parallel()

	p := newPKI(t)

	certs, err := Parse(pemOf(p.leaf, p.inter))
	require.NoError(t, err)
	assert.Equal(t, []string{"127.0.0.1", "Test intermediate"}, names(certs))

	certs, err = Parse("-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n" + pemOf(p.leaf))
	require.NoError(t, err, "other blocks are skipped")
	assert.Len(t, certs, 1)

	_, err = Parse("not a pem")
	require.ErrorIs(t, err, ErrNoCertificate)

	_, err = Parse("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n")
	require.Error(t, err, "a block that is not a certificate")
}

func TestAssemble(t *testing.T) {
	t.Parallel()

	p := newPKI(t)

	t.Run("the order given does not matter", func(t *testing.T) {
		t.Parallel()

		for _, given := range [][]*x509.Certificate{
			{p.leaf, p.inter}, {p.inter, p.leaf},
		} {
			chain, err := Assemble(given)
			require.NoError(t, err)
			assert.Equal(t, []string{"127.0.0.1", "Test intermediate"}, names(chain.Certs))
			assert.Nil(t, chain.Root)
		}
	})

	t.Run("a root given with the chain is set apart", func(t *testing.T) {
		t.Parallel()

		chain, err := Assemble([]*x509.Certificate{p.root, p.leaf, p.inter})
		require.NoError(t, err)
		assert.Equal(t, []string{"127.0.0.1", "Test intermediate"}, names(chain.Certs))
		require.NotNil(t, chain.Root)
		assert.Equal(t, "Test root", chain.Root.Subject.CommonName)
		assert.Equal(t, pemOf(p.leaf, p.inter), chain.PEM(), "the root is not written")
		assert.Equal(t, pemOf(p.leaf), chain.LeafPEM())
	})

	t.Run("a single self-signed certificate is its own chain", func(t *testing.T) {
		t.Parallel()

		chain, err := Assemble([]*x509.Certificate{p.stranger})
		require.NoError(t, err)
		assert.Equal(t, []string{"Stranger"}, names(chain.Certs))
		assert.Nil(t, chain.Root)
	})

	t.Run("a leaf alone keeps its issuer's name for the explanation", func(t *testing.T) {
		t.Parallel()

		chain, err := Assemble([]*x509.Certificate{p.leaf})
		require.NoError(t, err)

		subject, issuer := chain.Unknown()
		assert.Equal(t, "127.0.0.1", subject)
		assert.Equal(t, "Test intermediate", issuer)

		chain, err = Assemble([]*x509.Certificate{p.inter, p.leaf})
		require.NoError(t, err)

		subject, issuer = chain.Unknown()
		assert.Equal(t, "Test intermediate", subject, "the chain ends with the intermediate")
		assert.Equal(t, "Test root", issuer, "whose issuer is the root")
	})

	t.Run("a foreign certificate is refused and named", func(t *testing.T) {
		t.Parallel()

		_, err := Assemble([]*x509.Certificate{p.leaf, p.inter, p.stranger})

		var unrelated *UnrelatedError

		require.ErrorAs(t, err, &unrelated)
		assert.Equal(t, "Stranger", unrelated.Subject)
	})

	t.Run("two certificates that sign each other: no leaf", func(t *testing.T) {
		t.Parallel()

		_, err := Assemble([]*x509.Certificate{p.root, p.inter})
		require.NoError(t, err, "the intermediate is the leaf of that set, the root its root")

		_, err = Assemble(nil)
		require.True(t, errors.Is(err, ErrNoCertificate))
	})
}
