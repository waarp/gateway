package rest

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// InAccount is the JSON representation of a local/remote account in requests
// made to the REST interface.
type InAccount struct {
	Login    string `json:"login"`
	Password []byte `json:"password"`
}

// ToLocal transforms the JSON local account into its database equivalent.
func (i *InAccount) ToLocal(agent *model.LocalAgent) *model.LocalAccount {
	return &model.LocalAccount{
		LocalAgentID: agent.ID,
		Login:        i.Login,
		Password:     i.Password,
	}
}

// ToRemote transforms the JSON remote account into its database equivalent.
func (i *InAccount) ToRemote(agent *model.RemoteAgent) *model.RemoteAccount {
	return &model.RemoteAccount{
		RemoteAgentID: agent.ID,
		Login:         i.Login,
		Password:      i.Password,
	}
}

// OutAccount is the JSON representation of a local/remote account in responses
// sent by the REST interface.
type OutAccount struct {
	Login string `json:"login"`
}

// FromLocalAccount transforms the given database local account into its JSON
// equivalent.
func FromLocalAccount(acc *model.LocalAccount) *OutAccount {
	return &OutAccount{
		Login: acc.Login,
	}
}

// FromLocalAccounts transforms the given list of database local accounts into
// its JSON equivalent.
func FromLocalAccounts(accs []model.LocalAccount) []OutAccount {
	accounts := make([]OutAccount, len(accs))
	for i, acc := range accs {
		accounts[i] = OutAccount{
			Login: acc.Login,
		}
	}
	return accounts
}

// FromRemoteAccount transforms the given database remote account into its JSON
// equivalent.
func FromRemoteAccount(acc *model.RemoteAccount) *OutAccount {
	return &OutAccount{
		Login: acc.Login,
	}
}

// FromRemoteAccounts transforms the given list of database remote accounts into
// its JSON equivalent.
func FromRemoteAccounts(accs []model.RemoteAccount) []OutAccount {
	accounts := make([]OutAccount, len(accs))
	for i, acc := range accs {
		accounts[i] = OutAccount{
			Login: acc.Login,
		}
	}
	return accounts
}
