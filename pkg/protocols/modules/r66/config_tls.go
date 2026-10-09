package r66

import "code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"

// ServerConfigTLS represents the configuration of a local R66-TLS server.
type ServerConfigTLS struct {
	SharedServerConfig

	// The server's password for server authentication.
	//
	// Deprecated: use model.Credential instead.
	ServerPassword string `json:"serverPassword,omitempty"`

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting to this server. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (c *ServerConfigTLS) ValidConf() error {
	return c.ValidShared()
}

// PartnerConfigTLS represents the configuration of a remote R66-TLS partner.
type PartnerConfigTLS struct {
	SharedPartnerConfig

	// The server's password for server authentication.
	//
	// Deprecated: use model.Credential instead.
	ServerPassword string `json:"serverPassword,omitempty"`

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting to this partner. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

//nolint:dupl //it's better to keep the TLS & non-TLS config separated, as they will probably differ in the future
func (c *PartnerConfigTLS) ValidConf() error {
	if err := hashServerPassword(&c.ServerPassword); err != nil {
		return err
	}

	return c.ValidShared()
}

// ClientConfigTLS represents the configuration of a local R66-TLS client.
type ClientConfigTLS struct {
	SharedClientConfig

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`

	// CipherSuites specifies the list of TLS cipher suites to use when
	// connecting using this client. If empty, Go defaults are used.
	CipherSuites protoutils.TLSCiphersList `json:"cipherSuites,omitempty"`
}

func (c *ClientConfigTLS) ValidConf() error {
	return c.ValidShared()
}
