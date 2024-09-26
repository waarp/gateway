package migrations

import (
	"fmt"
)

// In some instances, we need some SQL identifier to be quoted. This will quote
// the given identifier with the appropriate character based on the dialect of
// the given database.
func quote(db Actions, identifier string) string {
	if db.GetDialect() == MySQL {
		return fmt.Sprintf("`%s`", identifier)
	}

	return fmt.Sprintf(`"%s"`, identifier)
}

func ltrim(db Actions, pref, from string) (sql string) {
	sql = fmt.Sprintf("LTRIM(%s, %s)", from, pref)
	if db.GetDialect() == MySQL {
		sql = fmt.Sprintf("TRIM(LEADING %s FROM %s)", pref, from)
	}

	return sql
}

func ifNull(db Actions, expr, defValue string) (sql string) {
	sql = fmt.Sprintf("IFNULL((%s), (%s))", expr, defValue)
	if db.GetDialect() == PostgreSQL {
		sql = fmt.Sprintf("COALESCE((%s), (%s))", expr, defValue)
	}

	return sql
}

func concat(db Actions, str1, str2 string) (sql string) {
	sql = fmt.Sprintf("%s || %s", str1, str2)
	if db.GetDialect() == MySQL {
		sql = fmt.Sprintf("CONCAT((%s), (%s))", str1, str2)
	}

	return sql
}
