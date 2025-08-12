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
		DropColumn{Name: "next_retry_delay"},
		DropColumn{Name: "retry_increment_factor"},
		DropColumn{Name: "next_retry"},
	); err != nil {
		return fmt.Errorf("failed to add the auto-resume columns: %w", err)
	}

	return ver0_9_0RestoreNormalizedTransfersViewUp(db)
}

func ver0_13_0AddClientAutoResumeUp(db Actions) error {
	if err := db.AlterTable("clients",
		AddColumn{Name: "nb_of_attempts", Type: TinyInt{}, NotNull: true, Default: 0},
		AddColumn{Name: "first_retry_delay", Type: Integer{}, NotNull: true, Default: 0},
		AddColumn{Name: "retry_increment_factor", Type: Float{}, NotNull: true, Default: 1},
	); err != nil {
		return fmt.Errorf("failed to add the client auto-resume columns: %w", err)
	}

	return nil
}

func ver0_13_0AddClientAutoResumeDown(db Actions) error {
	if err := db.AlterTable("clients",
		DropColumn{Name: "nb_of_attempts"},
		DropColumn{Name: "first_retry_delay"},
		DropColumn{Name: "retry_increment_factor"},
	); err != nil {
		return fmt.Errorf("failed to drop the client auto-resume columns: %w", err)
	}

	return nil
}

func ver0_13_0AddEmailTemplatesUp(db Actions) error {
	if err := db.CreateTable("email_templates", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "name", Type: Varchar(50), NotNull: true},
			{Name: "mime_type", Type: Varchar(50), NotNull: true, Default: "text/plain"},
			{Name: "subject", Type: Varchar(255), NotNull: true},
			{Name: "body", Type: Text{}, NotNull: true},
			{Name: "attachments", Type: Text{}, NotNull: true, Default: ""},
		},
		PrimaryKey: &PrimaryKey{Name: "email_template_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_email_template", Cols: []string{"name"}},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create table "email_templates" table: %w`, err)
	}

	return nil
}

func ver0_13_0AddEmailTemplatesDown(db Actions) error {
	if err := db.DropTable("email_templates"); err != nil {
		return fmt.Errorf(`failed to drop "email_templates" table: %w`, err)
	}

	return nil
}

func ver0_13_0AddSMTPCredentialsUp(db Actions) error {
	if err := db.CreateTable("smtp_credentials", &Table{
		Columns: []Column{
			{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
			{Name: "owner", Type: Varchar(50), NotNull: true},
			{Name: "email_address", Type: Varchar(255), NotNull: true},
			{Name: "server_address", Type: Varchar(255), NotNull: true},
			{Name: "login", Type: Varchar(255), NotNull: true},
			{Name: "password", Type: Text{}, NotNull: true},
		},
		PrimaryKey: &PrimaryKey{Name: "smtp_credentials_pkey", Cols: []string{"id"}},
		Uniques: []Unique{
			{Name: "unique_smtp_credential", Cols: []string{"owner", "email_address"}},
		},
	}); err != nil {
		return fmt.Errorf(`failed to create table "smtp_credentials" table: %w`, err)
	}

	return nil
}

func ver0_13_0AddSMTPCredentialsDown(db Actions) error {
	if err := db.DropTable("smtp_credentials"); err != nil {
		return fmt.Errorf(`failed to drop "smtp_credentials" table: %w`, err)
	}

	return nil
}
