package rest

import (
	"fmt"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/config"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func dbLocalAccountToRESTInput(old *model.LocalAccount) *api.InAccount {
	return &api.InAccount{
		Login:    &old.Login,
		Password: strPtr(old.PasswordHash),
	}
}

func dbRemoteAccountToRESTInput(old *model.RemoteAccount) *api.InAccount {
	return &api.InAccount{
		Login:    &old.Login,
		Password: strPtr(string(old.Password)),
	}
}

// restLocalAccountToDB transforms the JSON local account into its database equivalent.
func restLocalAccountToDB(restAccount *api.InAccount, parent *model.LocalAgent,
) (*model.LocalAccount, error) {
	if parent.Protocol == config.ProtocolR66 || parent.Protocol == config.ProtocolR66TLS {
		// Unlike other protocols, when authenticating, an R66 client sends a
		// hash instead of a password, so we replace the password with its hash.
		restAccount.Password = strPtr(string(r66.CryptPass([]byte(str(restAccount.Password)))))
	}

	hash, err := utils.HashPassword(database.BcryptRounds, str(restAccount.Password))
	if err != nil {
		return nil, fmt.Errorf("failed to hash passwordi: %w", err)
	}

	return &model.LocalAccount{
		LocalAgentID: parent.ID,
		Login:        str(restAccount.Login),
		PasswordHash: hash,
	}, nil
}

// restRemoteAccountToDB transforms the JSON remote account into its database equivalent.
func restRemoteAccountToDB(restAccount *api.InAccount, parent *model.RemoteAgent,
) *model.RemoteAccount {
	return &model.RemoteAccount{
		RemoteAgentID: parent.ID,
		Login:         str(restAccount.Login),
		Password:      cStr(restAccount.Password),
	}
}

// DBLocalAccountToREST transforms the given database local account into its JSON
// equivalent.
func DBLocalAccountToREST(db database.ReadAccess, dbAccount *model.LocalAccount,
) (*api.OutAccount, error) {
	authorizedRules, err := getAuthorizedRules(db, dbAccount)
	if err != nil {
		return nil, err
	}

	return &api.OutAccount{
		Login:           dbAccount.Login,
		AuthorizedRules: authorizedRules,
	}, nil
}

// DBLocalAccountsToRest transforms the given list of database local accounts into
// its JSON equivalent.
func DBLocalAccountsToRest(db database.ReadAccess, dbAccounts []*model.LocalAccount,
) ([]*api.OutAccount, error) {
	restAccounts := make([]*api.OutAccount, len(dbAccounts))

	for i, acc := range dbAccounts {
		var err error
		if restAccounts[i], err = DBLocalAccountToREST(db, acc); err != nil {
			return nil, err
		}
	}

	return restAccounts, nil
}

// DBRemoteAccountToREST transforms the given database remote account into its JSON
// equivalent.
func DBRemoteAccountToREST(db database.ReadAccess, dbAccount *model.RemoteAccount,
) (*api.OutAccount, error) {
	authorizedRules, err := getAuthorizedRules(db, dbAccount)
	if err != nil {
		return nil, err
	}

	return &api.OutAccount{
		Login:           dbAccount.Login,
		AuthorizedRules: authorizedRules,
	}, nil
}

// DBRemoteAccountsToREST transforms the given list of database remote accounts into
// its JSON equivalent.
func DBRemoteAccountsToREST(db database.ReadAccess, dbAccounts []*model.RemoteAccount,
) ([]*api.OutAccount, error) {
	restAccounts := make([]*api.OutAccount, len(dbAccounts))

	for i, acc := range dbAccounts {
		var err error
		if restAccounts[i], err = DBRemoteAccountToREST(db, acc); err != nil {
			return nil, err
		}
	}

	return restAccounts, nil
}
