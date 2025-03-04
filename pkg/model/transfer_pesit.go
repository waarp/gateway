package model

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync/atomic"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //a global var is unfortunately needed here
var pesitCounter atomic.Uint32

func initPesitCounter(db database.ReadAccess) error {
	row := db.QueryRow(`SELECT MAX(remote_transfer_id) FROM normalized_transfers
		WHERE is_send=true AND protocol IN ('pesit', 'pesit-tls')`)

	var val sql.NullString
	if err := row.Scan(&val); err != nil {
		return fmt.Errorf("failed to scan pesit counter: %w", err)
	}

	if val.Valid {
		lastID, convErr := strconv.ParseUint(val.String, 10, 32)
		if convErr != nil {
			return fmt.Errorf("failed to parse pesit transfer ID: %w", convErr)
		}

		pesitCounter.Store(uint32(lastID))
	}

	return nil
}

func mkPesitID(db database.Access, t *Transfer) error {
	const maxPesitID = 1<<24 - 1

	var check NormalizedTransferView
	if err := db.Get(&check, "id=?", t.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve transfer: %w", err)
	}

	if check.Protocol != "pesit" && check.Protocol != "pesit-tls" {
		return nil
	}

	if !check.IsSend {
		return nil
	}

	newID := pesitCounter.Add(1)
	if newID > maxPesitID {
		newID = 1
		pesitCounter.Store(newID)
	}

	t.RemoteTransferID = utils.FormatUint(newID)
	if err := db.Update(t).Cols("remote_transfer_id").Run(); err != nil {
		return fmt.Errorf("failed to update pesit ID: %w", err)
	}

	return nil
}
