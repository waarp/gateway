package rest

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
)

// DBLocalAccountToREST transforms the given database local account into its JSON
// equivalent.
func DBLocalAccountToREST(db database.ReadAccess, dbAccount *model.LocalAccount,
) (*api.OutAccount, error) {
	authorizedRules, err := getAuthorizedRules(db, dbAccount)
	if err != nil {
		return nil, err
	}

	credentials, err := makeCredList(db, dbAccount)
	if err != nil {
		return nil, err
	}

	return &api.OutAccount{
		Login:           dbAccount.Login,
		Credentials:     credentials,
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

	credentials, err := makeCredList(db, dbAccount)
	if err != nil {
		return nil, err
	}

	return &api.OutAccount{
		Login:           dbAccount.Login,
		Credentials:     credentials,
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

func updateAccountPassword(ses *database.Session, account model.CredOwnerTable, password string,
) error {
	var cred model.Credential
	if err := ses.Get(&cred, "type=?", auth.Password).And(
		account.GetCredCond()).Run(); database.IsNotFound(err) {
		cred.Type = auth.Password
		cred.Value = password
		account.SetCredOwner(&cred)

		if err2 := ses.Insert(&cred).Run(); err2 != nil {
			return fmt.Errorf("failed to insert account password: %w", err2)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("failed to retrieve old account password: %w", err)
	}

	if password == "" {
		if err := ses.Delete(&cred).Run(); err != nil {
			return fmt.Errorf("failed to delete old account password: %w", err)
		}
	}

	cred.Value = password

	if err := ses.Update(&cred).Run(); err != nil {
		return fmt.Errorf("failed to update account password: %w", err)
	}

	return nil
}
