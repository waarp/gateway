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

// Run executes the 'UPDATE' query.
func (u *UpdateQuery) Run() Error {
	logger := u.db.GetLogger()

	q := u.db.getUnderlying().NoAutoCondition().Table(u.bean.TableName()).
		Where("id=?", u.bean.GetID())
	exist, err := q.Exist()
	if err != nil {
		logger.Errorf("Failed to check if the %s to update exists: %s", u.bean.Appellation(), err)
		return NewInternalError(err)
	}
	if !exist {
		logger.Infof("No %s found with ID %d", u.bean.Appellation(), u.bean.GetID())
		return NewNotFoundError(u.bean)
	}

	if hook, ok := u.bean.(WriteHook); ok {
		if err := hook.BeforeWrite(u.db); err != nil {
			logger.Errorf("%s entry UPDATE validation failed: %s", u.bean.Appellation(), err)
			return err
		}
	}

	query := u.db.getUnderlying().NoAutoCondition().Table(u.bean.TableName()).ID(u.bean.GetID())
	if len(u.cols) == 0 {
		query = query.AllCols()
	} else {
		query = query.Cols(u.cols...)
	}

	_, err1 := query.Update(u.bean)
	logSQL(query, logger)
	if err1 != nil {
		logger.Errorf("Failed to update the %s entry: %s", u.bean.Appellation(), err1)
		return NewInternalError(err1)
	}

	return nil
}
