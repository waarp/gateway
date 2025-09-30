package internal

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrUnknownProtocol = errors.New("unknown protocol")

func ValidProtocols() []string {
	return slices.Collect(maps.Keys(protocols.List))
}

func GetServerConfig(server *model.LocalAgent) (protocol.ServerConfig, error) {
	module := protocols.Get(server.Protocol)
	if module == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProtocol, server.Protocol)
	}

	conf := module.MakeServerConfig()

	return conf, utils.JSONConvert(server.ProtoConfig, conf)
}

func GetClientConfig(client *model.Client) (protocol.ClientConfig, error) {
	module := protocols.Get(client.Protocol)
	if module == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProtocol, client.Protocol)
	}

	conf := module.MakeClientConfig()

	return conf, utils.JSONConvert(client.ProtoConfig, conf)
}

func GetPartnerConfig(partner *model.RemoteAgent) (protocol.PartnerConfig, error) {
	module := protocols.Get(partner.Protocol)
	if module == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownProtocol, partner.Protocol)
	}

	conf := module.MakePartnerConfig()

	return conf, utils.JSONConvert(partner.ProtoConfig, conf)
}
