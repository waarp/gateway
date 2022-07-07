package database

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
}

// Cols allows to specify the list of columns to update to perform a partial
// update of the entry, instead of a full replacement, which should improve
// performance a bit and make the logs more readable.
func (u *UpdateQuery) Cols(columns ...string) *UpdateQuery {
	u.cols = append(u.cols, columns...)

	return u
}

func (u *UpdateQuery) run(s *Session) Error {
	if hook, ok := u.bean.(WriteHook); ok {
		if err := hook.BeforeWrite(s); err != nil {
			s.logger.Error("%s entry UPDATE validation failed: %s", u.bean.Appellation(), err)

			return err
		}
	}

	query := s.session.NoAutoCondition().Table(u.bean.TableName()).ID(u.bean.GetID())
	if len(u.cols) == 0 {
		query = query.AllCols()
	} else {
		query = query.Cols(u.cols...)
	}

	defer logSQL(query, s.logger)

	if _, err1 := query.Update(u.bean); err1 != nil {
		s.logger.Error("Failed to update the %s entry: %s", u.bean.Appellation(), err1)

		return NewInternalError(err1)
	}

	return nil
}

// Run executes the 'UPDATE' query.
func (u *UpdateQuery) Run() Error {
	if err := checkExists(u.db, u.bean); err != nil {
		return err
	}

	switch db := u.db.(type) {
	case *Standalone:
		return db.Transaction(u.run)
	case *Session:
		return u.run(db)
	default:
		panic("unknown database accessor type")
	}
}
