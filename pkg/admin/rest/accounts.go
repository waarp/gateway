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

// ToLocal transforms the JSON local account into its database equivalent.
func (i *InAccount) ToLocal() *model.LocalAccount {
	return &model.LocalAccount{
		LocalAgentID: i.AgentID,
		Login:        i.Login,
		Password:     i.Password,
	}
}

// ToRemote transforms the JSON remote account into its database equivalent.
func (i *InAccount) ToRemote() *model.RemoteAccount {
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

// FromLocalAccount transforms the given database local account into its JSON
// equivalent.
func FromLocalAccount(acc *model.LocalAccount) *OutAccount {
	return &OutAccount{
		ID:      acc.ID,
		AgentID: acc.LocalAgentID,
		Login:   acc.Login,
	}
}

// FromLocalAccounts transforms the given list of database local accounts into
// its JSON equivalent.
func FromLocalAccounts(accs []model.LocalAccount) []OutAccount {
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

// FromRemoteAccount transforms the given database remote account into its JSON
// equivalent.
func FromRemoteAccount(acc *model.RemoteAccount) *OutAccount {
	return &OutAccount{
		ID:      acc.ID,
		AgentID: acc.RemoteAgentID,
		Login:   acc.Login,
	}
}

// FromRemoteAccounts transforms the given list of database remote accounts into
// its JSON equivalent.
func FromRemoteAccounts(accs []model.RemoteAccount) []OutAccount {
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
