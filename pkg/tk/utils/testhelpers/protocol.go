package testhelpers

import "errors"

var (
	ErrServerValidationFailed  = errors.New("server config validation failed")
	ErrPartnerValidationFailed = errors.New("partner config validation failed")
	ErrClientValidationFailed  = errors.New("client config validation failed")
)

// TestProtoConfig is a dummy implementation of config.ServerProtoConfig,
// config.PartnerProtoConfig & config.ClientProtoConfig for test purposes.
// The validation always succeeds.
type TestProtoConfig struct{}

// ValidServer is a dummy implementation of the server validation function.
// It does nothing, and never fails.
func (*TestProtoConfig) ValidServer() error { return nil }

// ValidPartner is a dummy implementation of the partner validation function.
// It does nothing, and never fails.
func (*TestProtoConfig) ValidPartner() error { return nil }

// ValidClient is a dummy implementation of the client validation function.
// It does nothing, and never fails.
func (*TestProtoConfig) ValidClient() error { return nil }

// TestProtoConfigFail is a dummy implementation of config.ServerProtoConfig,
// config.PartnerProtoConfig & config.ClientProtoConfig for test purposes.
// The validation always fails.
type TestProtoConfigFail struct{}

// ValidServer is a dummy implementation of the server validation function.
// It does nothing, and never fails.
func (*TestProtoConfigFail) ValidServer() error { return ErrServerValidationFailed }

// ValidPartner is a dummy implementation of the partner validation function.
// It does nothing, and never fails.
func (*TestProtoConfigFail) ValidPartner() error { return ErrPartnerValidationFailed }

// ValidClient is a dummy implementation of the client validation function.
// It does nothing, and never fails.
func (*TestProtoConfigFail) ValidClient() error { return ErrClientValidationFailed }
