package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// servToDB transforms the JSON local agent into its database equivalent.
func servToDB(logger *log.Logger, input *api.InServer, output *model.LocalAgent) {
	setIfDefined(input.Name, &output.Name)
	setIfDefined(input.Protocol, &output.Protocol)
	setIfDefined(input.Address, &output.Address)
	setIfDefined(input.RootDir, &output.RootDir)
	setIfDefined(input.SendDir, &output.SendDir)
	setIfDefined(input.ReceiveDir, &output.ReceiveDir)
	setIfDefined(input.TmpReceiveDir, &output.TmpReceiveDir)

	if len(input.ProtoConfig) != 0 {
		output.ProtoConfig = input.ProtoConfig
	}

	if input.Root != nil {
		logger.Warning("JSON field 'root' is deprecated, use 'rootDir' instead")

		if output.RootDir == "" {
			output.RootDir = utils.DenormalizePath(str(input.Root))
		}
	}

	if input.InDir != nil {
		logger.Warning("JSON field 'inDir' is deprecated, use 'receiveDir' instead")

		if output.ReceiveDir == "" {
			output.ReceiveDir = utils.DenormalizePath(str(input.InDir))
		}
	}

	if input.OutDir != nil {
		logger.Warning("JSON field 'outDir' is deprecated, use 'sendDir' instead")

		if output.SendDir == "" {
			output.SendDir = utils.DenormalizePath(str(input.OutDir))
		}
	}

	if input.WorkDir != nil {
		logger.Warning("JSON field 'workDir' is deprecated, use 'tmpLocalRcvDir' instead")

		if output.TmpReceiveDir == "" {
			output.TmpReceiveDir = utils.DenormalizePath(str(input.WorkDir))
		}
	}
}

func newInPartner(old *model.RemoteAgent) *api.InPartner {
	return &api.InPartner{
		Name:        &old.Name,
		Protocol:    &old.Protocol,
		Address:     &old.Address,
		ProtoConfig: old.ProtoConfig,
	}
}

// partToDB transforms the JSON remote agent into its database equivalent.
func partToDB(part *api.InPartner, id uint64) *model.RemoteAgent {
	return &model.RemoteAgent{
		ID:          id,
		Name:        str(part.Name),
		Protocol:    str(part.Protocol),
		Address:     str(part.Address),
		ProtoConfig: part.ProtoConfig,
	}
}

// FromLocalAgent transforms the given database local agent into its JSON
// equivalent.
func FromLocalAgent(ag *model.LocalAgent, rules *api.AuthorizedRules) *api.OutServer {
	if ag.Protocol == config.ProtocolR66TLS {
		var r66Conf *config.R66ProtoConfig
		if json.Unmarshal(ag.ProtoConfig, r66Conf) == nil && r66Conf.IsTLS != nil {
			// To preserve backwards compatibility, when `ìsTLS` is defined, we
			// change the protocol back to config.ProtocolR66, like it was before the addition
			// of the config.ProtocolR66TLS protocol.
			ag.Protocol = config.ProtocolR66
		}
	}

	return &api.OutServer{
		Name:            ag.Name,
		Protocol:        ag.Protocol,
		Address:         ag.Address,
		Enabled:         ag.Enabled,
		Root:            utils.NormalizePath(ag.RootDir),
		RootDir:         ag.RootDir,
		InDir:           utils.NormalizePath(ag.ReceiveDir),
		OutDir:          utils.NormalizePath(ag.SendDir),
		WorkDir:         utils.NormalizePath(ag.TmpReceiveDir),
		SendDir:         ag.SendDir,
		ReceiveDir:      ag.ReceiveDir,
		TmpReceiveDir:   ag.TmpReceiveDir,
		ProtoConfig:     ag.ProtoConfig,
		AuthorizedRules: *rules,
	}
}

// FromLocalAgents transforms the given list of database local agents into
// its JSON equivalent.
func FromLocalAgents(ags []model.LocalAgent, rules []api.AuthorizedRules) []api.OutServer {
	agents := make([]api.OutServer, len(ags))

	for i := range ags {
		agent := &ags[i]
		agents[i] = *FromLocalAgent(agent, &rules[i])
	}

	return agents
}

// FromRemoteAgent transforms the given database remote agent into its JSON
// equivalent.
func FromRemoteAgent(ag *model.RemoteAgent, rules *api.AuthorizedRules) *api.OutPartner {
	if ag.Protocol == config.ProtocolR66TLS {
		var r66Conf *config.R66ProtoConfig
		if json.Unmarshal(ag.ProtoConfig, r66Conf) == nil && r66Conf.IsTLS != nil {
			// To preserve backwards compatibility, when `ìsTLS` is defined, we
			// change the protocol back to config.ProtocolR66, like it was before the addition
			// of the config.ProtocolR66TLS protocol.
			ag.Protocol = config.ProtocolR66
		}
	}

	return &api.OutPartner{
		Name:            ag.Name,
		Protocol:        ag.Protocol,
		Address:         ag.Address,
		ProtoConfig:     ag.ProtoConfig,
		AuthorizedRules: rules,
	}
}

// FromRemoteAgents transforms the given list of database remote agents into
// its JSON equivalent.
func FromRemoteAgents(ags []model.RemoteAgent, rules []api.AuthorizedRules) []api.OutPartner {
	agents := make([]api.OutPartner, len(ags))

	for i := range ags {
		agent := &ags[i]
		agents[i] = *FromRemoteAgent(agent, &rules[i])
	}

	return agents
}

func parseProtoParam(r *http.Request, query *database.SelectQuery) error {
	if len(r.Form["protocol"]) > 0 {
		protos := make([]string, len(r.Form["protocol"]))

		for i, p := range r.Form["protocol"] {
			if _, ok := config.ProtoConfigs[p]; !ok {
				return badRequest(fmt.Sprintf("'%s' is not a valid protocol", p))
			}

			protos[i] = p
		}

		query.In("protocol", protos)
	}

	return nil
}
