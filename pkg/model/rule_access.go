package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &RuleAccess{})
}

// RuleAccess represents a authorised access to a rule.
type RuleAccess struct {
	RuleID     uint64 `xorm:"notnull 'rule_id'" json:"ruleId"`
	ObjectID   uint64 `xorm:"notnull 'object_id'" json:"objectId"`
	ObjectType string `xorm:"notnull 'object_type'" json:"objectType"`
}

// TableName returns the rule access table name.
func (*RuleAccess) TableName() string {
	return "rule_access"
}

// ValidateInsert is called before inserting a new `RuleAccess` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RuleAccess) ValidateInsert(acc database.Accessor) error {
	if res, err := acc.Query("SELECT id FROM rules WHERE id=?", r.RuleID); err != nil {
		return err
	} else if len(res) < 1 {
		return database.InvalidError("No rule found with ID %d", r.RuleID)
	}

	var res []map[string]interface{}
	var err error
	switch r.ObjectType {
	case "local_agents":
		res, err = acc.Query("SELECT id FROM local_agents WHERE id=?", r.ObjectID)
	case "remote_agents":
		res, err = acc.Query("SELECT id FROM remote_agents WHERE id=?", r.ObjectID)
	case "local_accounts":
		res, err = acc.Query("SELECT id FROM local_accounts WHERE id=?", r.ObjectID)
	case "remote_accounts":
		res, err = acc.Query("SELECT id FROM remote_accounts WHERE id=?", r.ObjectID)
	default:
		return database.InvalidError("The rule_access's object type must be one of %s",
			validOwnerTypes)
	}
	if err != nil {
		return err
	} else if len(res) == 0 {
		return database.InvalidError("No "+r.ObjectType+" found with ID '%v'", r.ObjectID)
	}

	return nil
}

// ValidateUpdate is called before updating and existing `RuleAccess` entry from
// the database. It rejects all update.
func (*RuleAccess) ValidateUpdate(acc database.Accessor) error {
	return database.InvalidError("Unallowed operation")
}
