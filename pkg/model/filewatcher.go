package model

import (
	"path"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/features"
)

const FilewatcherDefaultPattern = "*"

type FileWatcher struct {
	Identifier
	Owner            string        `gorm:"column:owner"`
	Disabled         bool          `gorm:"column:disabled"`
	Flow             string        `gorm:"column:flow"`
	Interval         time.Duration `gorm:"column:interval"`
	Pattern          string        `gorm:"column:pattern"`
	NoDuplicateCheck bool          `gorm:"column:no_duplicate_check"`

	RemoteAccountID int64 `gorm:"column:remote_account_id"`
	RemoteAccount   RemoteAccount

	ClientID int64 `gorm:"column:client_id"`
	Client   Client

	RuleID int64 `gorm:"column:rule_id"`
	Rule   Rule
}

func (*FileWatcher) TableName() string   { return TableFileWatchers }
func (*FileWatcher) Appellation() string { return NameFileWatcher }

func (*FileWatcher) Preloads() []string {
	return []string{"RemoteAccount", "Client", "Rule"}
}

func (r *FileWatcher) BeforeWrite(database.Access) error {
	if err := r.checkMandatoryValues(); err != nil {
		return err
	}

	if !features.Supports(r.Client.Protocol, features.Listing) {
		return database.NewValidationErrorf("protocol %q does not support listing files", r.Client.Protocol)
	}

	return nil
}

func (r *FileWatcher) checkMandatoryValues() error {
	if r.Pattern == "" {
		r.Pattern = FilewatcherDefaultPattern
	}

	if _, err := path.Match(r.Pattern, ""); err != nil {
		return database.NewValidationError("invalid filewatcher pattern")
	}

	if r.Flow == "" {
		return database.NewValidationError("the filewatcher's flow name is missing")
	}

	r.RemoteAccountID = r.RemoteAccount.ID
	r.ClientID = r.Client.ID
	r.RuleID = r.Rule.ID

	if r.RemoteAccountID == 0 {
		return database.NewValidationError("the filewatcher's account is missing")
	}

	if r.ClientID == 0 {
		return database.NewValidationError("the filewatcher's client is missing")
	}

	if r.RuleID == 0 {
		return database.NewValidationError("the filewatcher's rule is missing")
	}

	if r.Rule.IsSend {
		return database.NewValidationError("the filewatcher's rule must be a receive rule")
	}

	return nil
}
