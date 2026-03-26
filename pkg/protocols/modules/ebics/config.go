package ebics

import (
	"fmt"
	"net/url"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

type serverConfig struct {
	ProtocolVersion string                `json:"protocolVersion,omitempty"`
	RequestTimeout  int32                 `json:"requestTimeout,omitempty"`
	MaxSegmentSize  int64                 `json:"maxSegmentSize,omitempty"`
	AllowRecovery   bool                  `json:"allowRecovery,omitempty"`
	MinTLSVersion   protoutils.TLSVersion `json:"minTLSVersion,omitempty"`
	VerifyBankKeys  bool                  `json:"verifyBankKeys,omitempty"`
}

type clientConfig struct {
	ProtocolVersion          string                `json:"protocolVersion,omitempty"`
	EndpointURL              string                `json:"endpointURL,omitempty"`
	RequestTimeout           int32                 `json:"requestTimeout,omitempty"`
	MaxSegmentSize           int64                 `json:"maxSegmentSize,omitempty"`
	AllowRecovery            bool                  `json:"allowRecovery,omitempty"`
	MinTLSVersion            protoutils.TLSVersion `json:"minTLSVersion,omitempty"`
	VerifyBankKeys           bool                  `json:"verifyBankKeys,omitempty"`
	DefaultOrderDataEncoding string                `json:"defaultOrderDataEncoding,omitempty"`
	ProfilePolicy            string                `json:"profilePolicy,omitempty"`
}

type partnerConfig struct {
	ProtocolVersion string `json:"protocolVersion,omitempty"`
	EndpointURL     string `json:"endpointURL,omitempty"`
	HostID          string `json:"hostID,omitempty"`
	UseWSSRTN       bool   `json:"useWSSRTN,omitempty"` //nolint:tagliatelle // external config key should stay explicit
}

const (
	defaultRequestTimeoutSeconds int32 = 30
	defaultMaxSegmentSizeBytes   int64 = 1024 * 1024
)

func defaultServerConfig() *serverConfig {
	return &serverConfig{
		ProtocolVersion: protocolVersionH005,
		RequestTimeout:  defaultRequestTimeoutSeconds,
		MaxSegmentSize:  defaultMaxSegmentSizeBytes,
		AllowRecovery:   true,
		VerifyBankKeys:  true,
	}
}

func defaultClientConfig() *clientConfig {
	return &clientConfig{
		ProtocolVersion: protocolVersionH005,
		RequestTimeout:  defaultRequestTimeoutSeconds,
		MaxSegmentSize:  defaultMaxSegmentSizeBytes,
		AllowRecovery:   true,
		VerifyBankKeys:  true,
		ProfilePolicy:   profilePreferred,
	}
}

func defaultPartnerConfig() *partnerConfig {
	return &partnerConfig{
		ProtocolVersion: protocolVersionH005,
	}
}

func (c *serverConfig) ValidServer() error {
	normalized := defaultServerConfig()
	if c.ProtocolVersion == "" {
		c.ProtocolVersion = normalized.ProtocolVersion
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = normalized.RequestTimeout
	}
	if c.MaxSegmentSize == 0 {
		c.MaxSegmentSize = normalized.MaxSegmentSize
	}
	if err := validateProtocolVersion(c.ProtocolVersion); err != nil {
		return err
	}
	if c.RequestTimeout <= 0 {
		return database.NewValidationError("requestTimeout must be greater than zero")
	}
	if c.MaxSegmentSize <= 0 {
		return database.NewValidationError("maxSegmentSize must be greater than zero")
	}

	return nil
}

func (c *clientConfig) ValidClient() error {
	normalized := defaultClientConfig()
	if c.ProtocolVersion == "" {
		c.ProtocolVersion = normalized.ProtocolVersion
	}
	if c.RequestTimeout == 0 {
		c.RequestTimeout = normalized.RequestTimeout
	}
	if c.MaxSegmentSize == 0 {
		c.MaxSegmentSize = normalized.MaxSegmentSize
	}
	if c.ProfilePolicy == "" {
		c.ProfilePolicy = normalized.ProfilePolicy
	}
	if err := validateProtocolVersion(c.ProtocolVersion); err != nil {
		return err
	}
	if err := validateEndpointURL(c.EndpointURL, true); err != nil {
		return err
	}
	if err := validateProfilePolicy(c.ProfilePolicy); err != nil {
		return err
	}
	if c.RequestTimeout <= 0 {
		return database.NewValidationError("requestTimeout must be greater than zero")
	}
	if c.MaxSegmentSize <= 0 {
		return database.NewValidationError("maxSegmentSize must be greater than zero")
	}

	return nil
}

func (c *partnerConfig) ValidPartner() error {
	normalized := defaultPartnerConfig()
	if c.ProtocolVersion == "" {
		c.ProtocolVersion = normalized.ProtocolVersion
	}
	c.HostID = strings.TrimSpace(c.HostID)
	if err := validateProtocolVersion(c.ProtocolVersion); err != nil {
		return err
	}
	if err := validateEndpointURL(c.EndpointURL, false); err != nil {
		return err
	}
	if c.HostID == "" {
		return database.NewValidationError("hostID is missing")
	}

	return nil
}

func validateProtocolVersion(version string) error {
	switch strings.ToUpper(strings.TrimSpace(version)) {
	case protocolVersionH005:
		return nil
	case "":
		return database.NewValidationError("protocolVersion is missing")
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedVersion, version)
	}
}

func validateProfilePolicy(policy string) error {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case profileRequired, profilePreferred, freeInputAllowed:
		return nil
	case "":
		return database.NewValidationError("profilePolicy is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported profile policy", policy)
	}
}

func validateEndpointURL(rawURL string, required bool) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		if required {
			return database.NewValidationError("endpointURL is missing")
		}

		return nil
	}

	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return database.NewValidationErrorf("invalid endpointURL: %v", err)
	}
	if !strings.EqualFold(parsed.Scheme, transportHTTPS) {
		return fmt.Errorf("%w: %q", ErrUnsupportedTransport, parsed.Scheme)
	}

	return nil
}
