// Package config contains the stucts representing the different kinds of
// protocol configuration.
package config

import (
	"encoding/json"
	"fmt"
)

// ProtoConfigs is a map associating each transfer protocol with their respective
// struct constructor.
var ProtoConfigs = map[string]func() ProtoConfig{}

// ProtoConfig is the interface implemented by protocol configuration structs.
// It exposes 2 methods needed for validating the configuration.
type ProtoConfig interface {
	ValidServer() error
	ValidPartner() error
}

// GetProtoConfig parse and returns the given configuration according to the
// given protocol.
func GetProtoConfig(proto string, config json.RawMessage) (ProtoConfig, error) {
	cons, ok := ProtoConfigs[proto]
	if !ok {
		return nil, fmt.Errorf("unknown protocol")
	}
	conf := cons()
	if err := json.Unmarshal(config, conf); err != nil {
		return nil, fmt.Errorf("failed to parse protocol configuration: %s", err.Error())
	}
	return conf, nil
}
