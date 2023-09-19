// Package modeltest provides utilities for tests using the database models.
package modeltest

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// AddDummyProtoConfig adds a dummy protocol configuration to the model config
// checker. The checker will accept any configuration given to it for the
// specified protocol.
func AddDummyProtoConfig(protocol string) {
	if model.ConfigChecker == nil {
		model.ConfigChecker = testConfigChecker{}
	}

	confChecker, ok := model.ConfigChecker.(testConfigChecker)
	if !ok {
		panic("AddDummyProtoConfig should only be called in a test")
	}

	confChecker[protocol] = nil
}

var ErrUnknownProtocol = errors.New("unknown protocol")

type testConfigChecker map[string]error

func (t testConfigChecker) checkConfig(proto string) error {
	if err, ok := t[proto]; ok {
		return err
	}

	return fmt.Errorf("%w %q", ErrUnknownProtocol, proto)
}

func (t testConfigChecker) IsValidProtocol(proto string) bool {
	return !errors.Is(t.checkConfig(proto), ErrUnknownProtocol)
}

func (t testConfigChecker) CheckServerConfig(proto string, _ map[string]any) error {
	return t.checkConfig(proto)
}

func (t testConfigChecker) CheckClientConfig(proto string, _ map[string]any) error {
	return t.checkConfig(proto)
}

func (t testConfigChecker) CheckPartnerConfig(proto string, _ map[string]any) error {
	return t.checkConfig(proto)
}
