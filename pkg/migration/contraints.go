package migration

type constraintKind uint8

const (
	_ constraintKind = iota
	primaryKey
	notNull
	autoIncr
	unique
	defaultVal
)

type constraint struct {
	kind   constraintKind
	params []interface{}
}

// The different types of constraints usable in a CREATE TABLE statement.
var (
	// PRIMARYKEY declares the column as the table's primary key. Only 1 primary
	// key is allowed per table.
	PRIMARYKEY = constraint{kind: primaryKey}

	// NOTNULL adds a 'not null' constraint to the column.
	NOTNULL = constraint{kind: notNull}

	// AUTOINCR adds an auto-increment to the column. Only works on columns with
	// type TINYINT, SMALLINT, INTEGER & BIGINT.
	AUTOINCR = constraint{kind: autoIncr}

	// UNIQUE adds a 'unique' constraint to the column. To place a 'unique'
	// constraint on multiple columns, add the names of these columns as parameters
	// of UNIQUE.
	//
	// For example, to add a 'unique' constraint on column c1 and c2, the declaration
	// is as follow:
	//
	//     Col("c1", TEXT, UNIQUE("c2"))
	//
	UNIQUE = uniqueFn

	// DEFAULT adds a default value to the column. The value should be given as
	// parameter of the constraint (ex: DEFAULT(0))
	DEFAULT = defaultFn
)

func uniqueFn(cols ...string) constraint {
	params := make([]interface{}, len(cols))
	for i := range cols {
		params[i] = cols[i]
	}
	return constraint{kind: unique, params: params}
}

func defaultFn(val interface{}) constraint {
	return constraint{kind: defaultVal, params: []interface{}{val}}
}
