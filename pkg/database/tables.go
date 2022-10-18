package database

import (
	"database/sql"
	"fmt"

	"code.waarp.fr/lib/log"
	"xorm.io/xorm"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

//nolint:gochecknoglobals // global var is used by design
var (
	// Tables lists the schema of all database tables.
	tables []Table

	// BcryptRounds defines the number of rounds taken by bcrypt to hash passwords
	// in the database.
	BcryptRounds = 12
)

// AddTable adds the given model to the pool of database tables.
func AddTable(t Table) {
	tables = append(tables, t)
}

// UpdateTable adds the given model to the pool of database tables.
func UpdateTable(t Table) {
	for i, e := range tables {
		if e.TableName() == t.TableName() {
			tables[i] = t

			return
		}
	}

	tables = append(tables, t)
}

// initialiser is an interface which models can optionally implement in order to
// set default values after the table is created when the application is launched
// for the first time.
type initialiser interface {
	Init(Access) Error
}

type Executor struct {
	Dialect string
	Logger  *log.Logger
	ses     *xorm.Session
}

func (e *Executor) Exec(query string, args ...interface{}) (sql.Result, error) {
	elems := append([]interface{}{query}, args...)

	if res, err := e.ses.Exec(elems...); err != nil {
		return nil, fmt.Errorf("command failed: %w", err)
	} else {
		return res, nil
	}
}

func (e *Executor) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	elems := append([]interface{}{query}, args...)

	if res, err := e.ses.QueryInterface(elems...); err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	} else {
		return res, nil
	}
}

// initTables creates the database tables if they don't exist and fills them
// with the default entries.
func initTables(db *Standalone, withInit bool) error {
	return db.Transaction(func(ses *Session) Error {
		for _, tbl := range tables {
			if ok, err := ses.session.IsTableExist(tbl.TableName()); err != nil {
				db.logger.Critical("Failed to retrieve database table list: %s", err)

				return NewInternalError(err)
			} else if !ok {
				if err := ses.session.Table(tbl.TableName()).CreateTable(tbl); err != nil {
					db.logger.Critical("Failed to create the '%s' database table: %s",
						tbl.TableName(), err)

					return NewInternalError(err)
				}

				if fker, ok := tbl.(ExtraConstraintsMaker); ok {
					exe := &Executor{conf.GlobalConfig.Database.Type, ses.logger, ses.session}
					if err := fker.MakeExtraConstraints(exe); err != nil {
						return err
					}
				}

				if err := ses.session.Table(tbl.TableName()).CreateUniques(tbl); err != nil {
					db.logger.Critical("Failed to create the '%s' table uniques: %s",
						tbl.TableName(), err)

					return NewInternalError(err)
				}

				if err := ses.session.Table(tbl.TableName()).CreateIndexes(tbl); err != nil {
					db.logger.Critical("Failed to create the '%s' table indexes: %s",
						tbl.TableName(), err)

					return NewInternalError(err)
				}
			}

			if init, ok := tbl.(initialiser); ok && withInit {
				if err := init.Init(ses); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
