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
	authentication.AddExternalCredentialType(TLSCertificate, &TLSCertHandler{})
	authentication.AddInternalCredentialType(TLSTrustedCertificate, &TLSTrustedCertHandler{})
}

type TLSCertHandler struct{}

func (*TLSCertHandler) CanOnlyHaveOne() bool { return false }

func (*TLSCertHandler) ToDB(cert, plainPk string) (certificate, encryptedPk string, err error) {
	encryptedPk, err = utils.AESCrypt(database.GCM, plainPk)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt the private key: %w", err)
	}

	return cert, encryptedPk, nil
}

func (*TLSCertHandler) FromDB(cert, encryptedPk string) (certificate, plainPk string, err error) {
	plainPk, err = utils.AESDecrypt(database.GCM, encryptedPk)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt the private key: %w", err)
	}

	return cert, plainPk, nil
}

func (*TLSCertHandler) Validate(value, value2, _, host string, isServer bool) error {
	if err := checkCert(value, value2, host, isServer); err != nil {
		return fmt.Errorf("failed to validate certificate: %w", err)
	}

	return nil
}

type TLSTrustedCertHandler struct{}

func (*TLSTrustedCertHandler) CanOnlyHaveOne() bool { return false }

func (*TLSTrustedCertHandler) Validate(value, _, _, host string, isServer bool) error {
	if err := checkRemoteSelfSignedCert(value, host, isServer); err != nil {
		return fmt.Errorf("failed to validate certificate: %w", err)
	}

	return nil
}

func (*TLSTrustedCertHandler) Authenticate(db database.ReadAccess,
	owner authentication.Owner, val any,
) (*authentication.Result, error) {
	doVerify := func(chain []*x509.Certificate) (*authentication.Result, error) {
		rootCAs, rootErr := makeRootCAs(db, owner)
		if rootErr != nil {
			return nil, rootErr
		}

		if err := verifyCertChain(chain, rootCAs, owner.Host(),
			owner.IsServer()); err != nil {
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

func verifyCert(cert *x509.Certificate, trustedRoots []*x509.Certificate,
	options *x509.VerifyOptions,
) error {
	roots := utils.TLSCertPool()

	for _, root := range trustedRoots {
		roots.AddCert(root)
	}

	// if subject == issuer, then the certificate is self-signed, so we add it to the roots
	if bytes.Equal(cert.RawSubject, cert.RawIssuer) {
		roots.AddCert(cert)
	}

	options.Roots = roots

	if _, err := cert.Verify(*options); err != nil {
		return fmt.Errorf("certificate is invalid: %w", err)
	}

	return nil
}

func checkCert(certPEM, keyPEM, host string, isServer bool) error {
	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return fmt.Errorf("failed to parse the x509 certificate: %w", err)
	}

	//nolint:errcheck //cert if already parsed above, so checking for errors here is redundant
	leaf, _ := x509.ParseCertificate(cert.Certificate[0])

	options := &x509.VerifyOptions{DNSName: host}
	if isServer {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	return verifyCert(leaf, nil, options)
}

func checkRemoteSelfSignedCert(certPEM, host string, isServer bool) error {
	cert, err := ParseCertPEM(certPEM)
	if err != nil {
		return err
	}

	options := &x509.VerifyOptions{DNSName: host}
	if isServer {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	return verifyCert(cert, nil, options)
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
	host string, isServer bool,
) error {
	options := x509.VerifyOptions{
		DNSName:       host,
		Roots:         rootCAs,
		Intermediates: x509.NewCertPool(),
	}

	if isServer {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		options.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
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

		login := certs[0].Subject.CommonName
		if login == "" {
			return errors.New("tls: missing client certificate common name")
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

		if res, err := acc.Authenticate(db, server, TLSTrustedCertificate, certs); err != nil {
			logger.Errorf("Failed to authenticate client certificate: %v", err)

			return errors.New("internal authentication error")
		} else if !res.Success {
			logger.Warningf("Failed to verify client certificate %q: %v", login, res.Reason)

			return fmt.Errorf("invalid client certificate: %s", res.Reason)
		}

		return nil
	}
}
