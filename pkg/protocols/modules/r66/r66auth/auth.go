package r66auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
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
} = &LegacyCertificate{}

var _ authentication.InternalAuthHandler = &BcryptAuthHandler{}

type BcryptAuthHandler struct{ auth.BcryptAuthHandler }

func (r *BcryptAuthHandler) ToDB(plainPwd, _ string) (hashedPwd, _ string, err error) {
	if utils.IsHash(plainPwd) {
		return plainPwd, "", nil
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return r.BcryptAuthHandler.ToDB(CryptPass(plainPwd), "")
}

var ErrLegacyCertNotAllowed = errors.New("legacy certificates usage is not allowed on this instance")

type LegacyCertificate struct{}

func (r *LegacyCertificate) CanOnlyHaveOne() bool { return true }

func (r *LegacyCertificate) Validate(_, _, _, _ string, _ bool) error {
	if !compatibility.IsLegacyR66CertificateAllowed {
		return ErrLegacyCertNotAllowed
	}

	return nil
}

func (r *LegacyCertificate) Authenticate(db database.ReadAccess,
	owner authentication.Owner, val any,
) (*authentication.Result, error) {
	if !compatibility.IsLegacyR66CertificateAllowed || !UsesLegacyCert(db, owner) {
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
		//nolint:err113 //this is a base error
		return nil, fmt.Errorf(`type "%T" is not an acceptable TLS certificate type`, value)
	}

	if !compatibility.IsLegacyR66Cert(cert) {
		return authentication.Failure("unknown certificate"), nil
	}

	return authentication.Success(), nil
}

func (r *LegacyCertificate) ToDB(_, _ string) (_, _ string, err error) {
	return "", "", nil
}

func UsesLegacyCert(db database.ReadAccess, owner authentication.Owner) bool {
	if compatibility.IsLegacyR66CertificateAllowed {
		if n, err := db.Count(&model.Credential{}).Where(owner.GetCredCond()).
			Where("type=?", AuthLegacyCertificate).Run(); err == nil {
			return n != 0
		}
	}

	return false
}

// CryptPass returns the R66 hash of the given password.
func CryptPass(pwd string) string {
	return string(r66.CryptPass([]byte(pwd)))
}
