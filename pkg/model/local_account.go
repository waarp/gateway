package model

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// LocalAccount represents an account on a local agent. It is used by remote
// partners to authenticate on the gateway for transfers.
type LocalAccount struct {
	ID           int64 `xorm:"<- id AUTOINCR"` // The account's database ID.
	LocalAgentID int64 `xorm:"local_agent_id"` // The ID of the LocalAgent this account is attached to.

	Login       string       `xorm:"login"`        // The account's login.
	IPAddresses types.IPList `xorm:"ip_addresses"` // The account's allowed IP addresses.
}

func (*LocalAccount) TableName() string   { return TableLocAccounts }
func (*LocalAccount) Appellation() string { return NameLocalAccount }
func (l *LocalAccount) GetID() int64      { return l.ID }
func (l *LocalAccount) Host() string      { return "" }
func (*LocalAccount) IsServer() bool      { return false }

// GetCredentials fetch in the database then return the associated Credentials if they exist.
func (l *LocalAccount) GetCredentials(db database.ReadAccess, authTypes ...string,
) (Credentials, error) {
	return getCredentials(db, l, authTypes...)
}

// BeforeWrite checks if the new `LocalAccount` entry is valid and can be
// inserted in the database.
//
//nolint:dupl // too many differences
func (l *LocalAccount) BeforeWrite(db database.Access) error {
	if l.LocalAgentID == 0 {
		return database.NewValidationError("the account's agentID cannot be empty")
	}

	if l.Login == "" {
		return database.NewValidationError("the account's login cannot be empty")
	}

	if _, err := l.getParent(db); err != nil {
		return err
	}

	if err := l.IPAddresses.Validate(); err != nil {
		return database.NewValidationErrorf("invalid account IP address: %w", err)
	}

	if n, err := db.Count(l).Where("id<>? AND local_agent_id=? AND login=?",
		l.ID, l.LocalAgentID, l.Login).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate local accounts: %w", err)
	} else if n > 0 {
		return database.NewValidationErrorf("a local account with the same login %q "+
			"already exist", l.Login)
	}

	return nil
}

// BeforeDelete is called before deleting the account from the database. Its
// role is to check whether the account is still used in any ongoing transfer.
func (l *LocalAccount) BeforeDelete(db database.Access) error {
	if n, err := db.Count(&Transfer{}).Where("local_account_id=?", l.ID).Run(); err != nil {
		return fmt.Errorf("failed to check for ongoing transfers: %w", err)
	} else if n > 0 {
		//nolint:goconst //too specific
		return database.NewValidationError("this account is currently being used " +
			"in one or more running transfers and thus cannot be deleted, cancel " +
			"these transfers or wait for them to finish")
	}

	return nil
}

//nolint:goconst //duplicates are for different tables, best not to factorize
func (l *LocalAccount) GetCredCond() (string, int64)         { return "local_account_id=?", l.ID }
func (l *LocalAccount) SetCredOwner(a *Credential)           { a.LocalAccountID = utils.NewNullInt64(l.ID) }
func (l *LocalAccount) GenAccessSelectCond() (string, int64) { return "local_account_id=?", l.ID }
func (l *LocalAccount) SetAccessTarget(a *RuleAccess)        { a.LocalAccountID = utils.NewNullInt64(l.ID) }

func (l *LocalAccount) GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error) {
	var rules Rules
	if err := db.Select(&rules).Where(fmt.Sprintf(
		`id IN (SELECT DISTINCT rule_id FROM %s WHERE local_agent_id=? OR 
			local_account_id=?)
		  OR (SELECT COUNT(*) FROM %s WHERE rule_id = id) = 0`,
		TableRuleAccesses, TableRuleAccesses), l.LocalAgentID, l.ID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve the authorized rules: %w", err)
	}

	return rules, nil
}

func (l *LocalAccount) getParent(db database.ReadAccess) (*LocalAgent, error) {
	var parent LocalAgent
	if err := db.Get(&parent, "id=?", l.LocalAgentID).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, database.NewValidationErrorf(`no local agent found with the ID "%v"`, l.LocalAgentID)
		}

		return nil, fmt.Errorf("failed to check parent local agent: %w", err)
	}

	return &parent, nil
}

func (l *LocalAccount) GetProtocol(db database.ReadAccess) (string, error) {
	parent, err := l.getParent(db)
	if err != nil {
		return "", err
	}

	return parent.Protocol, nil
}

func (l *LocalAccount) Authenticate(db database.ReadAccess, localAgent *LocalAgent,
	authType string, value any,
) (*authentication.Result, error) {
	return authenticate(db, l, authType, localAgent.Protocol, value)
}
