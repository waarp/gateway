// Package config contains the stucts representing the different kinds of
// protocol configuration.
package config

import (
	"encoding/json"
	"fmt"
)

// ProtoConfig is the interface implemented by protocol configuration structs.
// It exposes 2 methods needed for validating the configuration.
type ProtoConfig interface {
	ValidServer() error
	ValidClient() error
}

// GetProtoConfig parse and returns the given configuration according to the
// given protocol.
func GetProtoConfig(proto string, config []byte) (ProtoConfig, error) {
	switch proto {
	case "sftp":
		conf := &SftpProtoConfig{}
		err := json.Unmarshal(config, conf)
		return conf, err
	default:
		return nil, fmt.Errorf("unknown protocol")
	}
}
