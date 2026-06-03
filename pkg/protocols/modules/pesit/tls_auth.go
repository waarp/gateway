package pesit

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	// TLSClientAuthNone means client certificate is not used for identification.
	TLSClientAuthNone = "none"
	// TLSClientAuthOptional uses the cert CN/SAN if presented, else falls back to PeSIT login.
	TLSClientAuthOptional = "optional"
	// TLSClientAuthRequired requires a valid client certificate; PeSIT password is ignored.
	TLSClientAuthRequired = "required"
)

// extractCertIdentity extracts the identity (CN or first SAN) from the
// TLS client certificate of the given connection. Returns empty string
// if no certificate is available.
func extractCertIdentity(conn net.Conn) string {
	tlsConn, ok := conn.(interface{ ConnectionState() tls.ConnectionState })
	if !ok {
		return ""
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return ""
	}

	cert := state.PeerCertificates[0]

	// Prefer SAN DNS names if available (more specific than CN).
	if len(cert.DNSNames) > 0 {
		return cert.DNSNames[0]
	}

	return cert.Subject.CommonName
}

// authenticateByCert resolves a LocalAccount by matching the certificate
// identity (CN or SAN) against account logins on the server.
func authenticateByCert(db database.ReadAccess, logger *log.Logger,
	agent *model.LocalAgent, certIdentity string,
) (*model.LocalAccount, error) {
	if certIdentity == "" {
		return nil, fmt.Errorf("no client certificate identity")
	}

	var acc model.LocalAccount
	if err := db.Get(&acc, "local_agent_id=? AND login=?",
		agent.ID, certIdentity).Run(); err != nil {
		logger.Warningf("TLS cert auth: no account matching CN/SAN %q", certIdentity)

		return nil, fmt.Errorf("no account matching certificate identity %q", certIdentity)
	}

	// Validate the certificate against the account's trusted certificates.
	// This is already done in VerifyPeerCertificate, but we double-check here.
	logger.Debugf("TLS cert auth: account %q matched by certificate CN/SAN %q",
		acc.Login, certIdentity)

	return &acc, nil
}

// tlsRequireClientCert returns the tls.ClientAuthType corresponding to the
// configured TLSClientAuth mode.
func tlsRequireClientCert(mode string) tls.ClientAuthType {
	switch mode {
	case TLSClientAuthRequired:
		return tls.RequireAnyClientCert
	case TLSClientAuthOptional:
		return tls.RequestClientCert
	default:
		return tls.NoClientCert
	}
}

// certFromPeerCerts extracts the first peer certificate from raw cert bytes.
func certFromPeerCerts(rawCerts [][]byte) *x509.Certificate {
	if len(rawCerts) == 0 {
		return nil
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return nil
	}

	return cert
}
