package migration

import (
	"errors"
	"fmt"
	"strings"
)

var errInvalidDefinition = errors.New("invalid table definition")

type tableMaker struct {
	querier *queryWriter

	name string
	defs []Definition

	trad translator
}

func (t *tableMaker) makeTable() error {
	if len(t.defs) == 0 {
		return fmt.Errorf("%w: tables must have at least 1 column", errInvalidDefinition)
	}

	builder := &tableBuilder{tableName: t.name}

	for i := range t.defs {
		switch def := t.defs[i].(type) {
		case Column:
			if err := t.makeColumn(&def, builder); err != nil {
				return err
			}
		case TableConstraint:
			if err := makeTblConstraint(t.trad, builder, def); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%w: '%T' is not a valid table definition type",
				errInvalidDefinition, def)
		}
	}

	statements := builder.build()
	for i := range statements {
		if err := t.querier.Exec(statements[i]); err != nil {
			return err
		}
	}

	return nil
}

func (t *tableMaker) makeColumn(col *Column, builder *tableBuilder) error {
	builder.addCol()
	colBuilder := builder.getLastCol()

	colBuilder.name = col.Name

	var err error
	if colBuilder.typ, err = makeType(col.Type, t.trad); err != nil {
		return err
	}

	for i := range col.Constraints {
		if err = makeConstraint(t.trad, builder, col, col.Constraints[i]); err != nil {
			return err
		}
	}

	return nil
}

type tableBuilder struct {
	tableName        string
	columns          []columnBuilder
	tableConstraints []string
	indexes          []string
}

func (t *tableBuilder) addCol() {
	t.columns = append(t.columns, columnBuilder{})
}

func (t *tableBuilder) getLastCol() *columnBuilder {
	if len(t.columns) == 0 {
		return nil
	}

	return &t.columns[len(t.columns)-1]
}

func (t *tableBuilder) build() []string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("CREATE TABLE %s (", t.tableName))

	var colStr []string
	for i := range t.columns {
		colStr = append(colStr, t.columns[i].build())
	}

	builder.WriteString(strings.Join(colStr, ", "))

	if len(t.tableConstraints) > 0 {
		builder.WriteString(", ")
		builder.WriteString(strings.Join(t.tableConstraints, ", "))
	}

	builder.WriteString(")")

	statements := []string{builder.String()}

	for i := range t.indexes {
		statements = append(statements, t.indexes[i])
	}

	return statements
}

func (t *tableBuilder) buildAddColumn() []string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("ALTER TABLE %s ADD COLUMN ", t.tableName))

	colStr := t.columns[0].build()
	builder.WriteString(colStr)

	statements := []string{builder.String()}

	for i := range t.indexes {
		statements = append(statements, t.indexes[i])
	}

	return statements
}

type columnBuilder struct {
	name, typ   string
	constraints []string
}

func (c *columnBuilder) addConstraint(constr string) {
	c.constraints = append(c.constraints, constr)
}

func (c *columnBuilder) build() string {
	conStr := strings.Join(c.constraints, " ")

	return fmt.Sprintf("%s %s %s", c.name, c.typ, conStr)
}
