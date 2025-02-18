package pesit

import (
	"crypto/tls"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func (c *clientTransfer) makeTLSConfig(servName string, conf *PartnerConfigTLS,
) (*tls.Config, error) {
	certs := make([]tls.Certificate, 0, len(c.pip.TransCtx.RemoteAccountCreds))

	for _, cred := range c.pip.TransCtx.RemoteAccountCreds {
		if cred.Type != auth.TLSCertificate {
			continue
		}

		cert, err := utils.X509KeyPair(cred.Value, cred.Value2)
		if err != nil {
			c.pip.Logger.Warning("Failed to parse the TLS certificate %q: %v", cred.Name, err)

			continue
		}

		certs = append(certs, cert)
	}

	rootCAs := utils.TLSCertPool()

	for _, cred := range c.pip.TransCtx.RemoteAgentCreds {
		if cred.Type != auth.TLSTrustedCertificate {
			continue
		}

		if !rootCAs.AppendCertsFromPEM([]byte(cred.Value)) {
			c.pip.Logger.Warning("Failed to parse the remote TLS certificate %q", cred.Name)
		}
	}

	//nolint:gosec //the TLS min version is set by the user
	tlsConfig := &tls.Config{
		ServerName:   servName,
		MinVersion:   protoutils.ParseTLSVersion(conf.MinTLSVersion),
		Certificates: certs,
		RootCAs:      rootCAs,
	}

	if err := auth.AddTLSAuthorities(c.pip.DB, tlsConfig); err != nil {
		return nil, fmt.Errorf("failed to setup the TLS authorities: %w", err)
	}

	return tlsConfig, nil
}
