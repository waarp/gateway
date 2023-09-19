package wg

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:varnamelen //formatter name is kept short for readability
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
	f.Value("Common name", cert.Subject.CommonName)
	f.ValueCond("Organization", cert.Subject.Organization)
	f.ValueCond("Country", cert.Subject.Country)
	f.UnIndent()

	f.Title("Issuer")
	f.Indent()
	f.Value("Common name", cert.Issuer.CommonName)
	f.ValueCond("Organization", cert.Issuer.Organization)
	f.ValueCond("Country", cert.Issuer.Country)
	f.UnIndent()

	f.Title("Validity")
	f.Indent()
	f.Value("Not before", cert.NotBefore.Format(time.UnixDate))
	f.Value("Not after", cert.NotAfter.Format(time.UnixDate))
	f.UnIndent()

	f.Title("Subject Alt Names")
	f.Indent()

	for _, dnsName := range cert.DNSNames {
		f.Value("DNS name", dnsName)
	}

	f.UnIndent()

	f.Value("Public Key Algorithm", cert.PublicKeyAlgorithm.String())
	f.Value("Signature Algorithm", cert.SignatureAlgorithm.String())
	f.Value("Signature", fmt.Sprintf("%X", cert.Signature))
	f.Value("Key Usages", strings.Join(utils.KeyUsageToStrings(cert.KeyUsage), ", "))
	f.Value("Extended Key Usages", strings.Join(utils.ExtKeyUsagesToStrings(cert.ExtKeyUsage), ", "))
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
	f.Value("Fingerprint", ssh.FingerprintSHA256(key))
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
	f.Value("Fingerprint", ssh.FingerprintSHA256(key))
}
