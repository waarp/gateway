package migrations

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// In some instances, we need some SQL identifier to be quoted. This will quote
// the given identifier with the appropriate character based on the dialect of
// the given database.
func quote(db Actions, identifier string) string {
	return quoteDial(db.GetDialect(), identifier)
}

func quoteDial(dialect, identifier string) string {
	if dialect == MySQL {
		return fmt.Sprintf("`%s`", identifier)
	}

	return fmt.Sprintf(`%q`, identifier)
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

type jsonObj map[string]any

func (j *jsonObj) Value() (driver.Value, error) {
	b, err := json.Marshal(j)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(b), nil
}

func (j *jsonObj) Scan(src any) error {
	var jErr error

	switch val := src.(type) {
	case string:
		jErr = json.Unmarshal([]byte(val), j)
	case []byte:
		jErr = json.Unmarshal(val, j)
	default:
		//nolint:err113 //too specific
		jErr = fmt.Errorf("unknown JSON type %T", src)
	}

	return jErr
}
