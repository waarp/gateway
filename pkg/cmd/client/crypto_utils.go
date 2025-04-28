package wg

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec //sha1 is only used for checksums
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gookit/color"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:varnamelen,funlen //formatter name is kept short for readability
func displayTLSInfo(w io.Writer, style *style, name, content string) error {
	certs, err := utils.ParsePEMCertChain(content)
	if err != nil || len(certs) == 0 {
		return fmt.Errorf("could not parse certificate: %w", err)
	}

	cert := certs[0]
	subStyle := nextStyle(style)
	subSubStyle := nextStyle(subStyle)

	style.printf(w, "Certificate %q:", name)

	subStyle.printf(w, "Subject")
	subSubStyle.printL(w, "Common Name", cert.Subject.CommonName)
	subSubStyle.option(w, "Organization", join(cert.Subject.Organization))
	subSubStyle.option(w, "Country", join(cert.Subject.Country))

	subStyle.printf(w, "Issuer")
	subSubStyle.printL(w, "Common Name", cert.Issuer.CommonName)
	subSubStyle.option(w, "Organization", join(cert.Issuer.Organization))
	subSubStyle.option(w, "Country", join(cert.Issuer.Country))

	subStyle.printf(w, "Validity")
	subSubStyle.printL(w, "Not Before", cert.NotBefore.Format(time.RFC1123))
	subSubStyle.printL(w, "Not After", cert.NotAfter.Format(time.RFC1123))

	subStyle.printf(w, "Subject Alt Names")
	subSubStyle.option(w, "DNS Names", join(cert.DNSNames))
	subSubStyle.option(w, "IP Addresses", joinStringers(cert.IPAddresses))
	subSubStyle.option(w, "Email Addresses", join(cert.EmailAddresses))

	subStyle.printf(w, "Public Key Info")
	subSubStyle.printL(w, "Algorithm", cert.PublicKeyAlgorithm.String())
	subSubStyle.printL(w, "Public Value", marshalPublicKey(cert.PublicKey))

	subStyle.printf(w, "Fingerprints")
	subSubStyle.printL(w, "SHA-256", sha256Sum(subSubStyle, cert.Raw))
	//nolint:gosec //sha1 is only used for checksums, no need to be cryptographically secure
	subSubStyle.printL(w, "SHA-1", fmt.Sprintf("% X", sha1.Sum(cert.Raw)))

	subStyle.printf(w, "Signature")
	subSubStyle.printL(w, "Algorithm", cert.SignatureAlgorithm.String())
	subSubStyle.printL(w, "Public Value", shortHex(cert.Signature))

	subStyle.printL(w, "Serial Number", shortHex(cert.SerialNumber.Bytes()))
	subStyle.printL(w, "Key Usages", strings.Join(utils.KeyUsageToStrings(cert.KeyUsage), ", "))
	subStyle.printL(w, "Extended Key Usages", strings.Join(utils.ExtKeyUsagesToStrings(cert.ExtKeyUsage), ", "))

	return nil
}

func marshalPublicKey(key any) string {
	var bytes []byte

	switch k := key.(type) {
	case *rsa.PublicKey:
		bytes = x509.MarshalPKCS1PublicKey(k)
	case *ecdsa.PublicKey:
		//nolint:staticcheck //keep for retro-compatibility
		bytes = elliptic.Marshal(k, k.X, k.Y)
	case *ecdh.PublicKey:
		bytes = k.Bytes()
	case *ed25519.PublicKey:
		bytes = *k
	default:
		return color.Red.Sprintf("<unknown public key type>")
	}

	return shortHex(bytes)
}

func sha256Sum(style *style, b []byte) string {
	sndLinePrefix := strings.Repeat(" ", utf8.RuneCountInString(style.bulletPrefix)+
		utf8.RuneCountInString("SHA-256: "))
	sum := sha256.Sum256(b)

	return fmt.Sprintf("% X\n%s% X", sum[:16], sndLinePrefix, sum[16:])
}

func shortHex(bytes []byte) string {
	const maxLen = 20
	if len(bytes) < maxLen {
		return fmt.Sprintf("% X", bytes)
	}

	return fmt.Sprintf("% X... (%d bytes total)", bytes[:maxLen-7], len(bytes))
}

func displaySSHKeyInfo(w io.Writer, style *style, name, content string) error {
	key, err := utils.ParseSSHAuthorizedKey(content)
	if err != nil || key == nil {
		return fmt.Errorf("could not parse SSH public key: %w", err)
	}

	subStyle := nextStyle(style)

	style.printf(w, "SSH Public Key %q:", name)
	subStyle.printL(w, "Type", key.Type())
	subStyle.printL(w, "SHA-256 Fingerprint", ssh.FingerprintSHA256(key))
	subStyle.printL(w, "MD5 Fingerprint", ssh.FingerprintLegacyMD5(key))

	return nil
}

func displayPrivateKeyInfo(w io.Writer, style *style, name, content string) error {
	pk, err := ssh.ParsePrivateKey([]byte(content))
	if err != nil {
		return fmt.Errorf("could not parse private key: %w", err)
	}

	subStyle := nextStyle(style)
	pubkey := pk.PublicKey()

	style.printf(w, "Private Key %q:", name)
	subStyle.printL(w, "Type", pubkey.Type())
	subStyle.printL(w, "SHA-256 Fingerprint", ssh.FingerprintSHA256(pubkey))
	subStyle.printL(w, "MD5 Fingerprint", ssh.FingerprintLegacyMD5(pubkey))

	return nil
}
