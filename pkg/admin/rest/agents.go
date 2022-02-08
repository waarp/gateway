package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func newInServer(old *model.LocalAgent) *api.InServer {
	return &api.InServer{
		Name:          &old.Name,
		Protocol:      &old.Protocol,
		Address:       &old.Address,
		RootDir:       &old.RootDir,
		ReceiveDir:    &old.ReceiveDir,
		SendDir:       &old.SendDir,
		TmpReceiveDir: &old.TmpReceiveDir,
		ProtoConfig:   old.ProtoConfig,
	}
}

// servToDB transforms the JSON local agent into its database equivalent.
func servToDB(serv *api.InServer, serverID uint64, logger *log.Logger) *model.LocalAgent {
	root := str(serv.RootDir)
	sndDir := str(serv.SendDir)
	rcvDir := str(serv.ReceiveDir)
	tmpDir := str(serv.TmpReceiveDir)

	if serv.Root != nil {
		logger.Warning("JSON field 'root' is deprecated, use 'rootDir' instead")

		if root == "" {
			root = utils.DenormalizePath(str(serv.Root))
		}
	}

	if serv.InDir != nil {
		logger.Warning("JSON field 'inDir' is deprecated, use 'receiveDir' instead")

		if rcvDir == "" {
			rcvDir = utils.DenormalizePath(str(serv.InDir))
		}
	}

	if serv.OutDir != nil {
		logger.Warning("JSON field 'outDir' is deprecated, use 'sendDir' instead")

		if sndDir == "" {
			sndDir = utils.DenormalizePath(str(serv.OutDir))
		}
	}

	if serv.WorkDir != nil {
		logger.Warning("JSON field 'workDir' is deprecated, use 'tmpLocalRcvDir' instead")

		if tmpDir == "" {
			tmpDir = utils.DenormalizePath(str(serv.WorkDir))
		}
	}

	return &model.LocalAgent{
		ID:            serverID,
		Owner:         conf.GlobalConfig.GatewayName,
		Name:          str(serv.Name),
		Address:       str(serv.Address),
		RootDir:       root,
		ReceiveDir:    rcvDir,
		SendDir:       sndDir,
		TmpReceiveDir: tmpDir,
		Protocol:      str(serv.Protocol),
		ProtoConfig:   serv.ProtoConfig,
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
	return &api.OutServer{
		Name:            ag.Name,
		Protocol:        ag.Protocol,
		Address:         ag.Address,
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
