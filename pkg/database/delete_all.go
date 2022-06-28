package database

import "xorm.io/builder"

// DeleteAllBean is the interface that a model must implement in order to be
// deletable via the Delete query builder.
type DeleteAllBean interface {
	Table
}

// DeleteAllQuery is the type representing a SQL DELETE statement.
type DeleteAllQuery struct {
	db   Access
	bean DeleteAllBean

	conds []cond
}

// Where adds a 'WHERE' clause to the 'DELETE' query with the given conditions
// and arguments. The function uses the `?` character as verb for the arguments
// in the given string.
//
// If the function is called multiple times, all the conditions will be chained
// using the 'AND' operator.
func (d *DeleteAllQuery) Where(sql string, args ...interface{}) *DeleteAllQuery {
	d.conds = append(d.conds, cond{sql: sql, args: args})

	return d
}

// Run executes the 'DELETE ALL' query.
func (d *DeleteAllQuery) Run() Error {
	logger := d.db.GetLogger()
	query := d.db.getUnderlying().NoAutoCondition()

	if len(d.conds) == 0 {
		_, err := query.Exec("DELETE FROM " + d.bean.TableName())
		logSQL(query, logger)

		if err != nil {
			logger.Error("Failed to delete the %s entries: %s", d.bean.Appellation(), err)

			return NewInternalError(err)
		}

		return nil
	}

	for i := range d.conds {
		query.Where(builder.Expr(d.conds[i].sql, d.conds[i].args...))
	}

	_, err := query.Table(d.bean.TableName()).Delete(d.bean)

	logSQL(query, logger)

	if err != nil {
		logger.Error("Failed to delete the %s entries: %s", d.bean.Appellation(), err)

		return NewInternalError(err)
	}

	return nil
}
