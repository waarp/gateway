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
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
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
		rootCAs, rootErr := makeRootCAs(db, owner)
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

		return authentication.Success(), nil
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
	default:
		//nolint:err113 //this is a base error
		return nil, fmt.Errorf(`unknown TLS certificate type "%T"`, value)
	}
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

func makeRootCAs(db database.ReadAccess, owner authentication.Owner) (*x509.CertPool, error) {
	rootCAs := utils.TLSCertPool()

	var trustedCert model.Credentials
	if err := db.Select(&trustedCert).Where("type=?", TLSTrustedCertificate).
		Where(owner.GetCredCond()).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the trusted certificates: %w", err)
	}

	for i := range trustedCert {
		rootCAs.AppendCertsFromPEM([]byte(trustedCert[i].Value))
	}

	var trustedAuthorities model.Authorities
	if err := db.Select(&trustedAuthorities).Where("type=?", AuthorityTLS).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the TLS certification authorities: %w", err)
	}

	for _, aut := range trustedAuthorities {
		if len(aut.ValidHosts) == 0 || slices.Contains(aut.ValidHosts, owner.Host()) {
			rootCAs.AppendCertsFromPEM([]byte(aut.PublicIdentity))
		}
	}

	return rootCAs, nil
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

//nolint:err113 //dynamic errors are needed here
func VerifyClientCert(db database.ReadAccess, logger *log.Logger, server *model.LocalAgent,
) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return nil
		}

		certs := make([]*x509.Certificate, len(rawCerts))

		for i, asn1Data := range rawCerts {
			var err error
			if certs[i], err = x509.ParseCertificate(asn1Data); err != nil {
				logger.Warningf("Failed to parse client certificate: %v", err)

				return fmt.Errorf("tls: failed to parse client certificate: %w", err)
			}
		}

		leaf := certs[0]
		login := GetClientCertLogin(leaf)
		if login == "" {
			return errors.New("tls: client certificate has no subject")
		}

		var acc model.LocalAccount
		if err := db.Get(&acc, "local_agent_id=? AND login=?", server.ID, login).
			Run(); err != nil {
			if database.IsNotFound(err) {
				logger.Warningf("Unknown certificate subject %q", login)

				return fmt.Errorf("tls: unknown certificate subject %q", login)
			}

			logger.Errorf("Failed to retrieve user credentials: %v", err)

			return errors.New("failed to retrieve user credentials")
		}

		if res, err := acc.Authenticate(db, TLSTrustedCertificate, certs); err != nil {
			logger.Errorf("Failed to authenticate client certificate: %v", err)

			return errors.New("internal authentication error")
		} else if !res.Success {
			logger.Warningf("Failed to verify client certificate %q: %v", login, res.Reason)

			return fmt.Errorf("invalid client certificate: %s", res.Reason)
		}

		return nil
	}
}

func GetClientCertLogin(cert *x509.Certificate) (login string) {
	for _, dns := range cert.DNSNames {
		return dns
	}

	for _, email := range cert.EmailAddresses {
		return email
	}

	return cert.Subject.CommonName
}
