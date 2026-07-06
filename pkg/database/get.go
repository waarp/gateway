package database

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// GetBean is the interface that a model must implement in order to be usable
// with the DB.Get function.
//
//nolint:iface //best keep generic table interface separate from bean interfaces
type GetBean interface {
	Table
}

// GetQuery is the type representing a SQL SELECT statement for a single entry.
type GetQuery struct {
	db   Access
	bean GetBean

	conds []*condition
	order string
	asc   bool
	eager bool
	all   bool
}

func (g *GetQuery) Eager() *GetQuery {
	g.eager = true
	return g
}

func (g *GetQuery) All() *GetQuery {
	g.all = true
	return g
}

func (g *GetQuery) And(sql string, args ...any) *GetQuery {
	g.conds = append(g.conds, &condition{sql: sql, args: args})

	return g
}

// In add a 'WHERE col IN' condition to the 'SELECT' query. Because the database/sql
// package cannot handle variadic placeholders in the Where function, a separate
// method is required.
func (g *GetQuery) In(col string, vals ...any) *GetQuery {
	g.conds = append(g.conds, makeInClause(col, vals...))
	return g
}

// OrderBy adds an 'ORDER BY' clause to the 'SELECT' query with the given order
// and direction.
func (g *GetQuery) OrderBy(order string, asc bool) *GetQuery {
	g.order = order
	g.asc = asc

	return g
}

// Deprecated: condition is automatic now.
func (g *GetQuery) Owner() *GetQuery { return g }

// Run executes the 'GET' query.
func (g *GetQuery) Run() error {
	logger := g.db.getLogger()
	query := g.db.getUnderlying().Table(g.bean.TableName())
	addOwnerCond(query, g.all, g.bean, g.db.getOwner())

	for _, cond := range g.conds {
		if cond.sql != "" {
			query.Where(cond.sql, cond.args...)
		}
	}

	if g.order != "" {
		if g.asc {
			query.Order(fmt.Sprintf("%s ASC", g.order))
		} else {
			query.Order(fmt.Sprintf("%s DESC", g.order))
		}
	}

	addPreloads(g.eager, query, g.bean)

	result := query.Take(g.bean)
	if getErr := result.Error; errors.Is(result.Error, gorm.ErrRecordNotFound) {
		logger.Debugf("No %s found with conditions (%s)", g.bean.Appellation(),
			explainStmt(query))

		return NewNotFoundError(g.bean)
	} else if getErr != nil {
		logger.Errorf("Failed to retrieve the %s entry: %v", g.bean.Appellation(), getErr)

		return NewInternalError(getErr)
	}

	if callBack, ok := g.bean.(ReadCallback); ok {
		if err := callBack.AfterRead(g.db); err != nil {
			logger.Errorf("%s entry GET callback failed: %v", g.bean.Appellation(), err)

			return fmt.Errorf("%s entry GET callback failed: %w", g.bean.Appellation(), err)
		}
	}

	return nil
}
