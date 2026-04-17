package as2

import (
	"errors"
	"strings"
)

var ErrInvalidAsyncConfig = errors.New("cannot handle async MDN with empty address")

type clientProtoConfig struct {
	// MaxFileSize is the maximum file size (in bytes) that can be sent by the
	// client. It cannot be larger than the total system memory.
	// By default, it is set to 1MB.
	MaxFileSize BufferSize `json:"maxFileSize"`
}

func (c *clientProtoConfig) ValidConf() error {
	if c.MaxFileSize <= 0 {
		c.MaxFileSize = defaultBufSize
	}

	return nil
}

type partnerProtoConfig struct {
	// SignatureAlgorithm is the signature algorithm used to sign the message.
	// Accepted values are:
	// - "sha1"
	// - "md5"
	// - "sha256"
	// - "sha384"
	// - "sha512"
	SignatureAlgorithm SignAlgo `json:"signatureAlgorithm"`

	// EncryptionAlgorithm is the encryption algorithm used to encrypt the message.
	// Accepted values are:
	// - "des-cbc"
	// - "aes128-cbc"
	// - "aes128-gcm"
	// - "aes256-cbc"
	// - "aes256-gcm"
	EncryptionAlgorithm EncryptAlgo `json:"encryptionAlgorithm"`

	// The address at which the acknowledgement MDN should be sent. Can be an
	// IP or a hostname. Must be set if UseAsyncMDN is on. If empty, no MDN will
	// be sent.
	AsyncMDNAddress string `json:"asyncMDNAddress,omitempty"`

	// If true, the client will handle asynchronous MDNs received at AsyncMDNAddress
	// directly. If false, the async MDN is assumed to be handled by a third-party
	// service. It is invalid to set this to true if AsyncMDNAddress is empty.
	HandleAsyncMDN bool `json:"handleAsyncMDN,omitempty"`
}

func (p *partnerProtoConfig) ValidConf() error {
	if p.HandleAsyncMDN && p.AsyncMDNAddress == "" {
		return ErrInvalidAsyncConfig
	}

	if p.AsyncMDNAddress != "" {
		p.AsyncMDNAddress = strings.TrimPrefix(p.AsyncMDNAddress, "http://")
		p.AsyncMDNAddress = strings.TrimPrefix(p.AsyncMDNAddress, "https://")
	}

	return nil
}

type serverProtoConfig struct {
	// MaxFileSize is the maximum file size (in bytes) that can be received by
	// the server. It cannot be larger than the total system memory.
	// By default, it is set to 1MB.
	MaxFileSize BufferSize `json:"maxFileSize"`

	// MDNSignatureAlgorithm is the signature algorithm used by the server to
	// sign the MDN.
	// Accepted values are:
	// - "sha1"
	// - "md5"
	// - "sha256"
	// - "sha384"
	// - "sha512"
	MDNSignatureAlgorithm SignAlgo `json:"mdnSignatureAlgorithm"`
}

func (s *serverProtoConfig) ValidConf() error {
	if s.MaxFileSize <= 0 {
		s.MaxFileSize = defaultBufSize
	}

	return nil
}
