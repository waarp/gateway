package r66

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

const (
	AuthLegacyCertificate = "r66_legacy_certificate"
)

var _ interface {
	authentication.InternalAuthHandler
	authentication.ExternalAuthHandler
} = &r66LegacyCertificate{}

var _ authentication.InternalAuthHandler = &r66BcryptAuthHandler{}

//nolint:gochecknoinits //init is used by design
func init() {
	authentication.AddInternalCredentialTypeForProtocol(auth.Password, R66, &r66BcryptAuthHandler{})
	authentication.AddInternalCredentialTypeForProtocol(auth.Password, R66TLS, &r66BcryptAuthHandler{})

	authentication.AddInternalCredentialTypeForProtocol(AuthLegacyCertificate, R66TLS, &r66LegacyCertificate{})
	authentication.AddExternalCredentialTypeForProtocol(AuthLegacyCertificate, R66TLS, &r66LegacyCertificate{})
}

type r66BcryptAuthHandler struct{ auth.BcryptAuthHandler }

func (r *r66BcryptAuthHandler) ToDB(val, _ string) (string, string, error) {
	if utils.IsHash(val) {
		return val, "", nil
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return r.BcryptAuthHandler.ToDB(CryptPass(val), "")
}

var ErrLegacyCertNotAllowed = errors.New("legacy certificates usage is not allowed on this instance")

type r66LegacyCertificate struct{}

func (r *r66LegacyCertificate) CanOnlyHaveOne() bool { return true }

func (r *r66LegacyCertificate) Validate(_, _, _, _ string, _ bool) error {
	if !compatibility.IsLegacyR66CertificateAllowed {
		return ErrLegacyCertNotAllowed
	}

	return nil
}

func (r *r66LegacyCertificate) Authenticate(db database.ReadAccess,
	owner authentication.Owner, val any,
) (*authentication.Result, error) {
	if !compatibility.IsLegacyR66CertificateAllowed || !usesLegacyCert(db, owner) {
		return nil, ErrLegacyCertNotAllowed
	}

	var cert *x509.Certificate

	switch value := val.(type) {
	case *tls.Certificate:
		var parsErr error
		if cert, parsErr = x509.ParseCertificate(value.Certificate[0]); parsErr != nil {
			return nil, fmt.Errorf("failed to parse TLS certificate: %w", parsErr)
		}
	case []*x509.Certificate:
		cert = value[0]
	default:
		//nolint:goerr113 //this is a base error
		return nil, fmt.Errorf(`type "%T" is not an acceptable TLS certificate type`, value)
	}

	if !compatibility.IsLegacyR66Cert(cert) {
		return authentication.Failure("unknown certificate"), nil
	}

	return authentication.Success(), nil
}

func (r *r66LegacyCertificate) ToDB(string, string) (string, string, error) {
	return "", "", nil
}
