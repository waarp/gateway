package database

import (
	"fmt"
	"strings"

	"github.com/go-xorm/builder"
)

// SelectBean is the interface that a model must implement in order to be
// selectable via the Select query builder. The model MUST be a slice.
type SelectBean interface {
	TableName() string
	Elem() string
}

// SelectQuery is the type representing a SQL SELECT statement. The values are
// returned inside the given bean (which must be a slice).
type SelectQuery struct {
	db   Access
	bean SelectBean

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
func (s *SelectQuery) Where(sql string, args ...interface{}) *SelectQuery {
	s.conds = append(s.conds, cond{sql: sql, args: args})
	return s
}

// In add a 'WHERE col IN' condition to the 'SELECT' query. Because the database/sql
// package cannot handle variadic placeholders in the Where function, a separate
// method is required.
func (s *SelectQuery) In(col string, vals ...interface{}) *SelectQuery {
	sql := &inCond{Builder: &strings.Builder{}}
	if builder.In(col, vals...).WriteTo(sql) != nil {
		return s
	}
	s.conds = append(s.conds, cond{sql: sql.String(), args: sql.args})
	return s
}

// Distinct is used to add a 'DISTINCT' clause to the 'SELECT' query on the
// specified columns. Be aware that if Distinct is called, only the specified
// columns will be retrieved from the database, all the others will be ignored.
//
// If the function is called multiple times, all the columns will be taken into
// account for the SELECT.
func (s *SelectQuery) Distinct(columns ...string) *SelectQuery {
	s.distinct = append(s.distinct, columns...)
	return s
}

// OrderBy adds an 'ORDER BY' clause to the 'SELECT' query with the given order
// and direction.
func (s *SelectQuery) OrderBy(order string, asc bool) *SelectQuery {
	s.order = order
	s.asc = asc
	return s
}

// Limit adds an 'LIMIT' clause to the 'SELECT' query with the given limit and
// offset.
func (s *SelectQuery) Limit(limit, offset int) *SelectQuery {
	s.lim = limit
	s.off = offset
	return s
}

// Run executes the 'SELECT' query.
func (s *SelectQuery) Run() Error {
	logger := s.db.GetLogger()
	query := s.db.getUnderlying().NoAutoCondition().Table(s.bean.TableName())

	for _, cond := range s.conds {
		query.Where(builder.Expr(cond.sql, cond.args...))
	}

	if s.lim != 0 || s.off != 0 {
		query.Limit(s.lim, s.off)
	}
	if s.order != "" {
		if s.asc {
			query.OrderBy(fmt.Sprintf("%s ASC", s.order))
		} else {
			query.OrderBy(fmt.Sprintf("%s DESC", s.order))
		}
	}

	if len(s.distinct) > 0 {
		query.Distinct(s.distinct...)
	} else {
		query.Cols("*")
	}

	err := query.Find(s.bean)
	logSQL(query, logger)

	if err != nil {
		logger.Errorf("Failed to retrieve the %s entries: %s", s.bean.Elem(), err)
		return NewInternalError(err)
	}

	return nil
}
