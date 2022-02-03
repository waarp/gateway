package migration

import (
	"fmt"

	// Register the MySQL driver.
	_ "github.com/go-sql-driver/mysql"
)

// MySQL is the constant name of the MySQL dialect translator.
const MySQL = "mysql"

//nolint:gochecknoinits // init is used by design
func init() {
	dialects[MySQL] = newMySQLEngine
}

type mysqlTranslator struct{ standardTranslator }

func (*mysqlTranslator) booleanType() string        { return "TINYINT" }
func (*mysqlTranslator) integerType() string        { return "INT" }
func (*mysqlTranslator) timeStampZType() string     { return "TEXT" } //nolint:goconst // unnecessary here
func (*mysqlTranslator) binaryType(s uint64) string { return fmt.Sprintf("BINARY(%d)", s) }

// mySQLActions is the dialect engine for SQLite.
type mySQLActions struct {
	*standardSQL
	trad *mysqlTranslator
}

func newMySQLEngine(db *queryWriter) Actions {
	return &mySQLActions{
		standardSQL: &standardSQL{queryWriter: db},
		trad:        &mysqlTranslator{},
	}
}

func (*mySQLActions) GetDialect() string { return MySQL }

func (m *mySQLActions) CreateTable(table string, defs ...Definition) error {
	return m.standardSQL.createTable(m.trad, table, defs)
}

func (m *mySQLActions) AddColumn(table, column string, dataType sqlType,
	constraints ...Constraint) error {
	return m.standardSQL.addColumn(m.trad, table, column, dataType, constraints)
}

func (m *mySQLActions) ChangeColumnType(table, col string, from, to sqlType) error {
	if !from.canConvertTo(to) {
		return fmt.Errorf("cannot convert from type %s to type %s: %w",
			from.code.String(), to.code.String(), errOperation)
	}

	newType, err := makeType(to, m.trad)
	if err != nil {
		return err
	}

	query := "ALTER TABLE %s\nMODIFY COLUMN %s %s"

	return m.Exec(query, table, col, newType)
}

func (m *mySQLActions) AddRow(table string, values Cells) error {
	return m.addRow(m.trad, table, values)
}

func (m *mySQLActions) SwapColumns(table, col1, col2, cond string) error {
	query := "UPDATE %s SET %s=(@temp:=%s), %s=%s, %s=@temp"
	if cond != "" {
		query += " WHERE " + cond
	}

	return m.Exec(query, table, col1, col1, col1, col2, col2)
}
