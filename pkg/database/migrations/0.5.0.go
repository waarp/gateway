package migrations

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"
)

type ver0_5_0RemoveRulePathSlash struct{}

func (v ver0_5_0RemoveRulePathSlash) Up(db migration.Actions) error {
	switch dial := db.GetDialect(); dial {
	case migration.SQLite:
		return db.Exec(`UPDATE rules SET path=LTRIM(path, '/')`)
	case migration.PostgreSQL, migration.MySQL:
		return db.Exec(`UPDATE rules SET path=TRIM(LEADING '/' FROM path)`)
	default:
		return errUnknownEngine(dial)
	}
}

func (v ver0_5_0RemoveRulePathSlash) Down(db migration.Actions) error {
	switch dial := db.GetDialect(); dial {
	case migration.SQLite, migration.PostgreSQL:
		return db.Exec(`UPDATE rules SET path='/' || path`)
	case migration.MySQL:
		return db.Exec(`UPDATE rules SET path=CONCAT("/", path)`)
	default:
		return errUnknownEngine(dial)
	}
}

type ver0_5_0CheckRulePathParent struct{}

func (v ver0_5_0CheckRulePathParent) Up(db migration.Actions) error {
	var query string
	switch dial := db.GetDialect(); dial {
	case migration.SQLite, migration.PostgreSQL:
		query = `SELECT A.name, A.path, B.name, B.path FROM rules A, rules B WHERE B.path LIKE A.path || '/%%'`
	case migration.MySQL:
		query = `SELECT A.name, A.path, B.name, B.path FROM rules A, rules B WHERE B.path LIKE CONCAT(A.path ,'/%%')`
	default:
		return errUnknownEngine(dial)
	}

	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		var aName, aPath, bName, bPath string
		if err := rows.Scan(&aName, &aPath, &bName, &bPath); err != nil {
			return err
		}
		return fmt.Errorf("the path of the rule '%s' (%s) must be changed so "+
			"that it is no longer a parent of the path of rule '%s' (%s)",
			aName, aPath, bName, bPath)
	}
	return nil
}

func (v ver0_5_0CheckRulePathParent) Down(migration.Actions) error {
	return nil // nothing to do
}
