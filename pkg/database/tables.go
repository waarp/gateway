package database

var (
	// Tables lists the schema of all database tables
	Tables []tableName

	// BcryptRounds defines the number of rounds taken by bcrypt to hash passwords
	// in the database
	BcryptRounds = 12
)

type initer interface {
	Init(Accessor) error
}

// initTables creates the database tables if they don't exist and fills them
// with the default entries.
func initTables(db *DB) error {
	trans, err := db.BeginTransaction()
	if err != nil {
		return err
	}
	defer trans.session.Close()

	for _, table := range Tables {
		if ok, err := trans.session.IsTableExist(table); err != nil {
			return NewInternalError(err, "cannot retrieve database table list")
		} else if !ok {
			if err := trans.session.CreateTable(table); err != nil {
				return NewInternalError(err, "failed to create '%s' database table",
					table.TableName())
			}

			if init, ok := table.(initer); ok {
				if err := init.Init(trans); err != nil {
					return err
				}
			}
		}
	}

	return trans.Commit()
}
