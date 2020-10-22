package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"github.com/go-xorm/builder"
)

// InServer is the JSON representation of a local agent in requests
// made to the REST interface.
type InServer struct {
	Name        *string         `json:"name,omitempty"`
	Protocol    *string         `json:"protocol,omitempty"`
	Address     *string         `json:"address,omitempty"`
	Root        *string         `json:"root,omitempty"`
	InDir       *string         `json:"inDir,omitempty"`
	OutDir      *string         `json:"outDir,omitempty"`
	WorkDir     *string         `json:"workDir,omitempty"`
	ProtoConfig json.RawMessage `json:"protoConfig,omitempty"`
}

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

// ToModel transforms the JSON local agent into its database equivalent.
func (i *InServer) ToModel(id uint64) *model.LocalAgent {
	return &model.LocalAgent{
		ID:          id,
		Owner:       database.Owner,
		Name:        str(i.Name),
		Address:     str(i.Address),
		Root:        str(i.Root),
		InDir:       str(i.InDir),
		OutDir:      str(i.OutDir),
		WorkDir:     str(i.WorkDir),
		Protocol:    str(i.Protocol),
		ProtoConfig: i.ProtoConfig,
	}
}

// InPartner is the JSON representation of a remote agent in requests
// made to the REST interface.
type InPartner struct {
	Name        *string         `json:"name,omitempty"`
	Protocol    *string         `json:"protocol,omitempty"`
	Address     *string         `json:"address,omitempty"`
	ProtoConfig json.RawMessage `json:"protoConfig,omitempty"`
}

func newInPartner(old *model.RemoteAgent) *InPartner {
	return &InPartner{
		Name:        &old.Name,
		Protocol:    &old.Protocol,
		Address:     &old.Address,
		ProtoConfig: old.ProtoConfig,
	}
}

// ToModel transforms the JSON remote agent into its database equivalent.
func (i *InPartner) ToModel(id uint64) *model.RemoteAgent {
	return &model.RemoteAgent{
		ID:          id,
		Name:        str(i.Name),
		Protocol:    str(i.Protocol),
		Address:     str(i.Address),
		ProtoConfig: i.ProtoConfig,
	}
}

// OutServer is the JSON representation of a local server in responses sent by
// the REST interface.
type OutServer struct {
	Name            string          `json:"name"`
	Protocol        string          `json:"protocol"`
	Address         string          `json:"address"`
	Root            string          `json:"root,omitempty"`
	InDir           string          `json:"inDir,omitempty"`
	OutDir          string          `json:"outDir,omitempty"`
	WorkDir         string          `json:"workDir,omitempty"`
	ProtoConfig     json.RawMessage `json:"protoConfig"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`
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

// OutPartner is the JSON representation of a remote partner in responses sent
// by the REST interface.
type OutPartner struct {
	Name            string           `json:"name"`
	Protocol        string           `json:"protocol"`
	Address         string           `json:"address"`
	ProtoConfig     json.RawMessage  `json:"protoConfig"`
	AuthorizedRules *AuthorizedRules `json:"authorizedRules,omitempty"`
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
