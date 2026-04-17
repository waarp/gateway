package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrUnknownProtocol = errors.New("unknown protocol")

//nolint:gochecknoglobals //global var is needed here
var Protocols = map[string]Protocol{}

type Protocol interface {
	// CanMakeTransfer returns whether the protocol can be used to make the
	// given transfer.
	CanMakeTransfer(ctx *TransferContext) error
	// CheckServerConfig checks the configuration of the server associated with
	// the protocol.
	CheckServerConfig(conf map[string]any) error
	// CheckClientConfig checks the configuration of the client associated with
	// the protocol.
	CheckClientConfig(conf map[string]any) error
	// CheckPartnerConfig checks the configuration of the partner associated with
	// the protocol.
	CheckPartnerConfig(conf map[string]any) error
}

func IsValidProtocol(proto string) bool {
	_, ok := Protocols[proto]

	return ok
}

func CheckServerConfig(proto string, conf map[string]any) error {
	protocol := Protocols[proto]
	if protocol == nil {
		return fmt.Errorf("%w %q", ErrUnknownProtocol, proto)
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return protocol.CheckServerConfig(conf)
}

func CheckClientConfig(proto string, conf map[string]any) error {
	protocol := Protocols[proto]
	if protocol == nil {
		return fmt.Errorf("%w %q", ErrUnknownProtocol, proto)
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return protocol.CheckClientConfig(conf)
}

func CheckPartnerConfig(proto string, conf map[string]any) error {
	protocol := Protocols[proto]
	if protocol == nil {
		return fmt.Errorf("%w %q", ErrUnknownProtocol, proto)
	}

	//nolint:wrapcheck //wrapping adds nothing here
	return protocol.CheckPartnerConfig(conf)
}

const (
	protoR66    = "r66"
	protoR66TLS = "r66-tls"
)

func isR66(proto string) bool { return proto == protoR66 || proto == protoR66TLS }

type ProtoConfigMap map[string]any

func (p *ProtoConfigMap) Map() map[string]any { return *p }

//nolint:wrapcheck //wrapping adds nothing here
func (p *ProtoConfigMap) UnmarshalJSON(b []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()

	m := map[string]any{}

	if err := decoder.Decode(&m); err != nil {
		return err
	}

	*p = m

	return nil
}
