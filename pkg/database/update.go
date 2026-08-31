package database

import (
	"fmt"

	"gorm.io/gorm/clause"
)

// UpdateBean is the interface that a model must implement in order to be
// updatable via the Access.Update query builder.
type UpdateBean interface {
	Table
	Identifier
}

// UpdateQuery is the type representing a SQL UPDATE statement for a single entry.
type UpdateQuery struct {
	db   Access
	bean UpdateBean

	cols []string
	all  bool
}

func (u *UpdateQuery) All() *UpdateQuery {
	u.all = true
	return u
}

// Cols allows to specify the list of columns to update to perform a partial
// update of the entry, instead of a full replacement, which should improve
// performance a bit and make the logs more readable.
func (u *UpdateQuery) Cols(columns ...string) *UpdateQuery {
	u.cols = append(u.cols, columns...)

	return u
}

func (u *UpdateQuery) run(ses *Session) error {
	logger := ses.getLogger()
	ses.addOwner(u.bean)
	addOwnerCond(ses.session, u.all, u.bean, ses.getOwner())

	if hook, ok := u.bean.(WriteHook); ok {
		if err := hook.BeforeWrite(ses); err != nil {
			logger.Errorf("%s entry UPDATE validation failed: %v", u.bean.Appellation(), err)

			return fmt.Errorf("%s entry UPDATE validation failed: %w", u.bean.Appellation(), err)
		}
	}

	// Associations are handled by the models' own hooks. Without this, the "*"
	// below makes GORM upsert them with an ON CONFLICT clause which has no
	// conflict target, and which PostgreSQL therefore rejects.
	query := ses.session.Table(u.bean.TableName()).Where("id=?", u.bean.GetID()).
		Omit(clause.Associations)

	if len(u.cols) != 0 {
		query = query.Select(u.cols)
	} else {
		query = query.Select("*") //nolint:unqueryvet // "*" is necessary here
	}

	if err := query.Updates(u.bean).Error; err != nil {
		logger.Errorf("Failed to update the %s entry n°%d: %v",
			u.bean.Appellation(), u.bean.GetID(), err)

		return NewInternalError(err)
	}

	if callback, ok := u.bean.(UpdateCallback); ok {
		if err := callback.AfterUpdate(ses); err != nil {
			logger.Errorf("%s entry UPDATE callback failed: %v", u.bean.Appellation(), err)

			return fmt.Errorf("%s entry UPDATE callback failed: %w", u.bean.Appellation(), err)
		}
	}

	return nil
}

// Run executes the 'UPDATE' query.
func (u *UpdateQuery) Run() error {
	if err := checkExists(u.db, u.bean); err != nil {
		return err
	}

	switch db := u.db.(type) {
	case *DB:
		return db.Transaction(u.run)
	case *Session:
		return u.run(db)
	default:
		panic("unknown database accessor type")
	}
}
