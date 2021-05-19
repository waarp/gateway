package migrations

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/migration"
)

func initVersion() migration.Script {
	return migration.Script{
		Up: func(db migration.Actions) error {
			if err := db.CreateTable("version", migration.Col("current", migration.TEXT)); err != nil {
				return err
			}
			return db.AddRow("version", migration.Cells{"current": migration.Cel(migration.TEXT, "0.0.0")})
		},
		Down: func(db migration.Actions) error {
			return db.DropTable("version")
		},
	}
}

func bumpVersion(from, to string) migration.Script {
	return migration.Script{
		Up: func(db migration.Actions) error {
			_, err := db.Exec("UPDATE 'version' SET current='%s'", to)
			return err
		},
		Down: func(db migration.Actions) error {
			_, err := db.Exec("UPDATE 'version' SET current='%s'", from)
			return err
		},
	}
}

func checkVersionTableExist(db *sql.DB, dialect string) (bool, error) {
	switch dialect {
	case migration.SQLite:
		row := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='version'")
		var name string
		if err := row.Scan(&name); err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, err
		}
		return true, nil
	case migration.PostgreSQL:
		row := db.QueryRow("SELECT to_regclass('version')")
		var regclass interface{}
		if err := row.Scan(&regclass); err != nil {
			return false, err
		}
		return regclass != nil, nil
	case migration.MySQL:
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
