package rest

import (
	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/models"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func newInLocAccount(old *model.LocalAccount) *InAccount {
	return &InAccount{
		Login:    &old.Login,
		Password: old.Password,
	}
}

func newInRemAccount(old *model.RemoteAccount) *InAccount {
	return &InAccount{
		Login:    &old.Login,
		Password: old.Password,
	}
}

// accToLocal transforms the JSON local account into its database equivalent.
func accToLocal(acc *InAccount, agent *model.LocalAgent, id uint64) *model.LocalAccount {
	return &model.LocalAccount{
		ID:           id,
		LocalAgentID: agent.ID,
		Login:        str(acc.Login),
		Password:     acc.Password,
	}
}

// accToRemote transforms the JSON remote account into its database equivalent.
func accToRemote(acc *InAccount, agent *model.RemoteAgent, id uint64) *model.RemoteAccount {
	return &model.RemoteAccount{
		ID:            id,
		RemoteAgentID: agent.ID,
		Login:         str(acc.Login),
		Password:      acc.Password,
	}
}

// FromLocalAccount transforms the given database local account into its JSON
// equivalent.
func FromLocalAccount(acc *model.LocalAccount, rules *AuthorizedRules) *OutAccount {
	return &OutAccount{
		Login:           acc.Login,
		AuthorizedRules: rules,
	}
}

// FromLocalAccounts transforms the given list of database local accounts into
// its JSON equivalent.
func FromLocalAccounts(accs []model.LocalAccount, rules []AuthorizedRules) []OutAccount {
	accounts := make([]OutAccount, len(accs))
	for i, acc := range accs {
		accounts[i] = OutAccount{
			Login:           acc.Login,
			AuthorizedRules: &rules[i],
		}
	}
	return accounts
}

// FromRemoteAccount transforms the given database remote account into its JSON
// equivalent.
func FromRemoteAccount(acc *model.RemoteAccount, rules *AuthorizedRules) *OutAccount {
	return &OutAccount{
		Login:           acc.Login,
		AuthorizedRules: rules,
	}
}

// FromRemoteAccounts transforms the given list of database remote accounts into
// its JSON equivalent.
func FromRemoteAccounts(accs []model.RemoteAccount, rules []AuthorizedRules) []OutAccount {
	accounts := make([]OutAccount, len(accs))
	for i, acc := range accs {
		accounts[i] = OutAccount{
			Login:           acc.Login,
			AuthorizedRules: &rules[i],
		}
	}
	return accounts
}
