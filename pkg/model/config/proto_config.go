// Package config contains the stucts representing the different kinds of
// protocol configuration.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	errInvalidProtoConfig = errors.New("the protocol configuration is invalid")
	errUnknownProtocol    = errors.New("unknown protocol")
)

// ProtoConfigs is a map associating each transfer protocol with their respective
// struct constructor.
//nolint:gochecknoglobals // global var is used by design
var ProtoConfigs = map[string]func() ProtoConfig{}

// ProtoConfig is the interface implemented by protocol configuration structs.
// It exposes 2 methods needed for validating the configuration.
type ProtoConfig interface {
	ValidServer() error
	ValidPartner() error
	CertRequired() bool
}

// GetProtoConfig parse and returns the given configuration according to the
// given protocol.
func GetProtoConfig(proto string, config json.RawMessage) (ProtoConfig, error) {
	cons, ok := ProtoConfigs[proto]
	if !ok {
		return nil, fmt.Errorf("unknown protocol '%s': %w", proto, errUnknownProtocol)
	}

	conf := cons()
	dec := json.NewDecoder(bytes.NewReader(config))
	dec.DisallowUnknownFields()

	if err := dec.Decode(conf); err != nil {
		return nil, fmt.Errorf("failed to parse protocol configuration: %w", err)
	}

	return conf, nil
}
