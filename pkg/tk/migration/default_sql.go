package migration

import (
	"fmt"
	"strings"
)

type translator interface {
	typer
	constraintMaker
	valueFormatter
}

type standardTranslator struct {
	standardFormatter
	standardConstraintMaker
	standardTyper
}

// standardSQL is the dialect engine for standard SQL. Other dialect engines should
// use this one as a base, and overwrite only the parts needed.
type standardSQL struct {
	*queryWriter
}

func (s *standardSQL) RenameTable(oldName, newName string) error {
	query := "ALTER TABLE %s RENAME TO %s"

	return s.Exec(query, oldName, newName)
}

func (s *standardSQL) DropTable(name string) error {
	return s.Exec("DROP TABLE %s", name)
}

func (s *standardSQL) RenameColumn(table, oldName, newName string) error {
	query := "ALTER TABLE %s RENAME COLUMN %s TO %s"

	return s.Exec(query, table, oldName, newName)
}

func (s *standardSQL) DropColumn(table, name string) error {
	query := "ALTER TABLE %s DROP COLUMN %s"

	return s.Exec(query, table, name)
}

func (s *standardSQL) addRow(conv valueFormatter, table string,
	values Cells) error {
	var colList, valuesList []string

	for col, cell := range values { //nolint:gocritic // FIXME to be refactored
		str, err := formatValue(cell.Val, cell.Type, conv)
		if err != nil {
			return fmt.Errorf("cannot format value to SQL: %w", err)
		}

		colList = append(colList, col)
		valuesList = append(valuesList, str)
	}

	return s.Exec("INSERT INTO %s (%s)\n VALUES (%s)", table,
		strings.Join(colList, ", "), strings.Join(valuesList, ", "))
}

func (s *standardSQL) createTable(trad translator, table string, defs []Definition) error {
	maker := &tableMaker{
		querier: s.queryWriter,
		name:    table,
		defs:    defs,
		trad:    trad,
	}

	return maker.makeTable()
}

func (s *standardSQL) addColumn(trad translator, table, column string, typ sqlType, cons []Constraint) error {
	maker := &tableMaker{name: table, trad: trad}
	builder := &tableBuilder{tableName: table}

	if err := maker.makeColumn(&Column{column, typ, cons}, builder); err != nil {
		return err
	}

	statements := builder.buildAddColumn()
	for i := range statements {
		if err := s.queryWriter.Exec(statements[i]); err != nil {
			return err
		}
	}

	return nil
}
