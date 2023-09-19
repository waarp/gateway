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
