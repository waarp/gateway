package migrations

import (
	"code.waarp.fr/lib/migration"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type Actions = migration.Actions

// Shortcuts for table schema.
type (
	Table  = migration.Table
	Column = migration.Column
	Index  = migration.Index
	View   = migration.View
)

// Shortcuts for SQL types.
type (
	Boolean        = migration.Boolean
	TinyInt        = migration.TinyInt
	SmallInt       = migration.SmallInt
	Integer        = migration.Integer
	BigInt         = migration.BigInt
	Float          = migration.Float
	Double         = migration.Double
	Varchar        = migration.Varchar
	Text           = migration.Text
	Date           = migration.Date
	DateTime       = migration.DateTime
	DateTimeOffset = migration.DateTimeOffset
	Blob           = migration.Blob

	CurrentTimestamp = migration.CurrentTimestamp
	AutoIncr         = migration.AutoIncr
)

// Shortcuts for table constraints.
type (
	PrimaryKey = migration.PrimaryKey
	ForeignKey = migration.ForeignKey
	Unique     = migration.Unique
	Check      = migration.Check
)

// Shortcuts for ALTER TABLE.
type (
	AddColumn      = migration.AddColumn
	DropColumn     = migration.DropColumn
	RenameColumn   = migration.RenameColumn
	AlterColumn    = migration.AlterColumn
	AddPrimaryKey  = migration.AddPrimaryKey
	AddForeignKey  = migration.AddForeignKey
	AddUnique      = migration.AddUnique
	AddCheck       = migration.AddCheck
	DropPrimaryKey = migration.DropPrimaryKey
	DropConstraint = migration.DropConstraint
)

// Shortcuts for REF ACTIONS.
const (
	NoAction   = migration.NoAction
	Restrict   = migration.Restrict
	Cascade    = migration.Cascade
	SetNull    = migration.SetNull
	SetDefault = migration.SetDefault
)

//nolint:gochecknoglobals //a global var is necessary here to make a function alias
var checkOnlyOneNotNull = utils.CheckOnlyOneNotNull
