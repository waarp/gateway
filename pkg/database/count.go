package database

type selectCountBean struct{ SelectBean }

func (s *selectCountBean) Appellation() string { return s.Elem() }

// CountQuery is the type representing a SQL COUNT statement.
type CountQuery struct {
	db   Access
	bean IterateBean

	conds []*condition
	all   bool
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

func (c *CountQuery) All() *CountQuery {
	c.all = true
	return c
}

// Run executes the 'COUNT' query and returns the count number.
func (c *CountQuery) Run() (uint64, error) {
	logger := c.db.getLogger()
	query := c.db.getUnderlying().Table(c.bean.TableName())
	addOwnerCond(query, c.all, c.bean, c.db.getOwner())

	for _, cond := range c.conds {
		query.Where(cond.sql, cond.args...)
	}

	var n int64
	if err := query.Count(&n).Error; err != nil {
		logger.Errorf("Failed to count the %s entries: %v", c.bean.Appellation(), err)

		return 0, NewInternalError(err)
	}

	return uint64(n), nil
}
