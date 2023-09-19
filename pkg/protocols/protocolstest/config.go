package protocolstest

type TestConfig map[string]any

func (TestConfig) ValidServer() error  { return nil }
func (TestConfig) ValidPartner() error { return nil }
func (TestConfig) ValidClient() error  { return nil }
