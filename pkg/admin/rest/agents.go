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

// InLocalAgent is the JSON representation of a local agent in requests
// made to the REST interface.
type InLocalAgent struct {
	Name        string          `json:"name"`
	Protocol    string          `json:"protocol"`
	Root        string          `json:"root"`
	ProtoConfig json.RawMessage `json:"protoConfig"`
}

// ToModel transforms the JSON local agent into its database equivalent.
func (i *InLocalAgent) ToModel() *model.LocalAgent {
	return &model.LocalAgent{
		Owner:       database.Owner,
		Name:        i.Name,
		Protocol:    i.Protocol,
		ProtoConfig: i.ProtoConfig,
	}
}

// InRemoteAgent is the JSON representation of a remote agent in requests
// made to the REST interface.
type InRemoteAgent struct {
	Name        string          `json:"name"`
	Protocol    string          `json:"protocol"`
	ProtoConfig json.RawMessage `json:"protoConfig"`
}

// ToModel transforms the JSON remote agent into its database equivalent.
func (i *InRemoteAgent) ToModel() *model.RemoteAgent {
	return &model.RemoteAgent{
		Name:        i.Name,
		Protocol:    i.Protocol,
		ProtoConfig: i.ProtoConfig,
	}
}

// OutServer is the JSON representation of a local server in responses sent by
// the REST interface.
type OutServer struct {
	Name            string          `json:"name"`
	Protocol        string          `json:"protocol"`
	Root            string          `json:"root"`
	ProtoConfig     json.RawMessage `json:"protoConfig"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`
}

// FromLocalAgent transforms the given database local agent into its JSON
// equivalent.
func FromLocalAgent(ag *model.LocalAgent, rules *AuthorizedRules) *OutServer {
	return &OutServer{
		Name:            ag.Name,
		Protocol:        ag.Protocol,
		Root:            ag.Root,
		ProtoConfig:     ag.ProtoConfig,
		AuthorizedRules: *rules,
	}
}

// FromLocalAgents transforms the given list of database local agents into
// its JSON equivalent.
func FromLocalAgents(ags []model.LocalAgent, rules []AuthorizedRules) []OutServer {
	agents := make([]OutServer, len(ags))
	for i, ag := range ags {
		agents[i] = OutServer{
			Name:            ag.Name,
			Protocol:        ag.Protocol,
			Root:            ag.Root,
			ProtoConfig:     ag.ProtoConfig,
			AuthorizedRules: rules[i],
		}
	}
	return agents
}

// OutPartner is the JSON representation of a remote partner in responses sent
// by the REST interface.
type OutPartner struct {
	Name            string          `json:"name"`
	Protocol        string          `json:"protocol"`
	ProtoConfig     json.RawMessage `json:"protoConfig"`
	AuthorizedRules AuthorizedRules `json:"authorizedRules"`
}

// FromRemoteAgent transforms the given database remote agent into its JSON
// equivalent.
func FromRemoteAgent(ag *model.RemoteAgent, rules *AuthorizedRules) *OutPartner {
	return &OutPartner{
		Name:            ag.Name,
		Protocol:        ag.Protocol,
		ProtoConfig:     ag.ProtoConfig,
		AuthorizedRules: *rules,
	}
}

// FromRemoteAgents transforms the given list of database remote agents into
// its JSON equivalent.
func FromRemoteAgents(ags []model.RemoteAgent, rules []AuthorizedRules) []OutPartner {
	agents := make([]OutPartner, len(ags))
	for i, ag := range ags {
		agents[i] = OutPartner{
			Name:            ag.Name,
			Protocol:        ag.Protocol,
			ProtoConfig:     ag.ProtoConfig,
			AuthorizedRules: rules[i],
		}
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
