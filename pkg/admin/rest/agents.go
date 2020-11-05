package rest

import (
	"fmt"
	"net/http"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/models"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/go-xorm/builder"
)

func newInServer(old *model.LocalAgent) *InServer {
	return &InServer{
		Name:        &old.Name,
		Protocol:    &old.Protocol,
		Address:     &old.Address,
		Root:        &old.Root,
		InDir:       &old.InDir,
		OutDir:      &old.OutDir,
		WorkDir:     &old.WorkDir,
		ProtoConfig: old.ProtoConfig,
	}
}

// servToDB transforms the JSON local agent into its database equivalent.
func servToDB(serv *InServer, id uint64) *model.LocalAgent {
	return &model.LocalAgent{
		ID:          id,
		Owner:       database.Owner,
		Name:        str(serv.Name),
		Address:     str(serv.Address),
		Root:        str(serv.Root),
		InDir:       str(serv.InDir),
		OutDir:      str(serv.OutDir),
		WorkDir:     str(serv.WorkDir),
		Protocol:    str(serv.Protocol),
		ProtoConfig: serv.ProtoConfig,
	}
}

func newInPartner(old *model.RemoteAgent) *InPartner {
	return &InPartner{
		Name:        &old.Name,
		Protocol:    &old.Protocol,
		Address:     &old.Address,
		ProtoConfig: old.ProtoConfig,
	}
}

// partToDB transforms the JSON remote agent into its database equivalent.
func partToDB(part *InPartner, id uint64) *model.RemoteAgent {
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
func FromLocalAgent(ag *model.LocalAgent, rules *AuthorizedRules) *OutServer {
	return &OutServer{
		Name:            ag.Name,
		Protocol:        ag.Protocol,
		Address:         ag.Address,
		Root:            ag.Root,
		InDir:           ag.InDir,
		OutDir:          ag.OutDir,
		WorkDir:         ag.WorkDir,
		ProtoConfig:     ag.ProtoConfig,
		AuthorizedRules: *rules,
	}
}

// FromLocalAgents transforms the given list of database local agents into
// its JSON equivalent.
func FromLocalAgents(ags []model.LocalAgent, rules []AuthorizedRules) []OutServer {
	agents := make([]OutServer, len(ags))
	for i, ag := range ags {
		agent := ag
		agents[i] = *FromLocalAgent(&agent, &rules[i])
	}
	return agents
}

// FromRemoteAgent transforms the given database remote agent into its JSON
// equivalent.
func FromRemoteAgent(ag *model.RemoteAgent, rules *AuthorizedRules) *OutPartner {
	return &OutPartner{
		Name:            ag.Name,
		Protocol:        ag.Protocol,
		Address:         ag.Address,
		ProtoConfig:     ag.ProtoConfig,
		AuthorizedRules: rules,
	}
}

// FromRemoteAgents transforms the given list of database remote agents into
// its JSON equivalent.
func FromRemoteAgents(ags []model.RemoteAgent, rules []AuthorizedRules) []OutPartner {
	agents := make([]OutPartner, len(ags))
	for i, ag := range ags {
		agent := ag
		agents[i] = *FromRemoteAgent(&agent, &rules[i])
	}
	return agents
}

func parseProtoParam(r *http.Request, filters *database.Filters) error {
	if len(r.Form["protocol"]) > 0 {
		protos := make([]string, len(r.Form["protocol"]))
		for i, p := range r.Form["protocol"] {
			if _, ok := config.ProtoConfigs[p]; !ok {
				return badRequest(fmt.Sprintf("'%s' is not a valid protocol", p))
			}
			protos[i] = p
		}
		filters.Conditions = builder.And(builder.In("protocol", protos), filters.Conditions)
	}
	return nil
}
