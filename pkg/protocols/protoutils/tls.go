// Package protoutils provides utility functions for protocol implementations.
package protoutils

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"slices"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrNoValidCert = errors.New("no valid x509 certificate found")

func MakeServerTLSConfig(db database.ReadAccess, logger *log.Logger, agentID int64,
) (*tls.Config, error) {
	var agent model.LocalAgent
	if err := db.Get(&agent, "id=?", agentID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve server agent from database: %w", err)
	}

	tlsCerts, err := GetServerCertificates(db, logger, &agent)
	if err != nil {
		return nil, err
	}

	if len(tlsCerts) == 0 {
		logger.Errorf("Could not find a valid certificate for %s server", agent.Protocol)

		return nil, ErrNoValidCert
	}

	return &tls.Config{
		MinVersion:       GetMinTLSVersion(agent.ProtoConfig),
		Certificates:     tlsCerts,
		ClientAuth:       tls.RequestClientCert,
		VerifyConnection: LogTLSConn(logger),
		CipherSuites:     GetTLSCiphers(agent.ProtoConfig),
	}, nil
}

func GetServerCertificates(db database.ReadAccess, logger *log.Logger, owner model.CredOwnerTable,
) ([]tls.Certificate, error) {
	creds, dbErr := owner.GetCredentials(db, auth.TLSCertificate)
	if dbErr != nil {
		logger.Errorf("Failed to retrieve server certificates: %s", dbErr)

		return nil, fmt.Errorf("failed to retrieve server certificates: %w", dbErr)
	}

	var tlsCerts []tls.Certificate

	for _, cred := range creds {
		cert, err := tls.X509KeyPair([]byte(cred.Value), []byte(cred.Value2))
		if err != nil {
			logger.Warningf("Failed to parse server certificate: %v", err)

			continue
		}

		tlsCerts = append(tlsCerts, cert)
	}

	return tlsCerts, nil
}

func GetServerTLSConfig(db database.ReadAccess, logger *log.Logger, agentID int64,
) *tls.Config {
	return &tls.Config{
		GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
			return MakeServerTLSConfig(db, logger, agentID)
		},
	}
}

func GetClientTLSConfig(ctx *model.TransferContext, logger *log.Logger) (*tls.Config, error) {
	return GetClientTLSConf(logger, ctx.RemoteAgent, ctx.Client,
		ctx.RemoteAgentCreds, ctx.RemoteAccountCreds, ctx.Authorities)
}

func GetClientTLSConf(logger *log.Logger, partner *model.RemoteAgent,
	client *model.Client, partnerCreds, accountCreds []*model.Credential,
	authorities []*model.Authority,
) (*tls.Config, error) {
	minVersion := GetMinTLSVersion(client.ProtoConfig)
	if partMinVersion := GetMinTLSVersion(partner.ProtoConfig); partMinVersion != 0 {
		minVersion = partMinVersion
	}

	cipherSuites := GetTLSCiphers(client.ProtoConfig)
	if partCiphers := GetTLSCiphers(partner.ProtoConfig); len(partCiphers) > 0 {
		cipherSuites = partCiphers
	}

	accCerts := make([]*tls.Certificate, 0, len(accountCreds))
	for _, cred := range accountCreds {
		if cred.Type != auth.TLSCertificate {
			continue
		}

		cert, err := utils.X509KeyPair(cred.Value, cred.Value2)
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate %s: %w", cred.Name, err)
		}

		accCerts = append(accCerts, &cert)
	}

	config := &tls.Config{
		ServerName:       partner.Address.Host,
		RootCAs:          utils.TLSCertPool(),
		VerifyConnection: LogTLSConn(logger),
		MinVersion:       minVersion,
		CipherSuites:     cipherSuites,
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			// Prefer the standards-compliant choice when one exists.
			for _, cert := range accCerts {
				if info.SupportsCertificate(cert) == nil {
					return cert, nil
				}
			}
			// Fallback: send the configured certificate anyway and let the server decide.
			if len(accCerts) > 0 {
				return accCerts[0], nil
			}

			return &tls.Certificate{}, nil
		},
	}

	for _, cred := range partnerCreds {
		if cred.Type != auth.TLSTrustedCertificate {
			continue
		}

		config.RootCAs.AppendCertsFromPEM([]byte(cred.Value))
	}

	for _, authority := range authorities {
		if len(authority.ValidHosts) != 0 && !slices.Contains(authority.ValidHosts, config.ServerName) {
			continue
		}

		config.RootCAs.AppendCertsFromPEM([]byte(authority.PublicIdentity))
	}

	return config, nil
}

func LogTLSConn(logger *log.Logger) func(tls.ConnectionState) error {
	return func(conn tls.ConnectionState) error {
		logger.Debugf("%s connection established using cipher suite %q",
			tls.VersionName(conn.Version),
			tls.CipherSuiteName(conn.CipherSuite))

		if len(conn.PeerCertificates) > 0 {
			logger.Debugf("Remote certificate: %s", displayCertInfo(conn.PeerCertificates[0]))
		}

		return nil
	}
}

func displayCertInfo(cert *x509.Certificate) string {
	return fmt.Sprintf("Subject=%s Issuer=%s NotBefore=%s NotAfter=%s DNS=[%s] IPs=[%s] PubKeyAlgo=%s SignAlgo=%s ",
		cert.Subject.CommonName,
		cert.Issuer.CommonName,
		cert.NotBefore.String(),
		cert.NotAfter.String(),
		strings.Join(cert.DNSNames, ", "),
		utils.JoinString(cert.IPAddresses, ", "),
		cert.PublicKeyAlgorithm.String(),
		cert.SignatureAlgorithm.String(),
	)
}
