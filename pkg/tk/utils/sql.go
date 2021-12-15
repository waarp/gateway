package utils

import (
	"database/sql"
	"fmt"
	"strings"
)

// NewNullInt64 returns a new sql.NullInt64 created from the given int64.
func NewNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{
		Int64: i,
		Valid: true,
	}
}

// CheckOnlyOneNotNull generates an SQL CHECK constraint for the given dialect
// stating that 1, and only 1 of the given column must be defined (i.e. that
// all but one of the column must be null).
//
// Valid dialects are sqlite, mysql & postgresql. Other dialects will return an
// empty string.
func CheckOnlyOneNotNull(dialect string, cols ...string) string {
	isNulls := make([]string, len(cols))

	switch dialect {
	case "postgresql":
		for i := range cols {
			isNulls[i] = fmt.Sprintf("(CASE WHEN %s IS NOT NULL THEN 1 ELSE 0 END)", cols[i])
		}
	case "sqlite", "mysql":
		for i := range cols {
			isNulls[i] = fmt.Sprintf("(%s IS NOT NULL)", cols[i])
		}
	default:
		return ""
	}

	sum := strings.Join(isNulls, " + ")

	return fmt.Sprintf("%s = 1", sum)
}
