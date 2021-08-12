package migrations

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/migration"
)

type ver0_4_2RemoveHistoryRemoteIDUnique struct{}

func (ver0_4_2RemoveHistoryRemoteIDUnique) Up(db migration.Actions) error {
	var err error
	switch db.GetDialect() {
	case migration.PostgreSQL, migration.SQLite:
		err = db.Exec(`DROP INDEX "UQE_transfer_history_histRemID"`)
	case migration.MySQL:
		err = db.Exec("DROP INDEX UQE_transfer_history_histRemID ON transfer_history")
	default:
		return fmt.Errorf("unknown dialect engine %T", db)
	}
	return err
}
func (ver0_4_2RemoveHistoryRemoteIDUnique) Down(db migration.Actions) error {
	switch db.GetDialect() {
	case migration.PostgreSQL:
		return db.Exec(`CREATE UNIQUE INDEX "UQE_transfer_history_histRemID"
				ON transfer_history (remote_transfer_id, account, agent)`)
	case migration.SQLite, migration.MySQL:
		return db.Exec(`CREATE UNIQUE INDEX UQE_transfer_history_histRemID
				ON transfer_history (remote_transfer_id, account, agent)`)
	default:
		return fmt.Errorf("unknown dialect engine %T", db)
	}
}
