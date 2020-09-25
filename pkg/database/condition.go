package database

import (
	"github.com/go-xorm/builder"
)

// Condition is the interface representing an SQL condition which can be used in
// a 'WHERE' statement. Different types of conditions can be generated using
// the functions defined bellow.
type Condition interface {
	convert() builder.Cond
	And(...Condition) Condition
	Or(...Condition) Condition
}

// And generates an SQL 'AND' condition with all the given conditions.
func And(conds ...Condition) Condition {
	return &andCond{conds: conds}
}

// Or generates an SQL 'OR' condition with all the given conditions.
func Or(conds ...Condition) Condition {
	return &orCond{conds: conds}
}

// In generates an SQL 'IN' condition.
func In(col string, vals ...interface{}) Condition {
	return &inCond{col: col, vals: vals}
}

// Equal generates an SQL `=` condition.
func Equal(col string, val interface{}) Condition {
	return &eqCond{col: col, val: val}
}

// NotEqual generates an SQL `<>` condition.
func NotEqual(col string, val interface{}) Condition {
	return &neqCond{col: col, val: val}
}

// GreaterThan generates an SQL `>` condition.
func GreaterThan(col string, val interface{}) Condition {
	return &gtCond{col: col, val: val}
}

// GreaterThanOrEqual generates an SQL `>=` condition.
func GreaterThanOrEqual(col string, val interface{}) Condition {
	return &gteCond{col: col, val: val}
}

// LowerThan generates an SQL `<` condition.
func LowerThan(col string, val interface{}) Condition {
	return &ltCond{col: col, val: val}
}

// LowerThanOrEqual generates an SQL `<=` condition.
func LowerThanOrEqual(col string, val interface{}) Condition {
	return &lteCond{col: col, val: val}
}

func convertConditions(cs []Condition) []builder.Cond {
	conds := make([]builder.Cond, len(cs))
	for i := range cs {
		conds[i] = cs[i].convert()
	}
	return conds
}

type eqCond struct {
	col string
	val interface{}
}

func (e *eqCond) convert() builder.Cond         { return builder.Eq{e.col: e.val} }
func (e *eqCond) And(cs ...Condition) Condition { return And(e, And(cs...)) }
func (e *eqCond) Or(cs ...Condition) Condition  { return Or(e, Or(cs...)) }

type neqCond struct {
	col string
	val interface{}
}

func (n *neqCond) convert() builder.Cond         { return builder.Neq{n.col: n.val} }
func (n *neqCond) And(cs ...Condition) Condition { return And(n, And(cs...)) }
func (n *neqCond) Or(cs ...Condition) Condition  { return Or(n, Or(cs...)) }

type gtCond struct {
	col string
	val interface{}
}

func (g *gtCond) convert() builder.Cond         { return builder.Gt{g.col: g.val} }
func (g *gtCond) And(cs ...Condition) Condition { return And(g, And(cs...)) }
func (g *gtCond) Or(cs ...Condition) Condition  { return Or(g, Or(cs...)) }

type gteCond struct {
	col string
	val interface{}
}

func (g *gteCond) convert() builder.Cond         { return builder.Gte{g.col: g.val} }
func (g *gteCond) And(cs ...Condition) Condition { return And(g, And(cs...)) }
func (g *gteCond) Or(cs ...Condition) Condition  { return Or(g, Or(cs...)) }

type ltCond struct {
	col string
	val interface{}
}

func (l *ltCond) convert() builder.Cond         { return builder.Lt{l.col: l.val} }
func (l *ltCond) And(cs ...Condition) Condition { return And(l, And(cs...)) }
func (l *ltCond) Or(cs ...Condition) Condition  { return Or(l, Or(cs...)) }

type lteCond struct {
	col string
	val interface{}
}

func (l *lteCond) convert() builder.Cond         { return builder.Lte{l.col: l.val} }
func (l *lteCond) And(cs ...Condition) Condition { return And(l, And(cs...)) }
func (l *lteCond) Or(cs ...Condition) Condition  { return Or(l, Or(cs...)) }

type andCond struct {
	conds []Condition
}

func (a *andCond) convert() builder.Cond         { return builder.And(convertConditions(a.conds)...) }
func (a *andCond) And(cs ...Condition) Condition { return And(a, And(cs...)) }
func (a *andCond) Or(cs ...Condition) Condition  { return Or(a, Or(cs...)) }

type orCond struct {
	conds []Condition
}

func (o *orCond) convert() builder.Cond         { return builder.Or(convertConditions(o.conds)...) }
func (o *orCond) And(cs ...Condition) Condition { return And(o, And(cs...)) }
func (o *orCond) Or(cs ...Condition) Condition  { return Or(o, Or(cs...)) }

type inCond struct {
	col  string
	vals []interface{}
}

func (i *inCond) convert() builder.Cond {
	if e, ok := i.vals[0].(*expr); ok {
		return builder.In(i.col, e.convert())
	}
	return builder.In(i.col, i.vals...)
}
func (i *inCond) And(cs ...Condition) Condition { return And(i, And(cs...)) }
func (i *inCond) Or(cs ...Condition) Condition  { return Or(i, Or(cs...)) }

// Expr can be used with a 'IN' condition when the desired values are the
// result of an SQL query (typically a SELECT).
// The said query can be entered directly with this function, and then given to
// the `In` function. For example:
//
//  database.Select(&customer).Where(database.In("id", database.Expr(
//  	"SELECT DISTINCT cust_id FROM orders WHERE item_id=?", 3)))
//
// will retrieve all customers who have ordered the item with the ID n°3.
//
// The function uses an fmt-style format to provide arguments to the query,
// with the character '?' as a verb
//
//nolint:golint //This exported function returns an unexported type on purpose
//so that instances of expr cannot be created outside of this function
func Expr(str string, args ...interface{}) *expr {
	return &expr{str: str, args: args}
}

type expr struct {
	str  string
	args []interface{}
}

func (e *expr) convert() builder.Cond {
	return builder.Expr(e.str, e.args...)
}
