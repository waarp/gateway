package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

// InAgent is the JSON representation of a local/remote agent in requests
// made to the REST interface.
type InAgent struct {
	Name        string          `json:"name"`
	Protocol    string          `json:"protocol"`
	ProtoConfig json.RawMessage `json:"protoConfig"`
}

// ToLocal transforms the JSON local agent into its database equivalent.
func (i *InAgent) ToLocal() *model.LocalAgent {
	return &model.LocalAgent{
		Name:        i.Name,
		Protocol:    i.Protocol,
		ProtoConfig: i.ProtoConfig,
	}
}

// ToRemote transforms the JSON remote agent into its database equivalent.
func (i *InAgent) ToRemote() *model.RemoteAgent {
	return &model.RemoteAgent{
		Name:        i.Name,
		Protocol:    i.Protocol,
		ProtoConfig: i.ProtoConfig,
	}
}

// OutAgent is the JSON representation of a local/remote agent in responses
// sent by the REST interface.
type OutAgent struct {
	ID          uint64          `json:"id"`
	Name        string          `json:"name"`
	Protocol    string          `json:"protocol"`
	ProtoConfig json.RawMessage `json:"protoConfig"`
}

// FromLocalAgent transforms the given database local agent into its JSON
// equivalent.
func FromLocalAgent(ag *model.LocalAgent) *OutAgent {
	return &OutAgent{
		ID:          ag.ID,
		Name:        ag.Name,
		Protocol:    ag.Protocol,
		ProtoConfig: ag.ProtoConfig,
	}
}

// FromLocalAgents transforms the given list of database local agents into
// its JSON equivalent.
func fromLocalAgents(ags []model.LocalAgent) []OutAgent {
	agents := make([]OutAgent, len(ags))
	for i, ag := range ags {
		agents[i] = OutAgent{
			ID:          ag.ID,
			Name:        ag.Name,
			Protocol:    ag.Protocol,
			ProtoConfig: ag.ProtoConfig,
		}
	}
	return agents
}

// FromRemoteAgent transforms the given database remote agent into its JSON
// equivalent.
func fromRemoteAgent(ag *model.RemoteAgent) *OutAgent {
	return &OutAgent{
		ID:          ag.ID,
		Name:        ag.Name,
		Protocol:    ag.Protocol,
		ProtoConfig: ag.ProtoConfig,
	}
}

// FromRemoteAgents transforms the given list of database remote agents into
// its JSON equivalent.
func fromRemoteAgents(ags []model.RemoteAgent) []OutAgent {
	agents := make([]OutAgent, len(ags))
	for i, ag := range ags {
		agents[i] = OutAgent{
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
			if !model.IsValidProtocol(p) {
				return &badRequest{msg: fmt.Sprintf("'%s' is not a valid protocol", p)}
			}
			protos[i] = p
		}
		filters.Conditions = builder.In("protocol", protos)
	}
	return nil
}
