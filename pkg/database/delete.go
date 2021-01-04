package database

// DeleteBean is the interface that a model must implement in order to be
// deletable via the Delete query builder.
type DeleteBean interface {
	Table
	Identifier
	DeletionHook
}

// DeleteQuery is the type representing a SQL DELETE statement with an ID
// condition (so for a single entry). The ID is taken from the given model.
type DeleteQuery struct {
	db   Access
	bean DeleteBean
}

// Run executes the 'DELETE' query.
func (d *DeleteQuery) Run() Error {
	logger := d.db.GetLogger()

	exist, err := d.db.getUnderlying().NoAutoCondition().ID(d.bean.GetID()).Exist(d.bean)
	if err != nil {
		logger.Errorf("Failed to check if the %s to update exists: %s", d.bean.Appellation(), err)
		return NewInternalError(err)
	}
	if !exist {
		logger.Infof("No %s found with ID %d", d.bean.Appellation(), d.bean.GetID())
		return NewNotFoundError(d.bean)
	}

	f := func(s *Session) Error {
		if err := d.bean.BeforeDelete(s); err != nil {
			logger.Errorf("%s deletion hook failed: %s", d.bean.Appellation(), err)
			return err
		}
		query := s.getUnderlying().NoAutoCondition().Table(d.bean.TableName()).
			ID(d.bean.GetID())
		_, err = query.Delete(d.bean)
		logSQL(query, logger)

		if err != nil {
			logger.Errorf("Failed to delete the %s entry: %s", d.bean.Appellation(), err)
			return NewInternalError(err)
		}
		return nil
	}

	switch db := d.db.(type) {
	case *Standalone:
		return db.Transaction(f)
	case *Session:
		return f(db)
	default:
		panic("unknown database accessor type")
	}
}
