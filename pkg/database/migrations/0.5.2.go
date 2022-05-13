package migrations

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

type ver0_5_2FillRemoteTransferID struct{}

func (v ver0_5_2FillRemoteTransferID) Up(db migration.Actions) error {
	if err := db.Exec(`UPDATE transfers SET remote_transfer_id=id
			WHERE remote_transfer_id=''`); err != nil {
		return fmt.Errorf("failed to fill the remote transfer id: %w", err)
	}

	return nil
}

func (v ver0_5_2FillRemoteTransferID) Down(db migration.Actions) error {
	if err := db.Exec(`UPDATE transfers SET remote_transfer_id='' 
			WHERE is_server=false`); err != nil {
		return fmt.Errorf("failed to re-empty ent remote transfer id: %w", err)
	}

	return nil
}
