// Package modeltest provides utilities for tests using the database models.
package modeltest

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// AddDummyProtoConfig adds a dummy protocol configuration to the model config
// checker. The checker will accept any configuration given to it for the
// specified protocol.
func AddDummyProtoConfig(name string) {
	model.Protocols[name] = dummyProtocol{}
}

func AddDummyProtoConfigWithErr(name string, err error) {
	model.Protocols[name] = dummyProtocol{err: err}
}

type dummyProtocol struct{ err error }

func (d dummyProtocol) CanMakeTransfer(*model.TransferContext) error { return d.err }
func (d dummyProtocol) CheckServerConfig(map[string]any) error       { return d.err }
func (d dummyProtocol) CheckClientConfig(map[string]any) error       { return d.err }
func (d dummyProtocol) CheckPartnerConfig(map[string]any) error      { return d.err }
