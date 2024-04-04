package database

import (
	"fmt"
	"strings"

	"xorm.io/builder"
)

// GetBean is the interface that a model must implement in order to be usable
// with the DB.Get function.
type GetBean interface {
	Table
}

// GetQuery is the type representing a SQL SELECT statement for a single entry.
type GetQuery struct {
	db   Access
	bean GetBean

	conds []*condition
}

func (g *GetQuery) And(sql string, args ...any) *GetQuery {
	g.conds = append(g.conds, &condition{sql: sql, args: args})

	return g
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
			logger.Debug("Failed to serialize the SQL condition: %v", err)
		}
	}

	exist, getErr := query.Get(g.bean)
	if getErr != nil {
		logger.Error("Failed to retrieve the %s entry: %s", g.bean.Appellation(), getErr)

		return NewInternalError(getErr)
	}

	if !exist {
		where := strings.Join(condsStr, " AND ")

		logger.Debug("No %s found with conditions (%s)", g.bean.Appellation(), where)

		return NewNotFoundError(g.bean)
	}

	if callBack, ok := g.bean.(ReadCallback); ok {
		if err := callBack.AfterRead(g.db); err != nil {
			logger.Error("%s entry GET callback failed: %s", g.bean.Appellation(), err)

			return fmt.Errorf("%s entry GET callback failed: %w", g.bean.Appellation(), err)
		}
	}

	return nil
}
