package database

import "fmt"

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

func (i *InsertQuery) run(s *Session) error {
	if hook, ok := i.bean.(WriteHook); ok {
		if err := hook.BeforeWrite(s); err != nil {
			s.logger.Errorf("%s entry INSERT validation failed: %v", i.bean.Appellation(), err)

			return fmt.Errorf("%s entry INSERT validation failed: %w", i.bean.Appellation(), err)
		}
	}

	query := s.session.Table(i.bean.TableName())

	if _, err := query.Insert(i.bean); err != nil {
		s.logger.Errorf("Failed to insert the new %s entry: %v", i.bean.Appellation(), err)

		return NewInternalError(err)
	}

	if callBack, ok := i.bean.(InsertCallback); ok {
		if err := callBack.AfterInsert(s); err != nil {
			s.logger.Errorf("%s entry INSERT callback failed: %v", i.bean.Appellation(), err)

			return fmt.Errorf("%s entry INSERT callback failed: %w", i.bean.Appellation(), err)
		}
	}

	return nil
}

// Run executes the 'INSERT' query.
func (i *InsertQuery) Run() error {
	switch db := i.db.(type) {
	case *DB:
		return db.Transaction(i.run)
	case *Session:
		return i.run(db)
	default:
		panic("unknown database accessor type")
	}
}
