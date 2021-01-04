package database

// InsertBean is the interface that a model must implement in order to be
// insertable via the Access.Insert query builder.
type InsertBean interface {
	Table
	WriteHook
}

// InsertQuery is the type representing a SQL INSERT statement.
type InsertQuery struct {
	db   Access
	bean InsertBean
}

// Run executes the 'INSERT' query.
func (i *InsertQuery) Run() Error {
	logger := i.db.GetLogger()

	if err := i.bean.BeforeWrite(i.db); err != nil {
		logger.Errorf("%s entry INSERT validation failed: %s", i.bean.Appellation(), err)
		return err
	}

	query := i.db.getUnderlying().Table(i.bean.TableName())
	_, err := query.InsertOne(i.bean)
	logSQL(query, logger)
	if err != nil {
		logger.Errorf("Failed to insert the new %s entry: %s", i.bean.Appellation(), err)
		return NewInternalError(err)
	}

	return nil
}
