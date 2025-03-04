package pesit

import (
	"crypto/tls"
	"errors"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrNoCertificates = errors.New("no valid TLS certificate found")
	ErrCertDBError    = errors.New("failed to retrieve the TLS certificates from the database")
)

func (s *server) getCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	credentials, dbErr := s.localAgent.GetCredentials(s.db, auth.TLSCertificate)
	if dbErr != nil {
		s.logger.Error("Failed to retrieve the TLS certificates: %v", dbErr)

		return nil, ErrCertDBError
	}

	for _, cred := range credentials {
		cert, err := utils.X509KeyPair(cred.Value, cred.Value2)
		if err != nil {
			s.logger.Warning("Failed to parse the TLS certificate %q: %v", cred.Name, err)

			continue
		}

		if info.SupportsCertificate(&cert) == nil {
			return &cert, nil
		}
	}

	return nil, ErrNoCertificates
}
