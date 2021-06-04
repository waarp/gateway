package rest

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

	"code.waarp.fr/waarp-r66/r66"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

func newInLocAccount(old *model.LocalAccount) *api.InAccount {
	return &api.InAccount{
		Login:    &old.Login,
		Password: strPtr(string(old.PasswordHash)),
	}
}

func newInRemAccount(old *model.RemoteAccount) *api.InAccount {
	return &api.InAccount{
		Login:    &old.Login,
		Password: strPtr(string(old.Password)),
	}
}

// accToLocal transforms the JSON local account into its database equivalent.
func accToLocal(acc *api.InAccount, agent *model.LocalAgent, id uint64) (*model.LocalAccount, error) {
	if agent.Protocol == "r66" {
		// Unlike other protocols, when authenticating, an R66 client sends a
		// hash instead of a password, so we replace the password with its hash.
		acc.Password = strPtr(string(r66.CryptPass([]byte(str(acc.Password)))))
	}
	hash, err := utils.HashPassword(database.BcryptRounds, []byte(str(acc.Password)))
	if err != nil {
		return nil, fmt.Errorf("failed to hash password")
	}

	return &model.LocalAccount{
		ID:           id,
		LocalAgentID: agent.ID,
		Login:        str(acc.Login),
		PasswordHash: hash,
	}, nil
}

// accToRemote transforms the JSON remote account into its database equivalent.
func accToRemote(acc *api.InAccount, agent *model.RemoteAgent, id uint64) *model.RemoteAccount {
	return &model.RemoteAccount{
		ID:            id,
		RemoteAgentID: agent.ID,
		Login:         str(acc.Login),
		Password:      cStr(acc.Password),
	}
}

// FromLocalAccount transforms the given database local account into its JSON
// equivalent.
func FromLocalAccount(acc *model.LocalAccount, rules *api.AuthorizedRules) *api.OutAccount {
	return &api.OutAccount{
		Login:           acc.Login,
		AuthorizedRules: rules,
	}
}

// FromLocalAccounts transforms the given list of database local accounts into
// its JSON equivalent.
func FromLocalAccounts(accs []model.LocalAccount, rules []api.AuthorizedRules) []api.OutAccount {
	accounts := make([]api.OutAccount, len(accs))
	for i, acc := range accs {
		accounts[i] = api.OutAccount{
			Login:           acc.Login,
			AuthorizedRules: &rules[i],
		}
	}
	return accounts
}

// FromRemoteAccount transforms the given database remote account into its JSON
// equivalent.
func FromRemoteAccount(acc *model.RemoteAccount, rules *api.AuthorizedRules) *api.OutAccount {
	return &api.OutAccount{
		Login:           acc.Login,
		AuthorizedRules: rules,
	}
}

// FromRemoteAccounts transforms the given list of database remote accounts into
// its JSON equivalent.
func FromRemoteAccounts(accs []model.RemoteAccount, rules []api.AuthorizedRules) []api.OutAccount {
	accounts := make([]api.OutAccount, len(accs))
	for i, acc := range accs {
		accounts[i] = api.OutAccount{
			Login:           acc.Login,
			AuthorizedRules: &rules[i],
		}
	}
	return accounts
}
