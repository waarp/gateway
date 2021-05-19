package migration

import (
	"fmt"
	"strings"
)

type sqlFormatter interface {
	formatValueToSQL(val interface{}, sqlTyp sqlType) (string, error)
	sqlTypeToDBType(sqlType sqlType) (string, error)
	makeConstraints(col *Column) ([]string, error)
}

// standardSQL is the dialect engine for standard SQL. Other dialect engines should
// use this one as a base, and overwrite only the parts needed.
type standardSQL struct {
	*queryWriter
}

func (s *standardSQL) RenameTable(oldName, newName string) error {
	query := "ALTER TABLE %s RENAME TO %s"
	_, err := s.Exec(query, oldName, newName)
	return err
}

func (s *standardSQL) DropTable(name string) error {
	_, err := s.Exec("DROP TABLE %s", name)
	return err
}

func (s *standardSQL) RenameColumn(table, oldName, newName string) error {
	query := "ALTER TABLE %s RENAME COLUMN %s TO %s"
	_, err := s.Exec(query, table, oldName, newName)
	return err
}

func (s *standardSQL) addColumn(form sqlFormatter, table, column string, typ sqlType, cons []Constraint) error {
	dbType, err := form.sqlTypeToDBType(typ)
	if err != nil {
		return err
	}
	c := Col(column, typ, cons...)
	consList, err := form.makeConstraints(&c)
	if err != nil {
		return err
	}

	query := "ALTER TABLE %s ADD COLUMN %s"
	def := append([]string{column, dbType}, consList...)
	_, err = s.Exec(query, table, strings.Join(def, " "))
	return err
}

func (s *standardSQL) DropColumn(table, name string) error {
	query := "ALTER TABLE %s DROP COLUMN %s"
	_, err := s.Exec(query, table, name)
	return err
}

func (s *standardSQL) addRow(conv sqlFormatter, table string,
	values Cells) error {
	var colList, valuesList []string
	for col, cell := range values {
		str, err := conv.formatValueToSQL(cell.Val, cell.Type)
		if err != nil {
			return err
		}
		colList = append(colList, col)
		valuesList = append(valuesList, str)
	}
	_, err := s.Exec("INSERT INTO %s (%s)\n VALUES (%s)", table,
		strings.Join(colList, ", "), strings.Join(valuesList, ", "))
	return err
}

func (s *standardSQL) makeColumnDef(formatter sqlFormatter, col Column) (string, error) {
	constr, err := formatter.makeConstraints(&col)
	if err != nil {
		return "", err
	}

	typ, err := formatter.sqlTypeToDBType(col.Type)
	if err != nil {
		return "", err
	}

	return strings.Join(append([]string{col.Name, typ}, constr...), " "), nil
}

func (s *standardSQL) makeTblConstraint(cons TableConstraint) (string, error) {
	switch con := cons.(type) {
	case tblPk:
		return fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(con.cols, ", ")), nil
	case tblUnique:
		return fmt.Sprintf("UNIQUE (%s)", strings.Join(con.cols, ", ")), nil
	default:
		return "", fmt.Errorf("invalid table definition %#v", con)
	}
}

func (s *standardSQL) createTable(formatter sqlFormatter, table string, defs []Definition) error {
	var colDefs []string
	var constrDefs []string
	for _, d := range defs {
		switch def := d.(type) {
		case Column:
			str, err := s.makeColumnDef(formatter, def)
			if err != nil {
				return err
			}
			colDefs = append(colDefs, str)
		case TableConstraint:
			str, err := s.makeTblConstraint(def)
			if err != nil {
				return err
			}
			constrDefs = append(constrDefs, str)
		}
	}

	if len(colDefs) == 0 {
		return fmt.Errorf("cannot create a table without columns")
	}
	defsStr := append(colDefs, constrDefs...)

	var err error
	if len(defsStr) > 1 {
		_, err = s.Exec("CREATE TABLE %s (\n    %s\n)", table, strings.Join(defsStr, ",\n    "))
	} else {
		_, err = s.Exec("CREATE TABLE %s (%s)", table, defsStr[0])
	}

	return err
}
