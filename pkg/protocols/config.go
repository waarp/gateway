package protocols

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoinits //init is required here
func init() {
	model.ConfigChecker = configChecker{}
}

type configChecker struct{}

func (configChecker) IsValidProtocol(proto string) bool { return IsValid(proto) }

func (configChecker) CheckServerConfig(proto string, conf map[string]any) error {
	return CheckServerConfig(proto, conf)
}

func (configChecker) CheckClientConfig(proto string, conf map[string]any) error {
	return CheckClientConfig(proto, conf)
}

func (configChecker) CheckPartnerConfig(proto string, conf map[string]any) error {
	return CheckPartnerConfig(proto, conf)
}

func CheckServerConfig(proto string, mapConf map[string]any) error {
	implementation := Get(proto)
	if implementation == nil {
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

	utils.EmptyMap(mapConf)

	//nolint:wrapcheck //no need to wrap, this should never return an error anyway
	return utils.JSONConvert(structConf, &mapConf)
}

func CheckPartnerConfig(proto string, mapConf map[string]any) error {
	implementation := Get(proto)
	if implementation == nil {
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

	utils.EmptyMap(mapConf)

	//nolint:wrapcheck //no need to wrap, this should never return an error anyway
	return utils.JSONConvert(structConf, &mapConf)
}

func CheckClientConfig(proto string, mapConf map[string]any) error {
	implementation := Get(proto)
	if implementation == nil {
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

	utils.EmptyMap(mapConf)

	//nolint:wrapcheck //no need to wrap, this should never return an error anyway
	return utils.JSONConvert(structConf, &mapConf)
}
