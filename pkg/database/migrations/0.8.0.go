package migrations

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

type ver0_8_0DropNormalizedTransfersView struct{}

func (ver0_8_0DropNormalizedTransfersView) Up(db Actions) error {
	if err := db.DropView("normalized_transfers"); err != nil {
		return fmt.Errorf("failed to drop the normalized transfer view: %w", err)
	}

	return nil
}

func (ver0_8_0DropNormalizedTransfersView) Down(db Actions) error {
	return (&ver0_7_0AddNormalizedTransfersView{}).Up(db)
}

type ver0_8_0AddTransferFilename struct{}

func (ver0_8_0AddTransferFilename) Up(db Actions) error {
	if err := db.AlterTable("transfers",
		AddColumn{Name: "src_filename", Type: Text{}, NotNull: true, Default: ""},
		AddColumn{Name: "dest_filename", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "local_path", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "remote_path", Type: Text{}, NotNull: true, Default: ""},
	); err != nil {
		return fmt.Errorf("failed to add the transfers 'filename' columns: %w", err)
	}

	if err := db.Exec(`UPDATE transfers SET remote_path='' WHERE 
        local_account_id IS NOT NULL`); err != nil {
		return fmt.Errorf("failed to update the transfer remote path: %w", err)
	}

	if err := db.Exec(`UPDATE transfers SET src_filename=local_path, dest_filename=remote_path
		WHERE (SELECT is_send FROM rules WHERE id=transfers.rule_id) = true`); err != nil {
		return fmt.Errorf("failed to update the transfer entries: %w", err)
	}

	if err := db.Exec(`UPDATE transfers SET src_filename=remote_path, dest_filename=local_path 
		WHERE (SELECT is_send FROM rules WHERE id=transfers.rule_id) = false`); err != nil {
		return fmt.Errorf("failed to update the transfer entries: %w", err)
	}

	return nil
}

func (ver0_8_0AddTransferFilename) Down(db Actions) error {
	if err := db.Exec(`UPDATE transfers SET remote_path=
    	(CASE WHEN src_filename='' THEN dest_filename ELSE src_filename END) 
		WHERE local_account_id IS NOT NULL`); err != nil {
		return fmt.Errorf("failed to restore the transfers 'remote_path': %w", err)
	}

	if err := db.AlterTable("transfers", DropConstraint{Name: "transfers_filename_check"}); err != nil {
		return fmt.Errorf("failed to drop the filename constraint: %w", err)
	}

	if err := db.AlterTable("transfers",
		DropColumn{Name: "src_filename"},
		DropColumn{Name: "dest_filename"},
		AlterColumn{Name: "local_path", Type: Text{}, NotNull: true},
		AlterColumn{Name: "remote_path", Type: Text{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to drop the transfers 'filename' columns: %w", err)
	}

	return nil
}

type ver0_8_0AddHistoryFilename struct{}

func (ver0_8_0AddHistoryFilename) Up(db Actions) error {
	if err := db.AlterTable("transfer_history",
		AddColumn{Name: "src_filename", Type: Text{}, NotNull: true, Default: ""},
		AddColumn{Name: "dest_filename", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "local_path", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "remote_path", Type: Text{}, NotNull: true, Default: ""},
	); err != nil {
		return fmt.Errorf("failed to add the history 'filename' columns: %w", err)
	}

	if err := db.Exec(`UPDATE transfer_history SET remote_path='' 
		WHERE is_server=true`); err != nil {
		return fmt.Errorf("failed to update the history remote path: %w", err)
	}

	if err := db.Exec(`UPDATE transfer_history SET src_filename=local_path, 
		dest_filename=remote_path WHERE is_send=true`); err != nil {
		return fmt.Errorf("failed to update the history entries: %w", err)
	}

	if err := db.Exec(`UPDATE transfer_history SET src_filename=remote_path, 
		dest_filename=local_path WHERE is_send=false`); err != nil {
		return fmt.Errorf("failed to update the history entries: %w", err)
	}

	return nil
}

func (ver0_8_0AddHistoryFilename) Down(db Actions) error {
	if err := db.Exec(`UPDATE transfer_history SET remote_path=
    	(CASE WHEN src_filename='' THEN dest_filename ELSE src_filename END) 
		WHERE is_server=true`); err != nil {
		return fmt.Errorf("failed to restore the transfers 'remote_path': %w", err)
	}

	if err := db.AlterTable("transfer_history", DropConstraint{Name: "history_filename_check"}); err != nil {
		return fmt.Errorf("failed to drop the filename constraint: %w", err)
	}

	if err := db.AlterTable("transfer_history",
		DropColumn{Name: "src_filename"},
		DropColumn{Name: "dest_filename"},
		AlterColumn{Name: "local_path", Type: Text{}, NotNull: true},
		AlterColumn{Name: "remote_path", Type: Text{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to drop the history 'filename' columns: %w", err)
	}

	return nil
}

type ver0_8_0UpdateNormalizedTransfersView struct{}

func (ver0_8_0UpdateNormalizedTransfersView) Up(db Actions) error {
	transStop := utils.If(db.GetDialect() == PostgreSQL,
		"null::timestamp", "null")

	if err := db.CreateView(&View{
		Name: "normalized_transfers",
		As: `WITH transfers_as_history(id, owner, remote_transfer_id, is_server,
				is_send, rule, account, agent, protocol, src_filename, dest_filename,
				local_path, remote_path, filesize, start, stop, status, step, progress,
				task_number, error_code, error_details, is_transfer) AS (
					SELECT t.id, t.owner, t.remote_transfer_id, 
						t.local_account_id IS NOT NULL, r.is_send, r.name,
						(CASE WHEN t.local_account_id IS NULL THEN ra.login ELSE la.login END),
						(CASE WHEN t.local_account_id IS NULL THEN p.name ELSE s.name END),
						(CASE WHEN t.local_account_id IS NULL THEN p.protocol ELSE s.protocol END),
						t.src_filename, t.dest_filename, t.local_path, t.remote_path, t.filesize, 
						t.start, ` + transStop + `, t.status, t.step, t.progress, 
						t.task_number, t.error_code, t.error_details, true
					FROM transfers AS t
					LEFT JOIN rules AS r ON t.rule_id = r.id
					LEFT JOIN local_accounts  AS la ON  t.local_account_id = la.id
					LEFT JOIN remote_accounts AS ra ON t.remote_account_id = ra.id
					LEFT JOIN local_agents    AS s ON la.local_agent_id = s.id 
					LEFT JOIN remote_agents   AS p ON ra.remote_agent_id = p.id
				)
			SELECT id, owner, remote_transfer_id, is_server, is_send, rule, account,
				agent, protocol, src_filename, dest_filename, local_path, remote_path,
				filesize, start, stop, status, step, progress, task_number, error_code,
				error_details, false AS is_transfer
			FROM transfer_history UNION
			SELECT * FROM transfers_as_history`,
	}); err != nil {
		return fmt.Errorf("failed to create the normalized transfer view: %w", err)
	}

	return nil
}

func (ver0_8_0UpdateNormalizedTransfersView) Down(db Actions) error {
	if err := db.DropView("normalized_transfers"); err != nil {
		return fmt.Errorf("failed to drop the normalized transfer view: %w", err)
	}

	return nil
}
