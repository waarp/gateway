package main

import (
	"crypto/tls"
	"net"
	"slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func getTLSConf(host string, tlsVersion protoutils.TLSVersion, partnerCreds, accountCreds []api.OutCred,
	authorities []api.OutAuthority,
) *tls.Config {
	//nolint:errcheck //gateway gives host as a splitable host port
	servName, _, _ := net.SplitHostPort(host)
	conf := &tls.Config{
		ServerName: servName,
		MinVersion: tlsVersion.TLS(),
	}

	rootCA := utils.TLSCertPool()

	for _, authority := range authorities {
		if authority.Type == auth.AuthorityTLS {
			if slices.Contains(authority.ValidHosts, host) {
				rootCA.AppendCertsFromPEM([]byte(authority.PublicIdentity))
			}
		}
	}

	for _, cred := range partnerCreds {
		if cred.Type == auth.TLSTrustedCertificate {
			rootCA.AppendCertsFromPEM([]byte(cred.Value))
		}
	}

	conf.RootCAs = rootCA

	for _, cred := range accountCreds {
		if cred.Type == auth.TLSCertificate {
			cert, err := utils.X509KeyPair(cred.Value, cred.Value2)
			if err != nil {
				conf.Certificates = append(conf.Certificates, cert)
			}
		}
	}

	return conf
}
