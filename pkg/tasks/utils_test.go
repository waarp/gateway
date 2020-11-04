package tasks

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

func fileNotFound(path string, op ...string) *errFileNotFound {
	if len(op) > 0 {
		return &errFileNotFound{op[0], path}
	}
	return &errFileNotFound{"open", path}
}
