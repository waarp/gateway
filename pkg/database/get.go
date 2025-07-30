package database

import (
	"fmt"
	"strings"

	"xorm.io/builder"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
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
}

func (g *GetQuery) And(sql string, args ...any) *GetQuery {
	g.conds = append(g.conds, &condition{sql: sql, args: args})

	return g
}

// OrderBy adds an 'ORDER BY' clause to the 'SELECT' query with the given order
// and direction.
func (g *GetQuery) OrderBy(order string, asc bool) *GetQuery {
	g.order = order
	g.asc = asc

	return g
}

func (g *GetQuery) Owner() *GetQuery {
	return g.And("owner=?", conf.GlobalConfig.GatewayName)
}

// Run executes the 'GET' query.
func (g *GetQuery) Run() error {
	logger := g.db.GetLogger()
	query := g.db.getUnderlying().NoAutoCondition().Table(g.bean.TableName())

	condsStr := make([]string, len(g.conds))

	for i, cond := range g.conds {
		query.And(cond.sql, cond.args...)

		var err error
		if condsStr[i], err = builder.ConvertToBoundSQL(cond.sql, cond.args); err != nil {
			logger.Debugf("Failed to serialize the SQL condition: %v", err)
		}
	}

	if g.order != "" {
		if g.asc {
			query.OrderBy(fmt.Sprintf("%s ASC", g.order))
		} else {
			query.OrderBy(fmt.Sprintf("%s DESC", g.order))
		}
	}

	exist, getErr := query.Get(g.bean)
	if getErr != nil {
		logger.Errorf("Failed to retrieve the %s entry: %v", g.bean.Appellation(), getErr)

		return NewInternalError(getErr)
	}

	if !exist {
		where := strings.Join(condsStr, " AND ")

		logger.Debugf("No %s found with conditions (%s)", g.bean.Appellation(), where)

		return NewNotFoundError(g.bean)
	}

	if callBack, ok := g.bean.(ReadCallback); ok {
		if err := callBack.AfterRead(g.db); err != nil {
			logger.Errorf("%s entry GET callback failed: %v", g.bean.Appellation(), err)

			return fmt.Errorf("%s entry GET callback failed: %w", g.bean.Appellation(), err)
		}
	}

	return nil
}
