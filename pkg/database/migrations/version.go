package migrations

import (
	"database/sql"
	"fmt"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"
)

func initVersion() Script {
	return Script{
		Up: func(db Dialect) error {
			if err := db.CreateTable("version", Col("current", TEXT)); err != nil {
				return err
			}
			return db.AddRow("version", Cells{"current": Cel(TEXT, "0.0.0")})
		},
		Down: func(db Dialect) error {
			return db.DropTable("version")
		},
	}
}

func bumpVersion(from, to string) Script {
	return Script{
		Up: func(db Dialect) error {
			_, err := db.Exec("UPDATE 'version' SET current='%s'", to)
			return err
		},
		Down: func(db Dialect) error {
			_, err := db.Exec("UPDATE 'version' SET current='%s'", from)
			return err
		},
	}
}

func checkVersionTableExist(db *sql.DB, dialect string) (bool, error) {
	switch dialect {
	case SQLite:
		row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='version'")
		var name string
		if err := row.Scan(&name); err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, err
		}
		return true, nil
	case PostgreSQL:
		row := db.QueryRow("SELECT to_regclass('version')")
		var regclass interface{}
		if err := row.Scan(&regclass); err != nil {
			return false, err
		}
		return regclass != nil, nil
	case MySQL:
		row := db.QueryRow("SHOW TABLES LIKE 'version')")
		var name string
		if err := row.Scan(&name); err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, err
		}
		return true, nil
	default:
		return false, fmt.Errorf("unknown SQL dialect %s", dialect)
	}
}
