package r66

import "code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"

// tlsServerConfig represents the configuration of a local R66-TLS server.
type tlsServerConfig struct {
	sharedServerConfig

	// The server's password for server authentication.
	// Deprecated: use model.Credential instead.
	ServerPassword string `json:"serverPassword,omitempty"`

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

func (c *tlsServerConfig) ValidServer() error {
	if err := encryptServerPassword(&c.ServerPassword); err != nil {
		return err
	}

	return c.ValidShared()
}

// tlsPartnerConfig represents the configuration of a remote R66-TLS partner.
type tlsPartnerConfig struct {
	sharedPartnerConfig

	// The server's password for server authentication.
	// Deprecated: use model.Credential instead.
	ServerPassword string `json:"serverPassword,omitempty"`

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

//nolint:dupl //it's better to keep the TLS & non-TLS config separated, as they will probably differ in the future
func (c *tlsPartnerConfig) ValidPartner() error {
	if err := hashServerPassword(&c.ServerPassword); err != nil {
		return err
	}

	return c.ValidShared()
}

// tlsClientConfig represents the configuration of a local R66-TLS client.
type tlsClientConfig struct {
	sharedClientConfig

	// MinTLSVersion specifies the minimum TLS version that the server should
	// allow. The accepted values are "v1.0", "v1.1", "v1.2", and "v1.3". The
	// default is "v1.2".
	MinTLSVersion protoutils.TLSVersion `json:"minTLSVersion"`
}

func (c *tlsClientConfig) ValidClient() error {
	return c.ValidShared()
}
