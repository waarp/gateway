package database

import "gorm.io/gorm/clause"

// UpdateAllBean is the interface that a model must implement in order to be
// updatable at once via the UpdateAll query builder: a table, whose entries
// receive the named columns of the bean.
type UpdateAllBean interface {
	Table
}

// UpdateAllQuery is the type representing a SQL UPDATE statement over the
// entries matching its conditions: every one of them receives the named
// columns of the bean, in one statement. The hooks of the model are not
// called, the entries are not read again: the caller has checked them one by
// one, and names the columns it writes (Cols) and the entries it writes to
// (Where, In), both mandatory.
type UpdateAllQuery struct {
	db   Access
	bean UpdateAllBean

	cols  []string
	conds []*condition
	all   bool
}

func (u *UpdateAllQuery) All() *UpdateAllQuery {
	u.all = true

	return u
}

// Cols names the columns written to the entries, from the fields of the
// bean. An update without any column is refused.
func (u *UpdateAllQuery) Cols(columns ...string) *UpdateAllQuery {
	u.cols = append(u.cols, columns...)

	return u
}

// Where adds a 'WHERE' clause to the 'UPDATE' query with the given conditions
// and arguments. The function uses the `?` character as verb for the arguments
// in the given string.
//
// If the function is called multiple times, all the conditions will be chained
// using the 'AND' operator.
func (u *UpdateAllQuery) Where(sql string, args ...any) *UpdateAllQuery {
	u.conds = append(u.conds, &condition{sql: sql, args: args})

	return u
}

// In adds a 'WHERE col IN' condition to the 'UPDATE' query. Because the
// database/sql package cannot handle variadic placeholders in the Where
// function, a separate method is required. Without any value, no condition is
// added: an update of the whole table from a bean is then refused by Run.
func (u *UpdateAllQuery) In(col string, vals ...any) *UpdateAllQuery {
	if len(vals) > 0 {
		u.conds = append(u.conds, makeInClause(col, vals...))
	}

	return u
}

// Run executes the 'UPDATE' query. It refuses an update without a column or
// without a condition: every entry of the table would otherwise receive the
// same values.
func (u *UpdateAllQuery) Run() error {
	logger := u.db.getLogger()

	if len(u.cols) == 0 {
		logger.Errorf("Failed to update the %s entries: no column named", u.bean.Appellation())

		return NewValidationError("an update of several entries names the columns it writes")
	}

	if len(u.conds) == 0 {
		logger.Errorf("Failed to update the %s entries: no condition given", u.bean.Appellation())

		return NewValidationError("an update of several entries needs a condition")
	}

	engine := u.db.getUnderlying().Table(u.bean.TableName())
	addOwnerCond(engine, u.all, u.bean, u.db.getOwner())

	for _, cond := range u.conds {
		engine.Where(cond.sql, cond.args...)
	}

	// Associations are handled by the models' own hooks (see UpdateQuery).
	if err := engine.Select(u.cols).Omit(clause.Associations).Updates(u.bean).Error; err != nil {
		logger.Errorf("Failed to update the %s entries: %v", u.bean.Appellation(), err)

		return NewInternalError(err)
	}

	return nil
}
