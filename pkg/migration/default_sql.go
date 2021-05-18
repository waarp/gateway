package migration

import (
	"fmt"
	"strings"
)

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

func (s *standardSQL) AddColumn(table, column, datatype string) error {
	query := "ALTER TABLE %s ADD COLUMN %s %s"
	_, err := s.Exec(query, table, column, datatype)
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

func (s *standardSQL) makeTblConstraint(colsDef []Column, cons TableConstraint) (string, error) {
	checkCol := func(col string) error {
		for _, def := range colsDef {
			if def.Name == col {
				return nil
			}
		}
		return fmt.Errorf("column %s does not exist", col)
	}

	checkCols := func(cols []string) error {
		for _, col := range cols {
			if err := checkCol(col); err != nil {
				return err
			}
		}
		return nil
	}

	switch con := cons.(type) {
	case tblPk:
		if err := checkCols(con.cols); err != nil {
			return "", err
		}
		return fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(con.cols, ", ")), nil
	case tblUnique:
		if err := checkCols(con.cols); err != nil {
			return "", err
		}
		return fmt.Sprintf("UNIQUE (%s)", strings.Join(con.cols, ", ")), nil
	default:
		return "", fmt.Errorf("invalid table definition %#v", con)
	}
}

func (s *standardSQL) createTable(formatter sqlFormatter, table string, defs []Definition) error {

	var colDefs []Column
	var constrDefs []TableConstraint
	for _, d := range defs {
		switch def := d.(type) {
		case Column:
			colDefs = append(colDefs, def)
		case TableConstraint:
			constrDefs = append(constrDefs, def)
		}
	}

	var defsStr []string
	for _, col := range colDefs {
		str, err := s.makeColumnDef(formatter, col)
		if err != nil {
			return err
		}
		defsStr = append(defsStr, str)
	}
	for _, con := range constrDefs {
		str, err := s.makeTblConstraint(colDefs, con)
		if err != nil {
			return err
		}
		defsStr = append(defsStr, str)
	}
	if len(defsStr) == 0 {
		return fmt.Errorf("cannot create a table without columns")
	}

	var err error
	if len(defsStr) > 1 {
		_, err = s.Exec("CREATE TABLE %s (\n    %s\n)", table, strings.Join(defsStr, ",\n    "))
	} else {
		_, err = s.Exec("CREATE TABLE %s (%s)", table, defsStr[0])
	}

	return err
}
