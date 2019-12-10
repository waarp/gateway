package rest

import (
	"fmt"
	"net/http"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

// InAccount is the JSON representation of a local/remote account in requests
// made to the REST interface.
type InAccount struct {
	AgentID  uint64 `json:"agentID"`
	Login    string `json:"login"`
	Password []byte `json:"password"`
}

func (i *InAccount) toLocal() *model.LocalAccount {
	return &model.LocalAccount{
		LocalAgentID: i.AgentID,
		Login:        i.Login,
		Password:     i.Password,
	}
}

func (i *InAccount) toRemote() *model.RemoteAccount {
	return &model.RemoteAccount{
		RemoteAgentID: i.AgentID,
		Login:         i.Login,
		Password:      i.Password,
	}
}

// OutAccount is the JSON representation of a local/remote account in responses
// sent by the REST interface.
type OutAccount struct {
	ID      uint64 `json:"id"`
	AgentID uint64 `json:"agentID"`
	Login   string `json:"login"`
}

func fromLocal(acc *model.LocalAccount) *OutAccount {
	return &OutAccount{
		ID:      acc.ID,
		AgentID: acc.LocalAgentID,
		Login:   acc.Login,
	}
}

func fromLocalArray(accs []model.LocalAccount) []OutAccount {
	accounts := make([]OutAccount, len(accs))
	for i, acc := range accs {
		accounts[i] = OutAccount{
			ID:      acc.ID,
			AgentID: acc.LocalAgentID,
			Login:   acc.Login,
		}
	}
	return accounts
}

func fromRemote(acc *model.RemoteAccount) *OutAccount {
	return &OutAccount{
		ID:      acc.ID,
		AgentID: acc.RemoteAgentID,
		Login:   acc.Login,
	}
}

func fromRemoteArray(accs []model.RemoteAccount) []OutAccount {
	accounts := make([]OutAccount, len(accs))
	for i, acc := range accs {
		accounts[i] = OutAccount{
			ID:      acc.ID,
			AgentID: acc.RemoteAgentID,
			Login:   acc.Login,
		}
	}
	return accounts
}

func parseAgentParam(r *http.Request, filters *database.Filters, col string) error {
	if len(r.Form["agent"]) > 0 {
		ids := make([]uint64, len(r.Form["agent"]))
		for i, agent := range r.Form["agent"] {
			id, err := strconv.ParseUint(agent, 10, 64)
			if err != nil {
				return &badRequest{fmt.Sprintf("'%s' is not a valid agent ID", agent)}
			}
			ids[i] = id
		}
		filters.Conditions = builder.In(col, ids)
	}
	return nil
}
