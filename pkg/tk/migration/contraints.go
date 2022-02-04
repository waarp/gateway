package migration

import "errors"

var errBadConstraint = errors.New("bad constraint")

// The different types of column constraints usable in a CREATE TABLE statement.
type (
	pk       struct{}
	fk       struct{ table, col string }
	notNull  struct{}
	autoIncr struct{}
	unique   struct{}
	defaul   struct{ val interface{} }
)

// PrimaryKey declares the column as the table's primary key. Only 1 primary
// key is allowed per table. For multi-column primary keys, use table constraints
// instead.
//nolint:gochecknoglobals // global var is used by design
var PrimaryKey Constraint = pk{}

// ForeignKey declares the column as a foreign key referencing the given table.
func ForeignKey(table, col string) Constraint { return fk{table: table, col: col} }

// NotNull adds a 'NOT NULL' constraint to the column.
//nolint:gochecknoglobals // global var is used by design
var NotNull Constraint = notNull{}

// AutoIncr adds an auto-increment to the column. Only works on columns with
// type TinyInt, SmallInt, Integer & BigInt.
//nolint:gochecknoglobals // global var is used by design
var AutoIncr Constraint = autoIncr{}

// Default adds a default value to the column. The value should be given as
// parameter of the constraint (ex: Default(0)).
func Default(val interface{}) Constraint { return defaul{val: val} }

// Unique adds a 'unique' constraint to the column. To place a 'unique'
// constraint on multiple columns, use table constraints instead.
//nolint:gochecknoglobals // global var is used by design
var Unique Constraint = unique{}

// The different types of table constraints usable in a CREATE TABLE statement.
type (
	tblPk     struct{ cols []string }
	tblUnique struct{ cols []string }
)

// PrimaryKeys adds a primary-key constraint to the given columns.
func PrimaryKeys(cols ...string) TableConstraint { return tblPk{cols: cols} }

// Uniques adds a 'unique' constraint to the given columns.
func Uniques(cols ...string) TableConstraint { return tblUnique{cols: cols} }
