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
//
//nolint:gochecknoglobals // global var is used by design
var ProtoConfigs = map[string]*ConfigMaker{}

func IsValidProtocol(proto string) bool {
	_, ok := ProtoConfigs[proto]

	return ok
}

type ConfigMaker struct {
	Server  func() ServerProtoConfig
	Partner func() PartnerProtoConfig
	Client  func() ClientProtoConfig
}

type (
	ServerProtoConfig  interface{ ValidServer() error }
	PartnerProtoConfig interface{ ValidPartner() error }
	ClientProtoConfig  interface{ ValidClient() error }
)

func initParser(proto string, rawConf json.RawMessage) (*json.Decoder, *ConfigMaker, error) {
	constr, ok := ProtoConfigs[proto]
	if !ok {
		return nil, nil, fmt.Errorf("%w %q", errUnknownProtocol, proto)
	}

	dec := json.NewDecoder(bytes.NewReader(rawConf))
	dec.DisallowUnknownFields()

	return dec, constr, nil
}

func ParseServerConfig(proto string, rawConf json.RawMessage) (ServerProtoConfig, error) {
	dec, constr, initErr := initParser(proto, rawConf)
	if initErr != nil {
		return nil, initErr
	}

	conf := constr.Server()
	if err := dec.Decode(conf); err != nil {
		return nil, fmt.Errorf("failed to parse the server protocol configuration: %w", err)
	}

	return conf, nil
}

func ParsePartnerConfig(proto string, rawConf json.RawMessage) (PartnerProtoConfig, error) {
	dec, constr, initErr := initParser(proto, rawConf)
	if initErr != nil {
		return nil, initErr
	}

	conf := constr.Partner()
	if err := dec.Decode(conf); err != nil {
		return nil, fmt.Errorf("failed to parse the partner protocol configuration: %w", err)
	}

	return conf, nil
}

func ParseClientConfig(proto string, rawConf json.RawMessage) (ClientProtoConfig, error) {
	dec, constr, initErr := initParser(proto, rawConf)
	if initErr != nil {
		return nil, initErr
	}

	conf := constr.Client()
	if err := dec.Decode(conf); err != nil {
		return nil, fmt.Errorf("failed to parse the client protocol configuration: %w", err)
	}

	return conf, nil
}
