package database

import (
	"fmt"
)

// InsertBean is the interface that a model must implement in order to be
// insertable via the Access.Insert query builder.
type InsertBean interface {
	Table
}

// InsertQuery is the type representing a SQL INSERT statement.
type InsertQuery struct {
	db   Access
	bean InsertBean
}

func (q *InsertQuery) run(s *Session) error {
	logger := s.getLogger()
	engine := s.getUnderlying()
	s.addOwner(q.bean)

	if hook, ok := q.bean.(WriteHook); ok {
		if err := hook.BeforeWrite(s); err != nil {
			logger.Errorf("%s entry INSERT validation failed: %v", q.bean.Appellation(), err)

			return fmt.Errorf("%s entry INSERT validation failed: %w", q.bean.Appellation(), err)
		}
	}

	query := engine.Table(q.bean.TableName())
	if err := query.Create(q.bean).Error; err != nil {
		logger.Errorf("Failed to insert the new %s entries: %v", q.bean.Appellation(), err)

		return NewInternalError(err)
	}

	if callBack, ok := q.bean.(InsertCallback); ok {
		if err := callBack.AfterInsert(s); err != nil {
			logger.Errorf("%s entry INSERT callback failed: %v", q.bean.Appellation(), err)

			return fmt.Errorf("%s entry INSERT callback failed: %w", q.bean.Appellation(), err)
		}
	}

	return nil
}

// Run executes the 'INSERT' query.
func (q *InsertQuery) Run() error {
	return q.db.Transaction(q.run)
}
