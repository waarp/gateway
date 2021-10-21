package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func newInServer(old *model.LocalAgent) *api.InServer {
	return &api.InServer{
		Name:        &old.Name,
		Protocol:    &old.Protocol,
		Address:     &old.Address,
		Root:        &old.Root,
		LocalInDir:  &old.InDir,
		LocalOutDir: &old.OutDir,
		LocalTmpDir: &old.TmpDir,
		ProtoConfig: old.ProtoConfig,
	}
}

// servToDB transforms the JSON local agent into its database equivalent.
func servToDB(serv *api.InServer, id uint64, logger *log.Logger) *model.LocalAgent {
	in := str(serv.LocalInDir)
	out := str(serv.LocalOutDir)
	tmp := str(serv.LocalTmpDir)

	if serv.InDir != nil {
		logger.Warning("JSON field 'inDir' is deprecated, use 'serverLocalInDir' instead")

		in = str(serv.InDir)
	}

	if serv.OutDir != nil {
		logger.Warning("JSON field 'outDir' is deprecated, use 'serverLocalOutDir' instead")

		out = str(serv.OutDir)
	}

	if serv.WorkDir != nil {
		logger.Warning("JSON field 'workDir' is deprecated, use 'serverLocalTmpDir' instead")

		tmp = str(serv.WorkDir)
	}

	return &model.LocalAgent{
		ID:          id,
		Owner:       database.Owner,
		Name:        str(serv.Name),
		Address:     str(serv.Address),
		Root:        str(serv.Root),
		InDir:       in,
		OutDir:      out,
		TmpDir:      tmp,
		Protocol:    str(serv.Protocol),
		ProtoConfig: serv.ProtoConfig,
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
		Root:            ag.Root,
		InDir:           utils.NormalizePath(ag.InDir),
		OutDir:          utils.NormalizePath(ag.OutDir),
		WorkDir:         utils.NormalizePath(ag.TmpDir),
		LocalInDir:      ag.InDir,
		LocalOutDir:     ag.InDir,
		LocalTmpDir:     ag.TmpDir,
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
