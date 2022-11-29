package model

import (
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

// Rule represents a transfer rule.
type Rule struct {
	ID int64 `xorm:"<- id AUTOINCR"` // The Rule's ID

	Name    string `xorm:"name"`    // The rule's name
	IsSend  bool   `xorm:"is_send"` // The rule's direction (pull/push)
	Comment string `xorm:"comment"` // An optional comment on the rule.

	// The path used to differentiate the rule when the protocol does not allow it.
	Path string `xorm:"path"`

	LocalDir       string `xorm:"local_dir"`             // The local directory for transfers.
	RemoteDir      string `xorm:"remote_dir"`            // The remote directory for transfers.
	TmpLocalRcvDir string `xorm:"tmp_local_receive_dir"` // The local temporary directory for transfers.
}

func (*Rule) TableName() string   { return TableRules }
func (*Rule) Appellation() string { return "rule" }
func (r *Rule) GetID() int64      { return r.ID }

func (r *Rule) normalizePaths() {
	r.Path = path.Clean(r.Path)
	if r.Path == "/" || r.Path == "." {
		r.Path = r.Name
	} else {
		r.Path = utils.ToStandardPath(r.Path)
		if path.IsAbs(r.Path) {
			r.Path = r.Path[1:]
		}
	}

	if r.LocalDir != "" {
		r.LocalDir = utils.ToOSPath(r.LocalDir)
	}

	if r.RemoteDir != "" {
		r.RemoteDir = utils.ToStandardPath(r.RemoteDir)
	}

	if r.TmpLocalRcvDir != "" {
		r.TmpLocalRcvDir = utils.ToOSPath(r.TmpLocalRcvDir)
	}
}

func (r *Rule) checkAncestor(db database.ReadAccess, rulePath string) database.Error {
	if rulePath == "" || rulePath == "." {
		return nil
	}

	var rule Rule
	if err := db.Get(&rule, "path=?", rulePath).Run(); err != nil {
		if database.IsNotFound(err) {
			return r.checkAncestor(db, path.Dir(rulePath))
		}

		return err
	}

	return database.NewValidationError("the rule's path cannot be the descendant of "+
		"another rule's path (the path '%s' is already used by rule '%s')", rulePath, rule.Name)
}

func (r *Rule) checkPath(db database.ReadAccess) database.Error {
	if n, err := db.Count(r).Where("id<>? AND path=? AND is_send=?", r.ID, r.Path,
		r.IsSend).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("a rule with path: %s already exist", r.Path)
	}

	// check descendents
	if n, err := db.Count(r).Where("path LIKE ?", r.Path+"/%").Run(); err != nil {
		return err
	} else if n != 0 {
		return database.NewValidationError("the rule's path cannot be the ancestor " +
			"of another rule's path")
	}

	return r.checkAncestor(db, path.Dir(r.Path))
}

// BeforeWrite is called before writing the `Rule` entry in the database. It
// checks whether the new entry is valid or not.
func (r *Rule) BeforeWrite(db database.ReadAccess) database.Error {
	if r.Name == "" {
		return database.NewValidationError("the rule's name cannot be empty")
	}

	n, err := db.Count(r).Where("id<>? AND name=? AND is_send=?", r.ID,
		r.Name, r.IsSend).Run()
	if err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("a %s rule named '%s' already exist",
			r.Direction(), r.Name)
	}

	r.normalizePaths()

	return r.checkPath(db)
}

// Direction returns the direction (send or receive) of the rule as a string.
func (r *Rule) Direction() string {
	if r.IsSend {
		return "send"
	}

	return "receive"
}

// BeforeDelete is called before deleting the rule from the database. Its
// role is to check whether the rule is still used in any ongoing transfer.
func (r *Rule) BeforeDelete(db database.Access) database.Error {
	if n, err := db.Count(&Transfer{}).Where("rule_id=?", r.ID).Run(); err != nil {
		return err
	} else if n > 0 {
		return database.NewValidationError("this rule is currently being used in a " +
			"running transfer and cannot be deleted, cancel the transfer or wait " +
			"for it to finish")
	}

	return nil
}

// IsAuthorized returns whether the given target is authorized to use the rule
// for transfers. It will return true either if the rule has no restrictions, or
// if the target has been given access to the rule.
//
// Valid target types are: LocalAgent, RemoteAgent, LocalAccount & RemoteAccount.
func (r *Rule) IsAuthorized(db database.Access, target database.IterateBean) (bool, database.Error) {
	var perms RuleAccess
	if n, err := db.Count(&perms).Where("rule_id=?", r.ID).Run(); err != nil {
		return false, err
	} else if n == 0 {
		return true, nil
	}

	var query *database.CountQuery
	switch object := target.(type) {
	case *LocalAgent:
		query = db.Count(&perms).Where("rule_id=? AND local_agent_id=?", r.ID, object.ID)
	case *RemoteAgent:
		query = db.Count(&perms).Where("rule_id=? AND remote_agent_id=?", r.ID, object.ID)
	case *LocalAccount:
		query = db.Count(&perms).Where("rule_id=? AND ( local_account_id=? OR "+
			"local_agent_id=? )", r.ID, object.ID, object.LocalAgentID)
	case *RemoteAccount:
		query = db.Count(&perms).Where("rule_id=? AND ( remote_account_id=? OR "+
			"remote_agent_id=?", r.ID, object.ID, object.RemoteAgentID)
	default:
		return false, database.NewValidationError("%T is not a valid target model for RuleAccess", target)
	}

	if permCount, err := query.Run(); err != nil {
		return false, err
	} else {
		return permCount != 0, nil
	}
}
