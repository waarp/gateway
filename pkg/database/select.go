package database

import (
	"fmt"

	"gorm.io/gorm/clause"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// SelectBean is the interface that a model must implement in order to be
// selectable via the Select query builder. The model MUST be a slice.
type SelectBean interface {
	// TableName returns the table's name.
	TableName() string
	// Elem returns the display name for a single table row.
	Elem() string
}

// SelectQuery is the type representing a SQL SELECT statement. The values are
// returned inside the given bean (which must be a slice).
type SelectQuery struct {
	db   Access
	bean SelectBean

	lim, off int
	conds    []*condition
	distinct []string
	order    string
	asc      bool
	forUpd   bool
	eager    bool
	all      bool
}

func (s *SelectQuery) Eager() *SelectQuery {
	s.eager = true
	return s
}

func (s *SelectQuery) All() *SelectQuery {
	s.all = true
	return s
}

// Where adds a 'WHERE' clause to the 'SELECT' query with the given conditions
// and arguments. The function uses the `?` character as verb for the arguments
// in the given string.
//
// If the function is called multiple times, all the conditions will be chained
// using the 'AND' operator.
func (s *SelectQuery) Where(sql string, args ...any) *SelectQuery {
	s.conds = append(s.conds, &condition{sql: sql, args: args})

	return s
}

// Deprecated: condition is automatic now.
func (s *SelectQuery) Owner() *SelectQuery { return s }

// In add a 'WHERE col IN' condition to the 'SELECT' query. Because the database/sql
// package cannot handle variadic placeholders in the Where function, a separate
// method is required.
func (s *SelectQuery) In(col string, vals ...any) *SelectQuery {
	s.conds = append(s.conds, makeInClause(col, vals...))
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

func (s *SelectQuery) Count() (uint64, error) {
	countQuery := &CountQuery{db: s.db, bean: &selectCountBean{s.bean}, conds: s.conds}

	return countQuery.Run()
}

// Run executes the 'SELECT' query.
func (s *SelectQuery) Run() error {
	logger := s.db.getLogger()
	query := s.db.getUnderlying().Table(s.bean.TableName())
	addOwnerCond(query, s.all, s.bean, s.db.getOwner())

	for _, cond := range s.conds {
		query.Where(cond.sql, cond.args...)
	}

	if s.lim != 0 {
		query.Limit(s.lim)
	}

	if s.off != 0 {
		query.Offset(s.off)
	}

	if s.order != "" {
		if s.asc {
			query.Order(fmt.Sprintf("%s ASC", s.order))
		} else {
			query.Order(fmt.Sprintf("%s DESC", s.order))
		}
	}

	if len(s.distinct) > 0 {
		query.Distinct(utils.AsAny(s.distinct)...)
	}

	if s.forUpd {
		query.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate})
	}

	addPreloads(s.eager, query, s.bean)

	if err := query.Find(s.bean).Error; err != nil {
		logger.Errorf("Failed to retrieve the %s entries: %v", s.bean.Elem(), err)

		return NewInternalError(err)
	}

	if callBack, ok := s.bean.(ReadCallback); ok {
		if err := callBack.AfterRead(s.db); err != nil {
			logger.Errorf("%s entry SELECT callback failed: %v", s.bean.Elem(), err)

			return fmt.Errorf("%s entry GET callback failed: %w", s.bean.Elem(), err)
		}
	}

	return nil
}
