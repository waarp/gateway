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

func (i *InAgent) toLocal() *model.LocalAgent {
	return &model.LocalAgent{
		Name:        i.Name,
		Protocol:    i.Protocol,
		ProtoConfig: i.ProtoConfig,
	}
}

func (i *InAgent) toRemote() *model.RemoteAgent {
	return &model.RemoteAgent{
		Name:        i.Name,
		Protocol:    i.Protocol,
		ProtoConfig: i.ProtoConfig,
	}
}

// OutAgent is the JSON representation of a local/remote agent in responses
// sent by the REST interface.
type OutAgent struct {
	ID          uint64                 `json:"id"`
	Name        string                 `json:"name"`
	Protocol    string                 `json:"protocol"`
	ProtoConfig map[string]interface{} `json:"protoConfig"`
}

func fromLocalAgent(ag *model.LocalAgent) *OutAgent {
	var protoConfig map[string]interface{}
	_ = json.Unmarshal(ag.ProtoConfig, &protoConfig)
	return &OutAgent{
		ID:          ag.ID,
		Name:        ag.Name,
		Protocol:    ag.Protocol,
		ProtoConfig: protoConfig,
	}
}

func fromLocalAgents(ags []model.LocalAgent) []OutAgent {
	agents := make([]OutAgent, len(ags))
	for i, acc := range ags {
		agents[i] = OutAgent{
			ID:       acc.ID,
			Name:     acc.Name,
			Protocol: acc.Protocol,
		}
		_ = json.Unmarshal(acc.ProtoConfig, &agents[i].ProtoConfig)
	}
	return agents
}

func fromRemoteAgent(ag *model.RemoteAgent) *OutAgent {
	var protoConfig map[string]interface{}
	_ = json.Unmarshal(ag.ProtoConfig, &protoConfig)
	return &OutAgent{
		ID:          ag.ID,
		Name:        ag.Name,
		Protocol:    ag.Protocol,
		ProtoConfig: protoConfig,
	}
}

func fromRemoteAgents(ags []model.RemoteAgent) []OutAgent {
	agents := make([]OutAgent, len(ags))
	for i, acc := range ags {
		agents[i] = OutAgent{
			ID:       acc.ID,
			Name:     acc.Name,
			Protocol: acc.Protocol,
		}
		_ = json.Unmarshal(acc.ProtoConfig, &agents[i].ProtoConfig)
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
