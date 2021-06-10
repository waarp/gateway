package database

import (
	"fmt"
	"strings"

	"xorm.io/builder"
)

// IterateBean is the interface that a model must implement in order to be
// selectable via the Iterate query builder.
type IterateBean interface {
	Table
}

type cond struct {
	sql  string
	args []interface{}
}

// IterateQuery is the type representing a SQL SELECT statement. The returned
// values are then wrapped inside an Iterator.
type IterateQuery struct {
	db   Access
	bean IterateBean

	lim, off int
	conds    []cond
	distinct []string
	order    string
	asc      bool
}

// Where adds a 'WHERE' clause to the 'SELECT' query with the given conditions
// and arguments. The function uses the `?` character as verb for the arguments
// in the given string.
//
// If the function is called multiple times, all the conditions will be chained
// using the 'AND' operator.
func (i *IterateQuery) Where(sql string, args ...interface{}) *IterateQuery {
	i.conds = append(i.conds, cond{sql: sql, args: args})
	return i
}

// In add a 'WHERE col IN' condition to the 'SELECT' query. Because the database/sql
// package cannot handle variadic placeholders in the Where function, a separate
// method is required.
func (i *IterateQuery) In(col string, vals ...interface{}) *IterateQuery {
	sql := &inCond{Builder: &strings.Builder{}}
	if builder.In(col, vals...).WriteTo(sql) != nil {
		return i
	}
	i.conds = append(i.conds, cond{sql: sql.String(), args: sql.args})
	return i
}

// Distinct is used to add a 'DISTINCT' clause to the 'SELECT' query on the
// specified columns. Be aware that if Distinct is called, only the specified
// columns will be retrieved from the database, all the others will be ignored.
//
// If the function is called multiple times, all the columns will be taken into
// account for the SELECT.
func (i *IterateQuery) Distinct(columns ...string) *IterateQuery {
	i.distinct = append(i.distinct, columns...)
	return i
}

// OrderBy adds an 'ORDER BY' clause to the 'SELECT' query with the given order
// and direction.
func (i *IterateQuery) OrderBy(order string, asc bool) *IterateQuery {
	i.order = order
	i.asc = asc
	return i
}

// Limit adds an 'LIMIT' clause to the 'SELECT' query with the given limit and
// offset.
func (i *IterateQuery) Limit(limit, offset int) *IterateQuery {
	i.lim = limit
	i.off = offset
	return i
}

// Run executes the 'SELECT' query.
func (i *IterateQuery) Run() (*Iterator, Error) {
	logger := i.db.GetLogger()
	query := i.db.getUnderlying().NoAutoCondition().Table(i.bean.TableName())

	for _, cond := range i.conds {
		query.Where(builder.Expr(cond.sql, cond.args...))
	}

	if i.lim != 0 || i.off != 0 {
		query.Limit(i.lim, i.off)
	}
	if i.order != "" {
		if i.asc {
			query.OrderBy(fmt.Sprintf("%s ASC", i.order))
		} else {
			query.OrderBy(fmt.Sprintf("%s DESC", i.order))
		}
	}

	if len(i.distinct) > 0 {
		query.Distinct(i.distinct...)
	}

	rows, err := query.Rows(i.bean)
	logSQL(query, logger)

	if err != nil {
		logger.Errorf("Failed to retrieve the %s entries: %s", i.bean.Appellation(), err)
		return nil, NewInternalError(err)
	}

	return &Iterator{Rows: rows}, nil
}
