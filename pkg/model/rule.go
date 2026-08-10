package model

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

// Rule represents a transfer rule.
type Rule struct {
	Identifier

	Name    string `gorm:"column:name"`    // The rule's name
	IsSend  bool   `gorm:"column:is_send"` // The rule's direction (pull/push)
	Comment string `gorm:"column:comment"` // An optional comment on the rule.

	// The path used to differentiate the rule when the protocol does not allow it.
	Path string `gorm:"column:path"`

	LocalDir       string `gorm:"column:local_dir"`             // The local directory for transfers.
	RemoteDir      string `gorm:"column:remote_dir"`            // The remote directory for transfers.
	TmpLocalRcvDir string `gorm:"column:tmp_local_receive_dir"` // The local temporary directory for transfers.

	Tasks      []*Task `gorm:"foreignKey:RuleID;references:ID"`
	PreTasks   []*Task `gorm:"-"`
	PostTasks  []*Task `gorm:"-"`
	ErrorTasks []*Task `gorm:"-"`

	AuthorizedServers        []*LocalAgent    `gorm:"many2many:rule_access"`
	AuthorizedPartners       []*RemoteAgent   `gorm:"many2many:rule_access"`
	AuthorizedLocalAccounts  []*LocalAccount  `gorm:"many2many:rule_access"`
	AuthorizedRemoteAccounts []*RemoteAccount `gorm:"many2many:rule_access"`
}

func (*Rule) TableName() string   { return TableRules }
func (*Rule) Appellation() string { return NameRule }
func (*Rule) Preloads() []string {
	return []string{
		"Tasks",
		"AuthorizedServers",
		"AuthorizedPartners",
		"AuthorizedLocalAccounts", "AuthorizedLocalAccounts.LocalAgent",
		"AuthorizedRemoteAccounts", "AuthorizedRemoteAccounts.RemoteAgent",
	}
}

func (r *Rule) checkAncestor(db database.ReadAccess, rulePath string) error {
	if rulePath == "" || rulePath == "." || rulePath == "/" {
		return nil
	}

	var rule Rule
	if err := db.Get(&rule, "path=?", rulePath).Run(); err != nil {
		if database.IsNotFound(err) {
			return r.checkAncestor(db, path.Dir(rulePath))
		}

		return fmt.Errorf("failed to check for ancestor rule paths: %w", err)
	}

	return database.NewValidationErrorf("the rule's path cannot be the descendant of "+
		"another rule's path (the path %q is already used by rule %q)", rulePath, rule.Name)
}

func (r *Rule) checkPath(db database.ReadAccess) error {
	if n, err := db.Count(r).Where("id<>? AND path=? AND is_send=?", r.ID, r.Path,
		r.IsSend).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate rule paths: %w", err)
	} else if n > 0 {
		return database.NewValidationErrorf("a rule with path: %s already exist", r.Path)
	}

	// check descendants
	if n, err := db.Count(r).Where("path LIKE ?", r.Path+"/%").Run(); err != nil {
		return fmt.Errorf("failed to check for descendants rule paths: %w", err)
	} else if n != 0 {
		return database.NewValidationError("the rule's path cannot be the ancestor " +
			"of another rule's path")
	}

	return r.checkAncestor(db, path.Dir(r.Path))
}

// BeforeWrite is called before writing the `Rule` entry in the database. It
// checks whether the new entry is valid or not.
func (r *Rule) BeforeWrite(db database.Access) error {
	if r.Name == "" {
		return database.NewValidationError("the rule's name cannot be empty")
	}

	n, err := db.Count(r).Where("id<>? AND name=? AND is_send=?", r.ID,
		r.Name, r.IsSend).Run()
	if err != nil {
		return fmt.Errorf("failed to check for duplicate rules: %w", err)
	} else if n > 0 {
		return database.NewValidationErrorf("a %s rule named %q already exist",
			r.Direction(), r.Name)
	}

	r.Path = path.Clean(filepath.ToSlash(r.Path))
	if r.Path == "/" || r.Path == "." {
		r.Path = r.Name
	} else if path.IsAbs(r.Path) {
		r.Path = strings.TrimLeft(r.Path, "/")
	}

	if !fs.IsLocalPath(r.TmpLocalRcvDir) {
		return database.NewValidationError("rule tmp directory must be local")
	}

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
func (r *Rule) BeforeDelete(db database.Access) error {
	if n, err := db.Count(&Transfer{}).Where("rule_id=?", r.ID).Run(); err != nil {
		return fmt.Errorf("failed to check for ongoing transfers: %w", err)
	} else if n > 0 {
		return database.NewValidationError("this rule is currently being used in a " +
			"running transfer and cannot be deleted, cancel the transfer or wait " +
			"for it to finish")
	}

	return nil
}

func (r *Rule) AfterRead(database.ReadAccess) error {
	return r.fillPreloadedTasks()
}

func (r *Rule) fillPreloadedTasks() error {
	if len(r.Tasks) == 0 {
		return nil
	}

	slices.SortFunc(r.Tasks, func(a, b *Task) int { return int(a.Rank - b.Rank) })

	r.PreTasks = make([]*Task, 0, len(r.Tasks))
	r.PostTasks = make([]*Task, 0, len(r.Tasks))
	r.ErrorTasks = make([]*Task, 0, len(r.Tasks))

	for _, task := range r.Tasks {
		switch task.Chain {
		case ChainPre:
			r.PreTasks = append(r.PreTasks, task)
		case ChainPost:
			r.PostTasks = append(r.PostTasks, task)
		case ChainError:
			r.ErrorTasks = append(r.ErrorTasks, task)
		}
	}

	return nil
}

func (r *Rule) IsAuthorized(acc *LocalAccount) bool {
	// Check if rule has permissions at all (if none, it is authorized)
	if len(r.AuthorizedServers) == 0 && len(r.AuthorizedPartners) == 0 &&
		len(r.AuthorizedLocalAccounts) == 0 && len(r.AuthorizedRemoteAccounts) == 0 {
		return true
	}

	for _, authAcc := range r.AuthorizedLocalAccounts {
		if authAcc.ID == acc.ID {
			return true
		}
	}

	for _, authServ := range r.AuthorizedServers {
		if authServ.ID == acc.LocalAgentID {
			return true
		}
	}

	return false
}
