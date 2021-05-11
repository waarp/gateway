package migration

import (
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

func (s *standardSQL) addRow(format func(interface{}, sqlType) (string, error), table string,
	values Cells) error {
	var colList, valuesList []string
	for col, cell := range values {
		str, err := format(cell.Val, cell.Type)
		if err != nil {
			return err
		}
		colList = append(colList, col)
		valuesList = append(valuesList, str)
	}
	_, err := s.Exec("INSERT INTO %s (%s) VALUES (%s)", table,
		strings.Join(colList, ", "), strings.Join(valuesList, ", "))
	return err
}

func (s *standardSQL) AddRow(table string, values Cells) error {
	return s.addRow(s.formatValueToSQL, table, values)
}
