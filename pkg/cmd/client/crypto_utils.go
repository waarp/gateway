package wg

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec //sha1 is only used for checksums
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:varnamelen,funlen //formatter name is kept short for readability
func displayTLSInfo(f *Formatter, name, content string) {
	f.Title("Certificate %q", name)
	f.Indent()

	defer f.UnIndent()

	certs, err := utils.ParsePEMCertChain(content)
	if err != nil || len(certs) == 0 {
		f.Error("Content", "<could not parse certificate>")

		return
	}

	cert := certs[0]

	f.Title("Subject")
	f.Indent()
	f.Value("Common Name", cert.Subject.CommonName)
	f.ValueCond("Organization", strings.Join(cert.Subject.Organization, ", "))
	f.ValueCond("Country", strings.Join(cert.Subject.Country, ", "))
	f.UnIndent()

	f.Title("Issuer")
	f.Indent()
	f.Value("Common Name", cert.Issuer.CommonName)
	f.ValueCond("Organization", strings.Join(cert.Issuer.Organization, ", "))
	f.ValueCond("Country", strings.Join(cert.Issuer.Country, ", "))
	f.UnIndent()

	f.Title("Validity")
	f.Indent()
	f.Value("Not Before", cert.NotBefore.Format(time.RFC1123))
	f.Value("Not After", cert.NotAfter.Format(time.RFC1123))
	f.UnIndent()

	f.Title("Subject Alt Names")
	f.Indent()

	for _, dnsName := range cert.DNSNames {
		f.Value("DNS Name", dnsName)
	}

	for _, ipAddr := range cert.IPAddresses {
		f.Value("IP Address", ipAddr.String())
	}

	for _, email := range cert.EmailAddresses {
		f.Value("Email Address", email)
	}

	for _, uri := range cert.URIs {
		f.Value("URI", uri.String())
	}

	f.UnIndent()

	f.Title("Public Key Info")
	f.Indent()
	f.Value("Algorithm", cert.PublicKeyAlgorithm.String())
	f.Value("Public Value", marshalPublicKey(cert.PublicKey))
	f.UnIndent()

	f.Title("Fingerprints")
	f.Indent()
	f.Value("SHA-256", fmt.Sprintf("% X", sha256.Sum256(cert.Raw)))
	//nolint:gosec //sha1 is used for checksum only, no need to be cryptographically secure
	f.Value("SHA-1", fmt.Sprintf("% X", sha1.Sum(cert.Raw)))
	f.UnIndent()

	f.Title("Signature")
	f.Indent()
	f.Value("Algorithm", cert.SignatureAlgorithm.String())
	f.Value("Public Value", prettyHexa(cert.Signature))
	f.UnIndent()

	f.Value("Serial Number", prettyHexa(cert.SerialNumber.Bytes()))
	f.Value("Key Usages", strings.Join(utils.KeyUsageToStrings(cert.KeyUsage), ", "))
	f.Value("Extended Key Usages", strings.Join(utils.ExtKeyUsagesToStrings(cert.ExtKeyUsage), ", "))
}

func marshalPublicKey(key any) string {
	var bytes []byte

	switch k := key.(type) {
	case *rsa.PublicKey:
		bytes = x509.MarshalPKCS1PublicKey(k)
	case *ecdsa.PublicKey:
		bytes = elliptic.Marshal(k, k.X, k.Y)
	case *ed25519.PublicKey:
		bytes = *k
	default:
		return text.FgRed.Sprintf("<unknown public key type>")
	}

	return prettyHexa(bytes)
}

func prettyHexa(bytes []byte) string {
	const (
		firstLinePrefix = " "
		firstLineLen    = 27
		lineLen         = 32
	)

	var lines []string

	if len(bytes) < firstLineLen {
		return fmt.Sprintf("% X", bytes)
	}

	lines = append(lines, fmt.Sprintf("%s% X", firstLinePrefix, bytes[:firstLineLen]))

	for i := firstLineLen; i < len(bytes); i += lineLen {
		end := utils.Min(i+lineLen, len(bytes))
		lines = append(lines, fmt.Sprintf("% X", bytes[i:end]))
	}

	return strings.Join(lines, "\n")
}

func displaySSHKeyInfo(f *Formatter, name, content string) {
	f.Title("SSH Public Key %q", name)
	f.Indent()

	defer f.UnIndent()

	key, err := utils.ParseSSHAuthorizedKey(content)
	if err != nil || key == nil {
		f.Error("Content", "<could not parse SSH public key>")

		return
	}

	f.Value("Type", key.Type())
	f.Value("SHA-256 Fingerprint", ssh.FingerprintSHA256(key))
	f.Value("MD5 Fingerprint", ssh.FingerprintLegacyMD5(key))
}

func displayPrivateKeyInfo(f *Formatter, name, content string) {
	f.Title("Private Key %q", name)
	f.Indent()

	defer f.UnIndent()

	pk, err := ssh.ParsePrivateKey([]byte(content))
	if err != nil {
		f.Error("Content", "<could not parse private key>")

		return
	}

	key := pk.PublicKey()

	f.Value("Type", key.Type())
	f.Value("SHA-256 Fingerprint", ssh.FingerprintSHA256(key))
	f.Value("MD5 Fingerprint", ssh.FingerprintLegacyMD5(key))
}
