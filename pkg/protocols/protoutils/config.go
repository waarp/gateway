package protoutils

import "code.waarp.fr/apps/gateway/gateway/pkg/utils"

type ProtoConfig interface {
	ValidConf() error
}

//nolint:wrapcheck //wrapping adds nothing here
func ValidateProtoConfig(conf map[string]any, target ProtoConfig) error {
	if err := utils.JSONConvert(conf, target); err != nil {
		return err
	}

	if err := target.ValidConf(); err != nil {
		return err
	}

	return utils.JSONConvert(target, &conf)
}
