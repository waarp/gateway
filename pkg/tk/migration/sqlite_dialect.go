package migration

import (
	"errors"
	"fmt"
	"strings"

	// Register the SQLite driver.
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/mod/semver"
)

var errSqliteVersion = errors.New("cannot get sqlite version")

// SQLite is the constant name of the SQLite dialect translator.
const SQLite = "sqlite"

//nolint:gochecknoinits // init is used by design
func init() {
	dialects[SQLite] = newSqliteEngine
}

type sqliteError string

func (s sqliteError) Error() string { return string(s) }

type sqliteTranslator struct{ standardTranslator }

func (*sqliteTranslator) booleanType() string       { return "INTEGER" } //nolint:goconst // unnecessary here
func (*sqliteTranslator) tinyIntType() string       { return "INTEGER" }
func (*sqliteTranslator) smallIntType() string      { return "INTEGER" }
func (*sqliteTranslator) bigIntType() string        { return "INTEGER" }
func (*sqliteTranslator) floatType() string         { return "REAL" }
func (*sqliteTranslator) doubleType() string        { return "REAL" }
func (*sqliteTranslator) varCharType(uint64) string { return "TEXT" }
func (*sqliteTranslator) dateType() string          { return "NUMERIC" }
func (*sqliteTranslator) timeStampType() string     { return "NUMERIC" }
func (*sqliteTranslator) timeStampZType() string    { return "TEXT" }
func (*sqliteTranslator) binaryType(uint64) string  { return "BLOB" }

func (*sqliteTranslator) makeAutoIncrement(builder *tableBuilder, colType sqlType) error {
	if !isIntegerType(colType) {
		return fmt.Errorf("auto-increments can only be used on "+
			"integer types (%s is not an integer type): %w",
			colType.code.String(), errBadConstraint)
	}

	builder.getLastCol().addConstraint("AUTOINCREMENT")

	return nil
}

// sqliteActions is the dialect engine for SQLite.
type sqliteActions struct {
	*standardSQL
	trad *sqliteTranslator
}

func newSqliteEngine(db *queryWriter) Actions {
	return &sqliteActions{
		standardSQL: &standardSQL{queryWriter: db},
		trad:        &sqliteTranslator{},
	}
}

func (s *sqliteActions) isAtLeastVersion(target string) (bool, error) {
	res, err := s.Query("SELECT sqlite_version()")
	if err != nil {
		return false, fmt.Errorf("failed to run a query to retrieve SQLite version: %w", err)
	}

	defer res.Close() //nolint:errcheck // no logger to handle the error

	var version string
	if !res.Next() || res.Scan(&version) != nil {
		return false, fmt.Errorf("failed to retrieve SQLite version: %w", errSqliteVersion)
	}

	version = "v" + version
	if !semver.IsValid(version) {
		return false, fmt.Errorf("failed to parse SQLite version: '%s' is not a valid version: %w",
			version, err)
	}

	return semver.Compare(version, target) >= 0, nil
}

func (*sqliteActions) GetDialect() string { return SQLite }

func (s *sqliteActions) CreateTable(table string, defs ...Definition) error {
	return s.standardSQL.createTable(s.trad, table, defs)
}

func (s *sqliteActions) ChangeColumnType(_, _ string, from, to sqlType) error {
	if from.canConvertTo(to) {
		return nil // nothing to do
	}

	return fmt.Errorf("cannot convert from type %s to type %s: %w",
		from.code.String(), to.code.String(), errOperation)
}

func (s *sqliteActions) AddRow(table string, values Cells) error {
	return s.addRow(s.trad, table, values)
}

func (s *sqliteActions) AddColumn(table, column string, dataType sqlType,
	constraints ...Constraint) error {
	return s.standardSQL.addColumn(s.trad, table, column, dataType, constraints)
}

func (s *sqliteActions) DropColumn(table, name string) error {
	ok, err := s.isAtLeastVersion("v3.35.0")
	if err != nil {
		return err
	}

	// DROP COLUMN is supported on SQLite versions >= 3.35
	if !isTest && ok {
		return s.standardSQL.DropColumn(table, name)
	}
	// otherwise, we have to create a new table without the column, and copy the content

	cols, err := getColumnsNames(s, table)
	if err != nil {
		return err
	}

	var found bool

	for i, col := range cols {
		if col == name {
			cols = append(cols[:i], cols[i+1:]...)
			found = true

			break
		}
	}

	if !found {
		return sqliteError(fmt.Sprintf(`no such column: "%s"`, name))
	}

	query := fmt.Sprintf("CREATE TABLE %s_new AS SELECT %s FROM %s", table,
		strings.Join(cols, ", "), table)

	if err := s.Exec(query); err != nil {
		return err
	}

	if err := s.DropTable(table); err != nil {
		return err
	}

	if err := s.RenameTable(table+"_new", table); err != nil {
		return err
	}

	return nil
}
