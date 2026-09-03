package auth

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	TLSCertificate        = "tls_certificate"
	TLSTrustedCertificate = "trusted_tls_certificate"
)

//nolint:gochecknoinits //init is used by design
func init() {
	authentication.AddExternalCredentialType(TLSCertificate, TLSCertHandler{})
	authentication.AddInternalCredentialType(TLSTrustedCertificate, TLSTrustedCertHandler{})
}

type TLSCertHandler struct{}

func (TLSCertHandler) CanOnlyHaveOne() bool { return false }

func (TLSCertHandler) ToDB(db database.Access, cert, plainPk string) (certificate, encryptedPk string, err error) {
	encryptedPk, err = db.Encrypt(plainPk)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt the private key: %w", err)
	}

	return cert, encryptedPk, nil
}

func (TLSCertHandler) FromDB(db database.ReadAccess, cert, encryptedPk string,
) (certificate, plainPk string, err error) {
	plainPk, err = db.Decrypt(encryptedPk)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt the private key: %w", err)
	}

	return cert, plainPk, nil
}

func (TLSCertHandler) Validate(db database.ReadAccess, value, value2, _, host string, isServer bool) error {
	if err := checkCert(db, value, value2, host, isServer); err != nil {
		return fmt.Errorf("failed to validate certificate: %w", err)
	}

	return nil
}

type TLSTrustedCertHandler struct{}

func (TLSTrustedCertHandler) CanOnlyHaveOne() bool { return false }

func (TLSTrustedCertHandler) ToDB(_ database.Access, cert, _ string,
) (certificate, _ string, err error) {
	return cert, "", nil
}

func (TLSTrustedCertHandler) Validate(_ database.ReadAccess, value, _, _, host string, isServer bool) error {
	if err := checkRemoteSelfSignedCert(value, host, isServer); err != nil {
		return fmt.Errorf("failed to validate certificate: %w", err)
	}

	return nil
}

func (TLSTrustedCertHandler) Authenticate(db database.ReadAccess,
	owner authentication.Owner, val any,
) (*authentication.Result, error) {
	doVerify := func(chain []*x509.Certificate) (*authentication.Result, error) {
		return AuthenticateRemoteCert(db, owner, chain)
	}

	switch value := val.(type) {
	case *tls.Certificate:
		chain, err := parseTLSCertChain(value)
		if err != nil {
			return nil, err
		}

		return doVerify(chain)
	case []*x509.Certificate:
		return doVerify(value)
	case nil:
		return authentication.Failure("no certificate chain provided"), nil
	default:
		//nolint:err113 //this is a base error
		return nil, fmt.Errorf(`unknown TLS certificate type "%T"`, value)
	}
}

func AuthenticateRemoteCert(db database.ReadAccess, owner authentication.Owner, chain []*x509.Certificate,
) (*authentication.Result, error) {
	if len(chain) == 0 {
		return authentication.Failure("no certificate chain provided"), nil
	}

	rootCAs, trustedCerts, rootErr := makeRootCAs(db, owner)
	if rootErr != nil {
		return nil, rootErr
	}

	usage := x509.ExtKeyUsageServerAuth
	if !owner.IsServer() {
		usage = x509.ExtKeyUsageClientAuth
	}

	if err := verifyCertChain(chain, rootCAs, owner.Host(), usage); err != nil {
		return authentication.Failure(err.Error()), nil
	}

	if !owner.IsServer() {
		if !matchClientCert(owner, trustedCerts, chain[0]) {
			return authentication.Failure("unknown client certificate"), nil
		}
	}

	return authentication.Success(), nil
}

var errInvalidPEM = errors.New("certificate input is not a valid PEM block")

func ParseCertPEM(pemBlock string) (*x509.Certificate, error) {
	var (
		cert  *x509.Certificate
		block *pem.Block
	)

	block, _ = pem.Decode([]byte(pemBlock))
	if block == nil {
		return nil, errInvalidPEM
	}

	if block.Type == "CERTIFICATE" {
		var err error
		if cert, err = x509.ParseCertificate(block.Bytes); err != nil {
			return nil, fmt.Errorf("failed to parsee x509 certificate: %w", err)
		}
	} else {
		//nolint:err113 //this is a base error
		return nil, fmt.Errorf("invalid PEM block type %q", block.Type)
	}

	return cert, nil
}

func ParseRawCertChain(rawCerts [][]byte) ([]*x509.Certificate, error) {
	certs := make([]*x509.Certificate, len(rawCerts))

	for i, rawCert := range rawCerts {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			return nil, fmt.Errorf("failed to parse x509 certificate: %w", err)
		}

		certs[i] = cert
	}

	return certs, nil
}

func checkCert(db database.ReadAccess, certPEM, keyPEM, host string, isServer bool) error {
	tlsCert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return fmt.Errorf("failed to parse the key/certificate pair: %w", err)
	}

	chain, err := ParseRawCertChain(tlsCert.Certificate)
	if err != nil {
		return err
	}

	usage := x509.ExtKeyUsageServerAuth
	if !isServer {
		usage = x509.ExtKeyUsageClientAuth
	}

	roots, err := newTLSRootPool(db, host)
	if err != nil {
		return err
	}

	if len(chain) == 1 && bytes.Equal(chain[0].RawIssuer, chain[0].RawSubject) {
		roots.AddCert(chain[0])
	}

	return verifyCertChain(chain, roots, host, usage)
}

func checkRemoteSelfSignedCert(certPEM, host string, isServer bool) error {
	cert, err := ParseCertPEM(certPEM)
	if err != nil {
		return err
	}

	usage := x509.ExtKeyUsageServerAuth
	if !isServer {
		usage = x509.ExtKeyUsageClientAuth
	}

	roots := x509.NewCertPool()
	roots.AddCert(cert)

	return verifyCertChain([]*x509.Certificate{cert}, roots, host, usage)
}

func parseTLSCertChain(cert *tls.Certificate) ([]*x509.Certificate, error) {
	chain := make([]*x509.Certificate, 0, len(cert.Certificate))

	for _, raw := range cert.Certificate {
		c, err := x509.ParseCertificate(raw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TLS certificate: %w", err)
		}

		chain = append(chain, c)
	}

	return chain, nil
}

func makeRootCAs(db database.ReadAccess, owner authentication.Owner,
) (*x509.CertPool, []*x509.Certificate, error,
) {
	rootCAs := utils.TLSCertPool()

	var trustedCreds model.Credentials
	if err := db.Select(&trustedCreds).Where("type=?", TLSTrustedCertificate).
		Where(owner.GetCredCond()).Run(); err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve the trusted certificates: %w", err)
	}

	trustedCerts := make([]*x509.Certificate, len(trustedCreds))
	for i, cred := range trustedCreds {
		cert, err := ParseCertPEM(cred.Value)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse credential %q: %w", cred.Name, err)
		}

		trustedCerts[i] = cert
		rootCAs.AddCert(cert)
	}

	var trustedAuthorities model.Authorities
	if err := db.Select(&trustedAuthorities).Where("type=?", AuthorityTLS).Run(); err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve the TLS certification authorities: %w", err)
	}

	for _, aut := range trustedAuthorities {
		if len(aut.ValidHosts) == 0 || slices.Contains(aut.ValidHosts, owner.Host()) {
			rootCAs.AppendCertsFromPEM([]byte(aut.PublicIdentity))
		}
	}

	return rootCAs, trustedCerts, nil
}

func verifyCertChain(certChain []*x509.Certificate, rootCAs *x509.CertPool,
	host string, usages ...x509.ExtKeyUsage,
) error {
	options := x509.VerifyOptions{
		DNSName:       host,
		Roots:         rootCAs,
		Intermediates: x509.NewCertPool(),
		KeyUsages:     usages,
	}

	for i := 1; i < len(certChain); i++ {
		options.Intermediates.AddCert(certChain[i])
	}

	if _, err := certChain[0].Verify(options); err != nil {
		//nolint:wrapcheck //wrapping here adds nothing
		return err
	}

	return nil
}

func matchClientCert(owner authentication.Owner, trustedCerts []*x509.Certificate,
	leaf *x509.Certificate,
) bool {
	locAcc, isLocAcc := owner.(*model.LocalAccount)
	if !isLocAcc {
		return false
	}

	if slices.Contains(leaf.DNSNames, locAcc.Login) {
		return true
	}

	if leaf.Subject.CommonName == locAcc.Login {
		return true
	}

	for _, candidate := range trustedCerts {
		if candidate.Equal(leaf) {
			return true
		}
	}

	return false
}
