package database

// DeleteBean is the interface that a model must implement in order to be
// deletable via the Delete query builder.
type DeleteBean interface {
	Table
	Identifier
}

// DeleteQuery is the type representing a SQL DELETE statement with an ID
// condition (so for a single entry). The ID is taken from the given model.
type DeleteQuery struct {
	db   Access
	bean DeleteBean
}

func (d *DeleteQuery) run(s *Session) Error {
	if hook, ok := d.bean.(DeletionHook); ok {
		if err := hook.BeforeDelete(s); err != nil {
			s.logger.Errorf("%s deletion hook failed: %s", d.bean.Appellation(), err)
			return err
		}
	}
	query := s.session.NoAutoCondition().Table(d.bean.TableName()).
		ID(d.bean.GetID())
	_, err := query.Delete(d.bean)
	logSQL(query, s.logger)

	if err != nil {
		s.logger.Errorf("Failed to delete the %s entry: %s", d.bean.Appellation(), err)
		return NewInternalError(err)
	}
	return nil
}

// Run executes the 'DELETE' query.
func (d *DeleteQuery) Run() Error {
	if err := checkExists(d.db, d.bean); err != nil {
		return err
	}

	switch db := d.db.(type) {
	case *Standalone:
		return db.Transaction(d.run)
	case *Session:
		return d.run(db)
	default:
		panic("unknown database accessor type")
	}
}
