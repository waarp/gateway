package migrations

import (
	"fmt"

	migration2 "code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

type ver0_4_2RemoveHistoryRemoteIDUnique struct{}

func (ver0_4_2RemoveHistoryRemoteIDUnique) Up(db migration2.Actions) error {
	var err error

	switch db.GetDialect() {
	case migration2.PostgreSQL, migration2.SQLite:
		err = db.Exec(`DROP INDEX "UQE_transfer_history_histRemID"`)
	case migration2.MySQL:
		err = db.Exec("DROP INDEX UQE_transfer_history_histRemID ON transfer_history")
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	if err != nil {
		return fmt.Errorf("cannot upgrade database: %w", err)
	}

	return nil
}

func (ver0_4_2RemoveHistoryRemoteIDUnique) Down(db migration2.Actions) error {
	var err error

	switch db.GetDialect() {
	case migration2.PostgreSQL:
		err = db.Exec(`CREATE UNIQUE INDEX "UQE_transfer_history_histRemID"
				ON transfer_history (remote_transfer_id, account, agent)`)
	case migration2.SQLite, migration2.MySQL:
		err = db.Exec(`CREATE UNIQUE INDEX UQE_transfer_history_histRemID
				ON transfer_history (remote_transfer_id, account, agent)`)
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	if err != nil {
		return fmt.Errorf("cannot upgrade database: %w", err)
	}

	return nil
}
