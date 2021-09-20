package tasks

import "code.waarp.fr/apps/gateway/gateway/pkg/model/config"

//nolint:gochecknoinits // designed this way
func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }
func (*TestProtoConfig) CertRequired() bool  { return false }

func fileNotFound(path string, op ...string) *FileNotFoundError {
	if len(op) > 0 {
		return &FileNotFoundError{op[0], path}
	}

	return &FileNotFoundError{"open", path}
}
