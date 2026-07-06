package database

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
	all   bool
}

func (d *DeleteAllQuery) All() *DeleteAllQuery {
	d.all = true
	return d
}

// Where adds a 'WHERE' clause to the 'DELETE' query with the given conditions
// and arguments. The function uses the `?` character as verb for the arguments
// in the given string.
//
// If the function is called multiple times, all the conditions will be chained
// using the 'AND' operator.
func (d *DeleteAllQuery) Where(sql string, args ...any) *DeleteAllQuery {
	d.conds = append(d.conds, &condition{sql: sql, args: args})

	return d
}

// Deprecated: condition is automatic now.
func (d *DeleteAllQuery) Owner() *DeleteAllQuery { return d }

// In add a 'WHERE col IN' condition to the 'DELETE' query. Because the database/sql
// package cannot handle variadic placeholders in the Where function, a separate
// method is required.
func (d *DeleteAllQuery) In(col string, vals ...any) *DeleteAllQuery {
	d.conds = append(d.conds, makeInClause(col, vals...))
	return d
}

// Run executes the 'DELETE ALL' query.
func (d *DeleteAllQuery) Run() error {
	logger := d.db.getLogger()
	engine := d.db.getUnderlying().Table(d.bean.TableName())
	addOwnerCond(engine, d.all, d.bean, d.db.getOwner())

	for _, cond := range d.conds {
		engine.Where(cond.sql, cond.args...)
	}

	if err := engine.Delete(nil).Error; err != nil {
		logger.Errorf("Failed to delete the %s entries: %v", d.bean.Appellation(), err)

		return NewInternalError(err)
	}

	return nil
}
