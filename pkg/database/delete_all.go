package database

import (
	"strings"

	"xorm.io/builder"
)

// DeleteAllBean is the interface that a model must implement in order to be
// deletable via the Delete query builder.
type DeleteAllBean interface {
	Table
}

// DeleteAllQuery is the type representing a SQL DELETE statement.
type DeleteAllQuery struct {
	db   Access
	bean DeleteAllBean

	conds []*condition
}

// Where adds a 'WHERE' clause to the 'DELETE' query with the given conditions
// and arguments. The function uses the `?` character as verb for the arguments
// in the given string.
//
// If the function is called multiple times, all the conditions will be chained
// using the 'AND' operator.
func (d *DeleteAllQuery) Where(sql string, args ...interface{}) *DeleteAllQuery {
	d.conds = append(d.conds, &condition{sql: sql, args: args})

	return d
}

// In add a 'WHERE col IN' condition to the 'DELETE' query. Because the database/sql
// package cannot handle variadic placeholders in the Where function, a separate
// method is required.
func (d *DeleteAllQuery) In(col string, vals ...interface{}) *DeleteAllQuery {
	sql := &inCond{Builder: &strings.Builder{}}
	if builder.In(col, vals...).WriteTo(sql) != nil {
		return d
	}

	d.conds = append(d.conds, &condition{sql: sql.String(), args: sql.args})

	return d
}

// Run executes the 'DELETE ALL' query.
func (d *DeleteAllQuery) Run() error {
	logger := d.db.GetLogger()
	query := d.db.getUnderlying().NoAutoCondition()

	if len(d.conds) == 0 {
		if _, err := query.Exec("DELETE FROM " + d.bean.TableName()); err != nil {
			logger.Error("Failed to delete the %s entries: %s", d.bean.Appellation(), err)

			return NewInternalError(err)
		}

		return nil
	}

	for _, cond := range d.conds {
		query.And(cond.sql, cond.args...)
	}

	if _, err := query.Table(d.bean.TableName()).Delete(d.bean); err != nil {
		logger.Error("Failed to delete the %s entries: %s", d.bean.Appellation(), err)

		return NewInternalError(err)
	}

	return nil
}
