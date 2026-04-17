// Package protoutils provides utility functions for protocol implementations.
package protoutils

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

type TLSVersion int

const (
	TLSv10 = "v1.0"
	TLSv11 = "v1.1"
	TLSv12 = "v1.2"
	TLSv13 = "v1.3"

	DefaultTLSVersion = tls.VersionTLS12
)

func TLSVersionFromString(v string) (TLSVersion, error) {
	switch v {
	case "", "null":
		return DefaultTLSVersion, nil
	case TLSv10:
		return tls.VersionTLS10, nil
	case TLSv11:
		return tls.VersionTLS11, nil
	case TLSv12:
		return tls.VersionTLS12, nil
	case TLSv13:
		return tls.VersionTLS13, nil
	default:
		return 0, UnsupportedTLSVersionError(v)
	}
}

func (t TLSVersion) TLS() uint16 { return uint16(t) }

func (t TLSVersion) String() string {
	switch t {
	case 0:
		return TLSVersion(DefaultTLSVersion).String()
	case tls.VersionTLS10:
		return TLSv10
	case tls.VersionTLS11:
		return TLSv11
	case tls.VersionTLS12:
		return TLSv12
	case tls.VersionTLS13:
		return TLSv13
	default:
		return fmt.Sprintf("<unknown TLS version %d>", t)
	}
}

func (t *TLSVersion) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err //nolint:wrapcheck //no need to wrap here
	}

	var err error
	*t, err = TLSVersionFromString(v)

	return err
}

func (t TLSVersion) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(t.String())), nil
}

type UnsupportedTLSVersionError string

func (e UnsupportedTLSVersionError) Error() string {
	return fmt.Sprintf("unknown TLS version %q (supported TLS versions: %s)", string(e),
		strings.Join([]string{TLSv10, TLSv11, TLSv12, TLSv13}, ", "))
}

var ErrNoValidCert = errors.New("no valid x509 certificate found")

func GetServerTLSConfig(db database.ReadAccess, logger *log.Logger,
	agent *model.LocalAgent, minVersion TLSVersion,
) func(*tls.ClientHelloInfo) (*tls.Config, error) {
	return func(*tls.ClientHelloInfo) (*tls.Config, error) {
		creds, dbErr := agent.GetCredentials(db, auth.TLSCertificate)
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

		if len(tlsCerts) == 0 {
			logger.Errorf("Could not find a valid certificate for %s server", agent.Protocol)

			return nil, ErrNoValidCert
		}

		return &tls.Config{
			MinVersion:            minVersion.TLS(),
			Certificates:          tlsCerts,
			ClientAuth:            tls.RequestClientCert,
			VerifyPeerCertificate: auth.VerifyClientCert(db, logger, agent),
			VerifyConnection:      compatibility.LogSha1(logger),
		}, nil
	}
}

func GetClientTLSConfig(ctx *model.TransferContext, logger *log.Logger, minVersion TLSVersion,
) (*tls.Config, error) {
	config := &tls.Config{
		ServerName:       ctx.RemoteAgent.Address.Host,
		RootCAs:          utils.TLSCertPool(),
		VerifyConnection: compatibility.LogSha1(logger),
		MinVersion:       minVersion.TLS(),
	}

	for _, cred := range ctx.RemoteAccountCreds {
		if cred.Type != auth.TLSCertificate {
			continue
		}

		cert, err := utils.X509KeyPair(cred.Value, cred.Value2)
		if err != nil {
			return nil, fmt.Errorf("failed to parse client certificate %s: %w", cred.Name, err)
		}

		config.Certificates = append(config.Certificates, cert)
	}

	for _, cred := range ctx.RemoteAgentCreds {
		if cred.Type != auth.TLSTrustedCertificate {
			continue
		}

		config.RootCAs.AppendCertsFromPEM([]byte(cred.Value))
	}

	for _, authority := range ctx.Authorities {
		if len(authority.ValidHosts) != 0 && !slices.Contains(authority.ValidHosts, config.ServerName) {
			continue
		}

		config.RootCAs.AppendCertsFromPEM([]byte(authority.PublicIdentity))
	}

	return config, nil
}
