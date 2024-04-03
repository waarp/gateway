package model

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// AccessTarget is the interface implemented by all valid RuleAccess target types.
// Valid owner types are LocalAgent, RemoteAgent, LocalAccount & RemoteAccount.
type AccessTarget interface {
	database.Identifier

	// SetAccessTarget sets the target AccessTarget as target of the given RuleAccess
	// instance (by setting the corresponding foreign key to its own ID).
	SetAccessTarget(access *RuleAccess)

	// GenAccessSelectCond returns the name of the RuleAccess column associated
	// with the target type.
	GenAccessSelectCond() (string, int64)

	GetAuthorizedRules(db database.ReadAccess) ([]*Rule, error)
}

// RuleAccess links an owner to a rule it is allowed to use.
//
//nolint:lll //sql tags can be long
type RuleAccess struct {
	RuleID int64 `xorm:"rule_id"` // The ID of the rule this permission applies to.

	// The ID of the target to which this permission applies. Only one can be
	// valid at a time.
	LocalAgentID    sql.NullInt64 `xorm:"local_agent_id"`
	RemoteAgentID   sql.NullInt64 `xorm:"remote_agent_id"`
	LocalAccountID  sql.NullInt64 `xorm:"local_account_id"`
	RemoteAccountID sql.NullInt64 `xorm:"remote_account_id"`
}

func (*RuleAccess) TableName() string   { return TableRuleAccesses }
func (*RuleAccess) Appellation() string { return NameRuleAccess }

// BeforeWrite is called before inserting a new `RuleAccess` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RuleAccess) BeforeWrite(db database.ReadAccess) error {
	if n, err := db.Count(&Rule{}).Where("id=?", r.RuleID).Run(); err != nil {
		return fmt.Errorf("failed to check access rule: %w", err)
	} else if n < 1 {
		return database.NewValidationError("no rule found with ID %d", r.RuleID)
	}

	if sum := countTrue(r.LocalAgentID.Valid, r.RemoteAgentID.Valid,
		r.LocalAccountID.Valid, r.RemoteAccountID.Valid); sum == 0 {
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
		return fmt.Errorf("failed to check access target: %w", err)
	} else if n == 0 {
		return database.NewValidationError("no %s found with ID %q", target.Appellation(),
			target.GetID())
	}

	if n, err := db.Count(r).Where("rule_id=?", r.RuleID).Where(
		target.GenAccessSelectCond()).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate accesses: %w", err)
	} else if n > 0 {
		return database.NewValidationError("the target has already been granted access " +
			"to this rule")
	}

	return nil
}

// IsRuleAuthorized verify if the rule requested by the transfer is authorized for
// the requesting transfer.
func IsRuleAuthorized(db database.ReadAccess, t *Transfer) (bool, error) {
	n, err := db.Count(&RuleAccess{}).Where("rule_id=?", t.RuleID).Run()
	if err != nil {
		return false, fmt.Errorf("failed to count rule accesses: %w", err)
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
		return false, fmt.Errorf("failed to count rule accesses: %w", err)
	} else if n < 1 {
		return false, nil
	}

	return true, nil
}
