package gwtesting

import (
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type (
	serverConstructor func(*database.DB, *model.LocalAgent) services.Server
	clientConstructor func(*database.DB, *model.Client) services.Client
)

var ErrUnknownProtocol = errors.New("unknown protocol")

type ProtoFeatures struct {
	MakeClient              clientConstructor
	MakeServer              serverConstructor
	MakeServerConfig        func() protocol.ServerConfig
	MakeClientConfig        func() protocol.ClientConfig
	MakePartnerConfig       func() protocol.PartnerConfig
	TransID, RuleName, Size bool
}

//nolint:gochecknoglobals //global var is required here for more flexibility
var Protocols = map[string]ProtoFeatures{}

//nolint:gochecknoinits //init is required here
func init() {
	model.ConfigChecker = configChecker{}
}

type configChecker struct{}

func (configChecker) IsValidProtocol(proto string) bool {
	_, ok := Protocols[proto]

	return ok
}

func (configChecker) CheckServerConfig(proto string, mapConf map[string]any) error {
	implementation, ok := Protocols[proto]
	if !ok {
		return fmt.Errorf("%w %q", ErrUnknownProtocol, proto)
	}

	structConf := implementation.MakeServerConfig()
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

func (configChecker) CheckClientConfig(proto string, mapConf map[string]any) error {
	implementation, ok := Protocols[proto]
	if !ok {
		return fmt.Errorf("%w %q", ErrUnknownProtocol, proto)
	}

	structConf := implementation.MakeClientConfig()
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

func (configChecker) CheckPartnerConfig(proto string, mapConf map[string]any) error {
	implementation, ok := Protocols[proto]
	if !ok {
		return fmt.Errorf("%w %q", ErrUnknownProtocol, proto)
	}

	structConf := implementation.MakePartnerConfig()
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
