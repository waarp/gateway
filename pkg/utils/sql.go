package utils

import (
	"database/sql"
)

// NewNullInt64 returns a new [sql.NullInt64] created from the given int64.
//
// Deprecated: use model.Identifier.NullableID() for variables, or
// sql.NullInt64{Int64: i, Valid: true} for constants.
func NewNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: i,
		Valid: true,
	}
}
