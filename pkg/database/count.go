package database

import "xorm.io/builder"

// CountQuery is the type representing a SQL COUNT statement.
type CountQuery struct {
	db   Access
	bean IterateBean

	conds []*condition
}

// Where adds a 'WHERE' clause to the 'COUNT' query with the given conditions
// and arguments. The function uses the `?` character as verb for the arguments
// in the given string.
//
// If the function is called multiple times, all the conditions will be chained
// using the 'AND' operator.
func (c *CountQuery) Where(sql string, args ...any) *CountQuery {
	c.conds = append(c.conds, &condition{sql: sql, args: args})

	return c
}

// Run executes the 'COUNT' query and returns the count number.
func (c *CountQuery) Run() (uint64, error) {
	logger := c.db.GetLogger()
	query := c.db.getUnderlying().NoAutoCondition()

	for _, cond := range c.conds {
		query.Where(builder.Expr(cond.sql, cond.args...))
	}

	n, err := query.Count(c.bean)
	if err != nil {
		logger.Errorf("Failed to insert the new %s entry: %v", c.bean.Appellation(), err)

		return 0, NewInternalError(err)
	}

	return uint64(n), nil
}
