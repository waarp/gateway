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

// OutLocalAgent is the JSON representation of a local agent in responses
// sent by the REST interface.
type OutLocalAgent struct {
	ID          uint64          `json:"id"`
	Name        string          `json:"name"`
	Protocol    string          `json:"protocol"`
	Root        string          `json:"root"`
	ProtoConfig json.RawMessage `json:"protoConfig"`
}

// FromLocalAgent transforms the given database local agent into its JSON
// equivalent.
func FromLocalAgent(ag *model.LocalAgent) *OutLocalAgent {
	return &OutLocalAgent{
		ID:          ag.ID,
		Name:        ag.Name,
		Protocol:    ag.Protocol,
		Root:        ag.Root,
		ProtoConfig: ag.ProtoConfig,
	}
}

// FromLocalAgents transforms the given list of database local agents into
// its JSON equivalent.
func FromLocalAgents(ags []model.LocalAgent) []OutLocalAgent {
	agents := make([]OutLocalAgent, len(ags))
	for i, ag := range ags {
		agents[i] = OutLocalAgent{
			ID:          ag.ID,
			Name:        ag.Name,
			Protocol:    ag.Protocol,
			Root:        ag.Root,
			ProtoConfig: ag.ProtoConfig,
		}
	}
	return agents
}

// OutRemoteAgent is the JSON representation of a remote agent in responses
// sent by the REST interface.
type OutRemoteAgent struct {
	ID          uint64          `json:"id"`
	Name        string          `json:"name"`
	Protocol    string          `json:"protocol"`
	ProtoConfig json.RawMessage `json:"protoConfig"`
}

// FromRemoteAgent transforms the given database remote agent into its JSON
// equivalent.
func FromRemoteAgent(ag *model.RemoteAgent) *OutRemoteAgent {
	return &OutRemoteAgent{
		ID:          ag.ID,
		Name:        ag.Name,
		Protocol:    ag.Protocol,
		ProtoConfig: ag.ProtoConfig,
	}
}

// FromRemoteAgents transforms the given list of database remote agents into
// its JSON equivalent.
func FromRemoteAgents(ags []model.RemoteAgent) []OutRemoteAgent {
	agents := make([]OutRemoteAgent, len(ags))
	for i, ag := range ags {
		agents[i] = OutRemoteAgent{
			ID:          ag.ID,
			Name:        ag.Name,
			Protocol:    ag.Protocol,
			ProtoConfig: ag.ProtoConfig,
		}
	}
	return agents
}

func parseProtoParam(r *http.Request, filters *database.Filters) error {
	if len(r.Form["protocol"]) > 0 {
		protos := make([]string, len(r.Form["protocol"]))
		for i, p := range r.Form["protocol"] {
			if _, ok := config.ProtoConfigs[p]; !ok {
				return &badRequest{msg: fmt.Sprintf("'%s' is not a valid protocol", p)}
			}
			protos[i] = p
		}
		filters.Conditions = builder.In("protocol", protos)
	}
	return nil
}
