package migrations

import (
	"fmt"
)

type ver0_5_2FillRemoteTransferID struct{}

func (ver0_5_2FillRemoteTransferID) Up(db Actions) error {
	if err := db.Exec(`UPDATE transfers SET remote_transfer_id=id
			WHERE remote_transfer_id=''`); err != nil {
		return fmt.Errorf("failed to fill the remote transfer id: %w", err)
	}

	return nil
}

func (ver0_5_2FillRemoteTransferID) Down(db Actions) error {
	if err := db.Exec(`UPDATE transfers SET remote_transfer_id='' 
			WHERE is_server=false`); err != nil {
		return fmt.Errorf("failed to re-empty ent remote transfer id: %w", err)
	}

	return nil
}
