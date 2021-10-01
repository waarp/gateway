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

// mySQLDialect is the dialect engine for SQLite.
type mySQLDialect struct{ *standardSQL }

func newMySQLEngine(db *queryWriter) Actions {
	return &mySQLDialect{standardSQL: &standardSQL{queryWriter: db}}
}

func (*mySQLDialect) GetDialect() string { return MySQL }

func (m *mySQLDialect) sqlTypeToDBType(typ sqlType) (string, error) {
	switch typ.code {
	case boolean, tinyint:
		return "TINYINT", nil
	case smallint:
		return "SMALLINT", nil
	case integer:
		return "INT", nil
	case bigint:
		return "BIGINT", nil
	case float:
		return "FLOAT", nil
	case double:
		return "DOUBLE", nil
	case varchar:
		return fmt.Sprintf("VARCHAR(%d)", typ.size), nil
	case text:
		return "TEXT", nil //nolint:goconst // no need here
	case date:
		return "DATE", nil
	case timestamp:
		return "TIMESTAMP", nil
	case timestampz:
		return "TEXT", nil
	case binary:
		return fmt.Sprintf("BINARY(%d)", typ.size), nil
	case blob:
		return "BLOB", nil
	default:
		return "", fmt.Errorf("unsupported SQL datatype") //nolint:goerr113 // base error
	}
}

//nolint:dupl // FIXME to be factorized
func (m *mySQLDialect) makeConstraints(col *Column) ([]string, error) {
	var consList []string

	for _, c := range col.Constraints {
		switch con := c.(type) {
		case pk:
			consList = append(consList, "PRIMARY KEY")
		case fk:
			consList = append(consList, fmt.Sprintf("REFERENCES %s(%s)", con.table, con.col))
		case notNull:
			consList = append(consList, "NOT NULL")
		case autoIncr:
			if !isIntegerType(col.Type) {
				return nil, fmt.Errorf("auto-increments can only be used on "+
					"integer types (%s is not an integer type): %w",
					col.Type.code.String(), errBadConstraint)
			}

			consList = append(consList, "AUTO_INCREMENT")

		case unique:
			consList = append(consList, "UNIQUE")
		case defaul:
			sqlVal, err := m.formatValueToSQL(con.val, col.Type)
			if err != nil {
				return nil, err
			}

			consList = append(consList, fmt.Sprintf("DEFAULT %s", sqlVal))

		default:
			return nil, fmt.Errorf("unknown constraint type %T: %w", c, errBadConstraint)
		}
	}

	return consList, nil
}

func (m *mySQLDialect) CreateTable(table string, defs ...Definition) error {
	return m.standardSQL.createTable(m, table, defs)
}

func (m *mySQLDialect) AddColumn(table, column string, dataType sqlType,
	constraints ...Constraint) error {
	return m.standardSQL.addColumn(m, table, column, dataType, constraints)
}

func (m *mySQLDialect) ChangeColumnType(table, col string, from, to sqlType) error {
	if !from.canConvertTo(to) {
		return fmt.Errorf("cannot convert from type %s to type %s: %w",
			from.code.String(), to.code.String(), errOperation)
	}

	newType, err := m.sqlTypeToDBType(to)
	if err != nil {
		return err
	}

	query := "ALTER TABLE %s\nMODIFY COLUMN %s %s"

	return m.Exec(query, table, col, newType)
}

func (m *mySQLDialect) AddRow(table string, values Cells) error {
	return m.addRow(m, table, values)
}
