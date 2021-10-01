package testhelpers

// TestProtoConfig is a dummy implementation of config.ProtoConfig for test purposes.
type TestProtoConfig struct{}

// ValidServer is a dummy implementation of the server validation function.
// It does nothing, and never fails.
func (*TestProtoConfig) ValidServer() error { return nil }

// ValidPartner is a dummy implementation of the partner validation function.
// It does nothing, and never fails.
func (*TestProtoConfig) ValidPartner() error { return nil }
