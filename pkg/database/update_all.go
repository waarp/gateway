package database

import "xorm.io/builder"

// UpdateAllBean is the interface that a model must implement in order to be
// updatable via the Access.Update query builder.
type UpdateAllBean interface {
	Table
}

// UpdVals represents a list of 'column = value' pairs to be used in the 'SET'
// clause of an SQL 'UPDATE' statement.
type UpdVals map[string]interface{}

// UpdateAllQuery is the type representing a SQL UPDATE statement for a single entry.
type UpdateAllQuery struct {
	db   Access
	bean UpdateAllBean

	vals UpdVals

	conds string
	args  []interface{}
}

// Run executes the 'UPDATE ALL' query.
func (u *UpdateAllQuery) Run() Error {
	logger := u.db.GetLogger()

	ses := u.db.getUnderlying().NoAutoCondition()
	query := builder.Update(builder.Eq(u.vals)).From(u.bean.TableName()).
		Where(builder.Expr(u.conds, u.args...))

	defer logSQL(ses, logger)

	if _, err := ses.Exec(query); err != nil {
		logger.Error("Failed to update the %s entries: %s", u.bean.Appellation(), err)

		return NewInternalError(err)
	}

	return nil
}
