package database

var (
	// tables lists the schema of all database tables
	tables []Table

	// BcryptRounds defines the number of rounds taken by bcrypt to hash passwords
	// in the database
	BcryptRounds = 12
)

// AddTable adds the given model to the pool of database tables.
func AddTable(t Table) {
	tables = append(tables, t)
}

// initialiser is an interface which models can optionally implement in order to
// set default values after the table is created when the application is launched
// for the first time.
type initialiser interface {
	Init(*Session) Error
}

// initTables creates the database tables if they don't exist and fills them
// with the default entries.
func initTables(db *Standalone) error {
	return db.Transaction(func(ses *Session) Error {
		for _, tbl := range tables {
			if ok, err := ses.session.IsTableExist(tbl.TableName()); err != nil {
				db.logger.Criticalf("Failed to retrieve database table list: %s", err)
				return NewInternalError(err)
			} else if !ok {
				if err := ses.session.Table(tbl.TableName()).CreateTable(tbl); err != nil {
					db.logger.Criticalf("Failed to create the '%s' database table: %s",
						tbl.TableName(), err)
					return NewInternalError(err)
				}
				if err := ses.session.Table(tbl.TableName()).CreateUniques(tbl); err != nil {
					db.logger.Criticalf("Failed to create the '%s' table uniques: %s",
						tbl.TableName(), err)
					return NewInternalError(err)
				}
				if err := ses.session.Table(tbl.TableName()).CreateIndexes(tbl); err != nil {
					db.logger.Criticalf("Failed to create the '%s' table indexes: %s",
						tbl.TableName(), err)
					return NewInternalError(err)
				}

				if init, ok := tbl.(initialiser); ok {
					if err := init.Init(ses); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}
