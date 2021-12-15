package model

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// AccessTarget is the interface implemented by all valid RuleAccess target types.
// Valid owner types are LocalAgent, RemoteAgent, LocalAccount & RemoteAccount.
type AccessTarget interface {
	// SetAccessTarget sets the target AccessTarget as target of the given RuleAccess
	// instance (by setting the corresponding foreign key to its own ID).
	SetAccessTarget(*RuleAccess)

	// GenAccessSelectCond returns the name of the RuleAccess column associated
	// with the target type.
	GenAccessSelectCond() (string, int64)

	GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error)
}

// RuleAccess links an owner to a rule it is allowed to use.
//
//nolint:lll //sql tags can be long
type RuleAccess struct {
	// foreign key (rules.id)
	RuleID          int64         `xorm:"BIGINT NOTNULL UNIQUE(locAg) UNIQUE(remAg) UNIQUE(locAcc) UNIQUE(remAcc) 'rule_id'"`
	LocalAgentID    sql.NullInt64 `xorm:"BIGINT UNIQUE(locAg) 'local_agent_id'"`     // foreign key (local_agents.id)
	RemoteAgentID   sql.NullInt64 `xorm:"BIGINT UNIQUE(remAg) 'remote_agent_id'"`    // foreign key (remote_agents.id)
	LocalAccountID  sql.NullInt64 `xorm:"BIGINT UNIQUE(locAcc) 'local_account_id'"`  // foreign key (local_accounts.id)
	RemoteAccountID sql.NullInt64 `xorm:"BIGINT UNIQUE(remAcc) 'remote_account_id'"` // foreign key (remote_accounts.id)
}

// TableName returns the rule access table name.
func (*RuleAccess) TableName() string {
	return TableRuleAccesses
}

// Appellation returns the name of 1 element of the rule access table.
func (*RuleAccess) Appellation() string {
	return "rule permission"
}

// BeforeWrite is called before inserting a new `RuleAccess` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RuleAccess) BeforeWrite(db database.ReadAccess) database.Error {
	i, err1 := db.Count(&Rule{}).Where("id=?", r.RuleID).Run()
	if err1 != nil {
		return err1
	} else if i < 1 {
		return database.NewValidationError("no rule found with ID %d", r.RuleID)
	}

	if sum := boolToInt(r.LocalAgentID.Valid) + boolToInt(r.RemoteAgentID.Valid) +
		boolToInt(r.LocalAccountID.Valid) + boolToInt(r.RemoteAccountID.Valid); sum == 0 {
		return database.NewValidationError("the rule access is missing a target")
	} else if sum > 1 {
		return database.NewValidationError("the rule access cannot have multiple targets")
	}

	var target interface {
		database.UpdateBean
		AccessTarget
	}

	switch {
	case r.LocalAgentID.Valid:
		target = &LocalAgent{ID: r.LocalAgentID.Int64}
	case r.RemoteAgentID.Valid:
		target = &RemoteAgent{ID: r.RemoteAgentID.Int64}
	case r.LocalAccountID.Valid:
		target = &LocalAccount{ID: r.LocalAccountID.Int64}
	case r.RemoteAccountID.Valid:
		target = &RemoteAccount{ID: r.RemoteAccountID.Int64}
	default:
		return database.NewValidationError("the rule access is missing a target") // impossible
	}

	if n, err := db.Count(target).Where("id=?", target.GetID()).Run(); err != nil {
		return err
	} else if n == 0 {
		return database.NewValidationError("no %s found with ID %v", target.Appellation(),
			target.GetID())
	}

	if n, err := db.Count(r).Where("rule_id=?", r.RuleID).Where(
		target.GenAccessSelectCond()).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("the target has already been granted access " +
			"to this rule")
	}

	return nil
}

// IsRuleAuthorized verify if the rule requested by the transfer is authorized for
// the requesting transfer.
func IsRuleAuthorized(db database.ReadAccess, t *Transfer) (bool, database.Error) {
	n, err := db.Count(&RuleAccess{}).Where("rule_id=?", t.RuleID).Run()
	if err != nil {
		return false, err
	} else if n == 0 {
		return true, nil
	}

	if t.IsServer() {
		n, err = db.Count(&RuleAccess{}).Where(`rule_id=? AND (local_account_id=? OR
			local_agent_id = (SELECT local_agent_id FROM local_accounts WHERE id=?) )`,
			t.RuleID, t.LocalAccountID, t.LocalAccountID).Run()
	} else {
		n, err = db.Count(&RuleAccess{}).Where(`rule_id=? AND (remote_account_id=? OR
			remote_agent_id = (SELECT remote_agent_id FROM remote_accounts WHERE id=?) )`,
			t.RuleID, t.RemoteAccountID, t.RemoteAccountID).Run()
	}

	if err != nil {
		return false, err
	} else if n < 1 {
		return false, nil
	}

	return true, nil
}

func (*RuleAccess) MakeExtraConstraints(db *database.Executor) database.Error {
	// add a foreign key to 'rule_id'
	if err := redefineColumn(db, TableRuleAccesses, "rule_id", fmt.Sprintf(
		`BIGINT NOT NULL REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE CASCADE`,
		TableRules)); err != nil {
		return err
	}

	// add a foreign key to 'local_agent_id'
	if err := redefineColumn(db, TableRuleAccesses, "local_agent_id", fmt.Sprintf(
		`BIGINT REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE CASCADE`,
		TableLocAgents)); err != nil {
		return err
	}

	// add a foreign key to 'remote_agent_id'
	if err := redefineColumn(db, TableRuleAccesses, "remote_agent_id", fmt.Sprintf(
		`BIGINT REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE CASCADE`,
		TableRemAgents)); err != nil {
		return err
	}

	// add a foreign key to 'local_account_id'
	if err := redefineColumn(db, TableRuleAccesses, "local_account_id", fmt.Sprintf(
		`BIGINT REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE CASCADE`,
		TableLocAccounts)); err != nil {
		return err
	}

	// add a foreign key to 'remote_account_id'
	if err := redefineColumn(db, TableRuleAccesses, "remote_account_id", fmt.Sprintf(
		`BIGINT REFERENCES %s(id) ON UPDATE RESTRICT ON DELETE CASCADE`,
		TableRemAccounts)); err != nil {
		return err
	}

	// add a constraint to enforce that one (and ONLY ONE) of 'local_agent_id',
	// 'remote_agent_id', 'local_account_id' and 'remote_account_id' must be defined
	return addTableConstraint(db, TableRuleAccesses, utils.CheckOnlyOneNotNull(db.Dialect,
		"local_agent_id", "remote_agent_id", "local_account_id", "remote_account_id"))
}
