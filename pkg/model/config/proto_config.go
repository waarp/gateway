// Package config contains the stucts representing the different kinds of
// protocol configuration.
package config

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

var (
	errInvalidProtoConfig = errors.New("the protocol configuration is invalid")
	errUnknownProtocol    = errors.New("unknown protocol")
)

// ProtoConfigs is a map associating each transfer protocol with their respective
// struct constructor.
//
//nolint:gochecknoglobals // global var is used by design
var ProtoConfigs = map[string]*Constructor{}

func IsValidProtocol(proto string) bool {
	_, ok := ProtoConfigs[proto]

	return ok
}

type Constructor struct {
	Server  func() ServerProtoConfig
	Partner func() PartnerProtoConfig
	Client  func() ClientProtoConfig
}

type (
	ServerProtoConfig  interface{ ValidServer() error }
	PartnerProtoConfig interface{ ValidPartner() error }
	ClientProtoConfig  interface{ ValidClient() error }
)

func CheckServerConfig(proto string, mapConf map[string]any) error {
	constr, ok := ProtoConfigs[proto]
	if !ok {
		return fmt.Errorf("%w %q", errUnknownProtocol, proto)
	}

	structConf := constr.Server()
	if err := utils.JSONConvert(mapConf, structConf); err != nil {
		return fmt.Errorf("invalid proto config: %w", err)
	}

	//nolint:wrapcheck //wrapping this error would add nothing
	if err := structConf.ValidServer(); err != nil {
		return err
	}

	//nolint:wrapcheck //no need to wrap, this should never return an error anyway
	return utils.JSONConvert(structConf, &mapConf)
}

func CheckPartnerConfig(proto string, mapConf map[string]any) error {
	constr, ok := ProtoConfigs[proto]
	if !ok {
		return fmt.Errorf("%w %q", errUnknownProtocol, proto)
	}

	structConf := constr.Partner()
	if err := utils.JSONConvert(mapConf, structConf); err != nil {
		return fmt.Errorf("invalid proto config: %w", err)
	}

	//nolint:wrapcheck //wrapping this error would add nothing
	if err := structConf.ValidPartner(); err != nil {
		return err
	}

	//nolint:wrapcheck //no need to wrap, this should never return an error anyway
	return utils.JSONConvert(structConf, &mapConf)
}

func CheckClientConfig(proto string, mapConf map[string]any) error {
	constr, ok := ProtoConfigs[proto]
	if !ok {
		return fmt.Errorf("%w %q", errUnknownProtocol, proto)
	}

	structConf := constr.Client()
	if err := utils.JSONConvert(mapConf, structConf); err != nil {
		return fmt.Errorf("invalid proto config: %w", err)
	}

	//nolint:wrapcheck //wrapping this error would add nothing
	if err := structConf.ValidClient(); err != nil {
		return err
	}

	//nolint:wrapcheck //no need to wrap, this should never return an error anyway
	return utils.JSONConvert(structConf, &mapConf)
}
