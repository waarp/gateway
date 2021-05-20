package database

var (
	// Tables lists the schema of all database tables
	Tables []Table

	// BcryptRounds defines the number of rounds taken by bcrypt to hash passwords
	// in the database
	BcryptRounds = 12
)

type initer interface {
	Init(*Session) Error
}

// initTables creates the database tables if they don't exist and fills them
// with the default entries.
func initTables(db *Standalone) error {
	return db.Transaction(func(ses *Session) Error {
		for _, tbl := range Tables {
			if ok, err := ses.session.IsTableExist(tbl.TableName()); err != nil {
				db.logger.Criticalf("Failed to retrieve database table list: %s", err)
				return NewInternalError(err)
			} else if !ok {
				if err := ses.session.Table(tbl.TableName()).CreateTable(tbl); err != nil {
					db.logger.Criticalf("Failed to create the '%s' database table: %s",
						tbl.TableName(), err)
					return NewInternalError(err)
				}

				if init, ok := tbl.(initer); ok {
					if err := init.Init(ses); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}
