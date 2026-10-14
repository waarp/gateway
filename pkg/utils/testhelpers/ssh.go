package testhelpers

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// SSHUserCert is a test SSH user certificate, along with the private key it
// was issued for and the authority which signed it.
type SSHUserCert struct {
	// Cert is the signed user certificate.
	Cert *ssh.Certificate
	// Signer can be used by a client to authenticate with the certificate.
	Signer ssh.Signer
	// CAPublicKey is the certificate authority's public key in the
	// "authorized_keys" format (suitable for an authority's PublicIdentity).
	CAPublicKey string
}

// MakeSSHUserCert generates a new SSH certificate authority and a new user key
// pair, and returns a certificate for the user key signed by the authority.
// The certificate is valid for the given principals (all of them if none are
// given) and expires one hour after its creation.
func MakeSSHUserCert(tb testing.TB, principals ...string) *SSHUserCert {
	tb.Helper()

	_, caPrivKey, caErr := ed25519.GenerateKey(rand.Reader)
	require.NoError(tb, caErr)

	caSigner, caSignErr := ssh.NewSignerFromKey(caPrivKey)
	require.NoError(tb, caSignErr)

	_, userPrivKey, userErr := ed25519.GenerateKey(rand.Reader)
	require.NoError(tb, userErr)

	userSigner, userSignErr := ssh.NewSignerFromKey(userPrivKey)
	require.NoError(tb, userSignErr)

	//nolint:gosec //no need for a secure serial number in tests
	cert := &ssh.Certificate{
		Key:             userSigner.PublicKey(),
		Serial:          1,
		CertType:        ssh.UserCert,
		KeyId:           "test-user-cert",
		ValidPrincipals: principals,
		ValidAfter:      uint64(time.Now().Add(-time.Minute).Unix()),
		ValidBefore:     uint64(time.Now().Add(time.Hour).Unix()),
	}
	require.NoError(tb, cert.SignCert(rand.Reader, caSigner))

	certSigner, certSignErr := ssh.NewCertSigner(cert, userSigner)
	require.NoError(tb, certSignErr)

	return &SSHUserCert{
		Cert:        cert,
		Signer:      certSigner,
		CAPublicKey: string(ssh.MarshalAuthorizedKey(caSigner.PublicKey())),
	}
}

const RSAPk = `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAlwAAAAdzc2gtcn
NhAAAAAwEAAQAAAIEAwg0KXLz4S0UF4QRgDp+WTWzCewS+sLUtZ0aLQSAN2HWdCssm/Lu0
k2gLxH3XJjYJd400tc/CMcWW+IUVrhMhfLNR1pjVixqGVPSxXjGpNgfwzxyZ5QC0o6aNbB
TZl1U8v/qi7+xFbEX7gNTMnAgQy79BzCoqH+CIXUHjHl5CEi8AAAIItGeBDLRngQwAAAAH
c3NoLXJzYQAAAIEAwg0KXLz4S0UF4QRgDp+WTWzCewS+sLUtZ0aLQSAN2HWdCssm/Lu0k2
gLxH3XJjYJd400tc/CMcWW+IUVrhMhfLNR1pjVixqGVPSxXjGpNgfwzxyZ5QC0o6aNbBTZ
l1U8v/qi7+xFbEX7gNTMnAgQy79BzCoqH+CIXUHjHl5CEi8AAAADAQABAAAAgHafwCjnAC
YLQMfIg+wMlLYp+U14nhWp++J5VmFudehQJbtWazPujznZTfBFOUPLnsftkz5djgodDuhH
evRaD4p6FvjZ8Rpr0zH5Ieg8hSRpjJZezGuOsAEswpLFau5bKkLj5LkQrFNv1TndcrUadi
Ht6lJPopALaJ5918BRSeqZAAAAQHT/b5HbiwGx/7uTisLQ19YK9GwqH9kW1a6kvzyG2Yct
EoGd6VmkZ5cxr0AIY2XyoxhfXFPyk0bqilC+K0e316AAAABBAN/OkGhlhlO1zvkiDIsdzh
39YrjB7SJD0H6qViru04IndUnwLet2d9B7/tAnoqMEkSWGLZVMvE/cl2i0sG7genMAAABB
AN32v0DkVrcZBl39vbbk2+syaNWxx5rjnVDSlVkpTf/5lAhtnNLwX6LoOMDTCsfY8nodJ2
wZCoCb3M+TY690blUAAAAMcGFvbG9AcXVhc2FyAQIDBAUGBw==
-----END OPENSSH PRIVATE KEY-----`

const SSHPbk = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDCDQpcvPhLRQXhBGAOn5ZNbMJ" +
	"7BL6wtS1nRotBIA3YdZ0Kyyb8u7STaAvEfdcmNgl3jTS1z8IxxZb4hRWuEyF8s1HWmNWLGoZU9" +
	"LFeMak2B/DPHJnlALSjpo1sFNmXVTy/+qLv7EVsRfuA1MycCBDLv0HMKiof4IhdQeMeXkISLw=="
