package database

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

func (i *InsertQuery) run(s *Session) Error {
	if hook, ok := i.bean.(WriteHook); ok {
		if err := hook.BeforeWrite(s); err != nil {
			s.logger.Errorf("%s entry INSERT validation failed: %s", i.bean.Appellation(), err)
			return err
		}
	}

	query := s.session.Table(i.bean.TableName())
	_, err := query.InsertOne(i.bean)
	logSQL(query, s.logger)
	if err != nil {
		s.logger.Errorf("Failed to insert the new %s entry: %s", i.bean.Appellation(), err)
		return NewInternalError(err)
	}

	return nil
}

// Run executes the 'INSERT' query.
func (i *InsertQuery) Run() Error {
	switch db := i.db.(type) {
	case *Standalone:
		return db.Transaction(i.run)
	case *Session:
		return i.run(db)
	default:
		panic("unknown database accessor type")
	}
}
