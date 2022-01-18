package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// RemoteAccount represents an account on a remote agent. It is used by the
// gateway to authenticate on distant servers for transfers.
type RemoteAccount struct {
	ID            int64 `xorm:"<- id AUTOINCR"`  // The account's database ID
	RemoteAgentID int64 `xorm:"remote_agent_id"` // The ID of the RemoteAgent this account is attached to

	Login string `xorm:"login"` // The account's login
}

func (*RemoteAccount) TableName() string   { return TableRemAccounts }
func (*RemoteAccount) Appellation() string { return NameRemoteAccount }
func (r *RemoteAccount) GetID() int64      { return r.ID }
func (r *RemoteAccount) Host() string      { return "" }
func (*RemoteAccount) IsServer() bool      { return false }

// BeforeWrite checks if the new `RemoteAccount` entry is valid and can be
// inserted in the database.
//
//nolint:dupl // too many differences to be factorized easily
func (r *RemoteAccount) BeforeWrite(db database.ReadAccess) error {
	if r.RemoteAgentID == 0 {
		return database.NewValidationError("the account's agentID cannot be empty")
	}

	if r.Login == "" {
		return database.NewValidationError("the account's login cannot be empty")
	}

	if n, err := db.Count(&RemoteAgent{}).Where("id=?", r.RemoteAgentID).Run(); err != nil {
		return fmt.Errorf("failed to check parent remote agent: %w", err)
	} else if n == 0 {
		return database.NewValidationError(`no remote agent found with the ID "%v"`,
			r.RemoteAgentID)
	}

	if n, err := db.Count(&RemoteAccount{}).Where("id<>? AND remote_agent_id=? AND login=?",
		r.ID, r.RemoteAgentID, r.Login).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate remote accounts: %w", err)
	} else if n > 0 {
		return database.NewValidationError(
			"a remote account with the same login %q already exist", r.Login)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to check whether the account is still used in any ongoing transfer.
func (r *RemoteAccount) BeforeDelete(db database.Access) error {
	if n, err := db.Count(&Transfer{}).Where("remote_account_id=?", r.ID).Run(); err != nil {
		return fmt.Errorf("failed to check for ongoing transfers: %w", err)
	} else if n > 0 {
		return database.NewValidationError("this account is currently being used " +
			"in one or more running transfers and thus cannot be deleted, cancel " +
			"these transfers or wait for them to finish")
	}

	return nil
}

// GetCredentials fetch in the database then return the associated Credentials if they exist.
func (r *RemoteAccount) GetCredentials(db database.ReadAccess, authTypes ...string,
) (Credentials, error) {
	return getCredentials(db, r, authTypes...)
}

//nolint:goconst //duplicates are for different tables, best not to factorize
func (r *RemoteAccount) GetCredCond() (string, int64)         { return "remote_account_id=?", r.ID }
func (r *RemoteAccount) SetCredOwner(a *Credential)           { a.RemoteAccountID = utils.NewNullInt64(r.ID) }
func (r *RemoteAccount) GenAccessSelectCond() (string, int64) { return "remote_account_id=?", r.ID }
func (r *RemoteAccount) SetAccessTarget(a *RuleAccess)        { a.RemoteAccountID = utils.NewNullInt64(r.ID) }

func (r *RemoteAccount) GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error) {
	var rules Rules
	if err := db.Select(&rules).Where(fmt.Sprintf(
		`id IN (SELECT DISTINCT rule_id FROM %s WHERE remote_agent_id=? OR 
			remote_account_id=?)
		  OR (SELECT COUNT(*) FROM %s WHERE rule_id = id) = 0`,
		TableRuleAccesses, TableRuleAccesses), r.RemoteAgentID, r.ID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the authorized rules: %w", err)
	}

	return rules, nil
}
