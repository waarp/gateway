package migration

type pk struct{}
type fk struct{ table, col string }
type notNull struct{}
type autoIncr struct{}
type unique struct{}
type defaul struct{ val interface{} }

func defaultFn(val interface{}) defaul { return defaul{val: val} }
func fkFn(table, col string) fk        { return fk{table: table, col: col} }

// The different types of column constraints usable in a CREATE TABLE statement.
var (
	// PrimaryKey declares the column as the table's primary key. Only 1 primary
	// key is allowed per table. For multi-column primary keys, use table constraints
	// instead.
	PrimaryKey = pk{}

	// ForeignKey declares the column as a foreign key referencing the given table.
	ForeignKey = fkFn

	// NotNull adds a 'not null' constraint to the column.
	NotNull = notNull{}

	// AutoIncr adds an auto-increment to the column. Only works on columns with
	// type TinyInt, SmallInt, Integer & BigInt.
	AutoIncr = autoIncr{}

	// Unique adds a 'unique' constraint to the column. To place a 'unique'
	// constraint on multiple columns, use table constraints instead.
	Unique = unique{}

	// Default adds a default value to the column. The value should be given as
	// parameter of the constraint (ex: Default(0))
	Default = defaultFn
)

type tblPk struct{ cols []string }
type tblUnique struct{ cols []string }

func pkFn(cols ...string) tblPk         { return tblPk{cols: cols} }
func uniqueFn(cols ...string) tblUnique { return tblUnique{cols: cols} }

// The different types of table constraints usable in a CREATE TABLE statement.
var (
	// MultiPrimaryKey adds a primary-key constraint to the given columns.
	MultiPrimaryKey = pkFn

	// MultiUnique adds a 'unique' constraint to the given columns.
	MultiUnique = uniqueFn
)
