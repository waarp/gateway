package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

func init() {
	database.Tables = append(database.Tables, &RuleAccess{})
}

// RuleAccess represents a authorised access to a rule.
type RuleAccess struct {
	RuleID     uint64 `xorm:"notnull unique(perm) 'rule_id'"`
	ObjectID   uint64 `xorm:"notnull unique(perm) 'object_id'"`
	ObjectType string `xorm:"notnull unique(perm) 'object_type'"`
}

// TableName returns the rule access table name.
func (*RuleAccess) TableName() string {
	return "rule_access"
}

// ElemName returns the name of 1 element of the rule access table.
func (*RuleAccess) ElemName() string {
	return "rule permission"
}

// Validate is called before inserting a new `RuleAccess` entry in the
// database. It checks whether the new entry is valid or not.
func (r *RuleAccess) Validate(db database.Accessor) error {
	if res, err := db.Query("SELECT id FROM rules WHERE id=?", r.RuleID); err != nil {
		return database.NewInternalError(err, "failed to retrieve the list of rules")
	} else if len(res) < 1 {
		return database.NewValidationError("no rule found with ID %d", r.RuleID)
	}

	var res []map[string]interface{}
	var err error
	switch r.ObjectType {
	case "local_agents":
		res, err = db.Query("SELECT id FROM local_agents WHERE id=?", r.ObjectID)
	case "remote_agents":
		res, err = db.Query("SELECT id FROM remote_agents WHERE id=?", r.ObjectID)
	case "local_accounts":
		res, err = db.Query("SELECT id FROM local_accounts WHERE id=?", r.ObjectID)
	case "remote_accounts":
		res, err = db.Query("SELECT id FROM remote_accounts WHERE id=?", r.ObjectID)
	default:
		return database.NewValidationError("the rule_access's object type must be one of %s",
			validOwnerTypes)
	}
	if err != nil {
		return database.NewInternalError(err, "failed to retrieve the list of %s",
			r.ObjectType)
	} else if len(res) == 0 {
		return database.NewValidationError("no %s found with ID %v", r.ObjectType, r.ObjectID)
	}

	if err := db.Get(r); err == nil {
		return database.NewValidationError("the agent has already been granted access " +
			"to this rule")
	} else if _, ok := err.(*database.NotFoundError); !ok {
		return database.NewInternalError(err, "failed to check the existing rule permissions")
	}

	return nil
}

// IsRuleAuthorized verify if the rule requested by the transfer is authorized for
// the requesting transfer
func IsRuleAuthorized(db database.Accessor, t *Transfer) (bool, error) {
	res, err := db.Query("SELECT rule_id FROM rule_access WHERE rule_id=?", t.RuleID)
	if err != nil {
		return false, err
	} else if len(res) == 0 {
		return true, nil
	}

	agent := "remote_agents"
	account := "remote_accounts"
	if t.IsServer {
		agent = "local_agents"
		account = "local_accounts"
	}
	res, err = db.Query("SELECT rule_id FROM rule_access WHERE rule_id=? AND "+
		"((object_type=? AND object_id=?) OR (object_type=? and object_id=?))",
		t.RuleID, agent, t.AgentID, account, t.AccountID)
	if err != nil {
		return false, err
	} else if len(res) < 1 {
		return false, nil
	}
	return true, nil
}
