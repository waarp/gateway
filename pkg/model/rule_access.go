package model

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

func init() {
	database.AddTable(&RuleAccess{})
}

// RuleAccess represents a authorised access to a rule.
type RuleAccess struct {
	RuleID     uint64 `xorm:"notnull unique(perm) 'rule_id'"`
	ObjectID   uint64 `xorm:"notnull unique(perm) 'object_id'"`
	ObjectType string `xorm:"notnull unique(perm) 'object_type'"`
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

	var n uint64
	var err database.Error
	switch r.ObjectType {
	case TableLocAgents:
		n, err = db.Count(&LocalAgent{}).Where("id=?", r.ObjectID).Run()
	case TableRemAgents:
		n, err = db.Count(&RemoteAgent{}).Where("id=?", r.ObjectID).Run()
	case TableLocAccounts:
		n, err = db.Count(&LocalAccount{}).Where("id=?", r.ObjectID).Run()
	case TableRemAccounts:
		n, err = db.Count(&RemoteAccount{}).Where("id=?", r.ObjectID).Run()
	default:
		return database.NewValidationError("the rule_access's object type must be one of %s",
			validOwnerTypes)
	}
	if err != nil {
		return err
	} else if n == 0 {
		return database.NewValidationError("no %s found with ID %v", r.ObjectType, r.ObjectID)
	}

	n, err = db.Count(r).Where("rule_id=? AND object_type=? AND object_id=?",
		r.RuleID, r.ObjectType, r.ObjectID).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("the agent has already been granted access " +
			"to this rule")
	}

	return nil
}

// IsRuleAuthorized verify if the rule requested by the transfer is authorized for
// the requesting transfer
func IsRuleAuthorized(db database.ReadAccess, t *Transfer) (bool, database.Error) {
	n, err := db.Count(&RuleAccess{}).Where("rule_id=?", t.RuleID).Run()
	if err != nil {
		return false, err
	} else if n == 0 {
		return true, nil
	}

	agent := TableRemAgents
	account := TableRemAccounts
	if t.IsServer {
		agent = TableLocAgents
		account = TableLocAccounts
	}
	n, err = db.Count(&RuleAccess{}).Where("rule_id=? AND "+
		"((object_type=? AND object_id=?) OR (object_type=? and object_id=?))",
		t.RuleID, agent, t.AgentID, account, t.AccountID).Run()
	if err != nil {
		return false, err
	} else if n < 1 {
		return false, nil
	}
	return true, nil
}
