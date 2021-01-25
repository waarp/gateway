package database

import (
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

	sql  string
	args []interface{}
}

// Run executes the 'GET' query.
func (g *GetQuery) Run() Error {
	logger := g.db.GetLogger()

	query := g.db.getUnderlying().NoAutoCondition().Table(g.bean.TableName()).
		Where(g.sql, g.args...).Cols("*")
	exist, err := query.Get(g.bean)
	logSQL(query, logger)

	if err != nil {
		logger.Errorf("Failed to retrieve the %s entry: %s", g.bean.Appellation(), err)
		return NewInternalError(err)
	}
	if !exist {
		where, _ := builder.ConvertToBoundSQL(g.sql, g.args)
		logger.Infof("No %s found with conditions '%s'", g.bean.Appellation(), where)
		return NewNotFoundError(g.bean)
	}

	return nil
}
