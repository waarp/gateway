package migrations

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func ver0_13_0AddTransferAutoResumeUp(db Actions) error {
	if err := db.DropView("normalized_transfers"); err != nil {
		return fmt.Errorf("failed to drop the transfers view: %w", err)
	}

	if err := db.AlterTable("transfers",
		AddColumn{Name: "remaining_tries", Type: TinyInt{}, NotNull: true, Default: 0},
		AddColumn{Name: "next_retry_delay", Type: Integer{}, NotNull: true, Default: 0},
		AddColumn{Name: "retry_increment_factor", Type: Float{}, NotNull: true, Default: 1},
		AddColumn{Name: "next_retry", Type: DateTime{}},
	); err != nil {
		return fmt.Errorf("failed to add the auto-resume columns: %w", err)
	}

	transStop := utils.If(db.GetDialect() == PostgreSQL,
		"null::timestamp", "null")

	if err := db.CreateView(&View{
		Name: "normalized_transfers",
		As: `WITH transfers_as_history(id, owner, remote_transfer_id, is_server,
				is_send, rule, client, account, agent, protocol, src_filename, 
				dest_filename, local_path, remote_path, filesize, start, stop, 
				status, step, progress, task_number, error_code, error_details,
				is_transfer, remaining_tries, next_retry_delay, retry_increment_factor,
				next_retry) AS (
					SELECT t.id, t.owner, t.remote_transfer_id, 
						t.local_account_id IS NOT NULL, r.is_send, r.name, 
						(CASE WHEN t.client_id IS NULL THEN '' ELSE c.name END),
						(CASE WHEN t.local_account_id IS NULL THEN ra.login ELSE la.login END),
						(CASE WHEN t.local_account_id IS NULL THEN p.name ELSE s.name END),
						(CASE WHEN t.local_account_id IS NULL THEN p.protocol ELSE s.protocol END),
						t.src_filename, t.dest_filename, t.local_path, t.remote_path, t.filesize,
						t.start, ` + transStop + `, t.status, t.step, t.progress, t.task_number,
						t.error_code, t.error_details, true, t.remaining_tries, t.next_retry_delay,
						t.retry_increment_factor, t.next_retry
					FROM transfers AS t
					LEFT JOIN rules AS r ON t.rule_id = r.id
					LEFT JOIN clients AS c ON t.client_id = c.id
					LEFT JOIN local_accounts  AS la ON  t.local_account_id = la.id
					LEFT JOIN remote_accounts AS ra ON t.remote_account_id = ra.id
					LEFT JOIN local_agents    AS s ON la.local_agent_id = s.id 
					LEFT JOIN remote_agents   AS p ON ra.remote_agent_id = p.id
				)
			SELECT id, owner, remote_transfer_id, is_server, is_send, rule, client,
		        account, agent, protocol, src_filename, dest_filename, local_path,
				remote_path, filesize, start, stop, status, step, progress, 
				task_number, error_code, error_details, false AS is_transfer,
		        0 AS remaining_tries, 0 AS next_retry_delay, 1 AS retry_increment_factor,
		        null AS next_retry
			FROM transfer_history UNION
			SELECT * FROM transfers_as_history`,
	}); err != nil {
		return fmt.Errorf("failed to re-create the normalized transfer view: %w", err)
	}

	return nil
}

func ver0_13_0AddTransferAutoResumeDown(db Actions) error {
	if err := db.DropView("normalized_transfers"); err != nil {
		return fmt.Errorf("failed to drop the transfers view: %w", err)
	}

	if err := db.AlterTable("transfers",
		DropColumn{Name: "remaining_tries"},
		DropColumn{Name: "retry_delay"},
		DropColumn{Name: "retry_increment_factor"},
		DropColumn{Name: "next_retry"},
	); err != nil {
		return fmt.Errorf("failed to add the auto-resume columns: %w", err)
	}

	if err := ver0_9_0RestoreNormalizedTransfersViewUp(db); err != nil {
		return err
	}

	return nil
}
