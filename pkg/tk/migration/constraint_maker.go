package migration

import (
	"fmt"
	"strings"
)

type constraintMaker interface {
	makePrimaryKey(builder *tableBuilder, colType sqlType) error
	makeForeignKey(builder *tableBuilder, table, col string) error
	makeNotNull(builder *tableBuilder) error
	makeDefault(builder *tableBuilder, formatter valueFormatter, value interface{}, colType sqlType) error
	makeUnique(builder *tableBuilder) error
	makeAutoIncrement(builder *tableBuilder, colType sqlType) error
	makePrimaryKeys(builder *tableBuilder, uqe *tblPk) error
	makeUniques(builder *tableBuilder, pk *tblUnique) error
}

type standardConstraintMaker struct{}

//nolint:unparam // some implementations might return an error
func (*standardConstraintMaker) makePrimaryKey(builder *tableBuilder, _ sqlType) error {
	builder.getLastCol().addConstraint("PRIMARY KEY")

	return nil
}

//nolint:unparam // some implementations might return an error
func (*standardConstraintMaker) makeForeignKey(builder *tableBuilder, table, col string) error {
	builder.getLastCol().addConstraint(fmt.Sprintf("REFERENCES %s(%s)", table, col))

	return nil
}

//nolint:unparam // some implementations might return an error
func (*standardConstraintMaker) makeNotNull(builder *tableBuilder) error {
	builder.getLastCol().addConstraint("NOT NULL")

	return nil
}

func (*standardConstraintMaker) makeDefault(builder *tableBuilder, formatter valueFormatter,
	value interface{}, colType sqlType) error {
	sqlVal, err := formatValue(value, colType, formatter)
	if err != nil {
		return err
	}

	builder.getLastCol().addConstraint(fmt.Sprintf("DEFAULT %s", sqlVal))

	return nil
}

//nolint:unparam // some implementations might return an error
func (*standardConstraintMaker) makeUnique(builder *tableBuilder) error {
	col := builder.getLastCol().name
	indName := fmt.Sprintf("UQE_%s_%s", builder.tableName, col)

	builder.indexes = append(builder.indexes, fmt.Sprintf(
		"CREATE UNIQUE INDEX %s ON %s (%s)", indName, builder.tableName, col))

	return nil
}

func (*standardConstraintMaker) makeAutoIncrement(builder *tableBuilder, colType sqlType) error {
	if !isIntegerType(colType) {
		return fmt.Errorf("auto-increments can only be used on "+
			"integer types (%s is not an integer type): %w",
			colType.code.String(), errBadConstraint)
	}

	builder.getLastCol().addConstraint("AUTO_INCREMENT")

	return nil
}

//nolint:unparam // some implementations might return an error
func (*standardConstraintMaker) makePrimaryKeys(builder *tableBuilder, uqe *tblPk) error {
	builder.tableConstraints = append(builder.tableConstraints, fmt.Sprintf(
		"PRIMARY KEY (%s)", strings.Join(uqe.cols, ", ")))

	return nil
}

//nolint:unparam // some implementations might return an error
func (*standardConstraintMaker) makeUniques(builder *tableBuilder, pk *tblUnique) error {
	cols := strings.Join(pk.cols, ", ")
	indCols := strings.Join(pk.cols, "_")
	indName := fmt.Sprintf("UQE_%s_%s", builder.tableName, indCols)

	builder.indexes = append(builder.indexes, fmt.Sprintf(
		"CREATE UNIQUE INDEX %s ON %s (%s)", indName, builder.tableName, cols))

	return nil
}

//nolint:wrapcheck // wrapping errors would hurt readability without adding any value to the errors themselves
func makeConstraint(trad translator, builder *tableBuilder, col *Column, c Constraint) error {
	switch constr := c.(type) {
	case pk:
		return trad.makePrimaryKey(builder, col.Type)
	case fk:
		return trad.makeForeignKey(builder, constr.table, constr.col)
	case notNull:
		return trad.makeNotNull(builder)
	case autoIncr:
		return trad.makeAutoIncrement(builder, col.Type)
	case unique:
		return trad.makeUnique(builder)
	case defaul:
		return trad.makeDefault(builder, trad, constr.val, col.Type)
	default:
		return fmt.Errorf("%w: unknown constraint type '%T'", errBadConstraint, c)
	}
}

//nolint:wrapcheck // wrapping errors would hurt readability without adding any value to the errors themselves
func makeTblConstraint(maker constraintMaker, builder *tableBuilder, c TableConstraint) error {
	switch constr := c.(type) {
	case tblPk:
		return maker.makePrimaryKeys(builder, &constr)
	case tblUnique:
		return maker.makeUniques(builder, &constr)
	default:
		return fmt.Errorf("%w: unknown table constraint type '%T'", errBadConstraint, c)
	}
}
