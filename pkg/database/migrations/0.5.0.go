package migrations

import (
	"fmt"
	"path"
	"runtime"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
)

type ver0_5_0LocalAgentChangePaths struct{}

func (ver0_5_0LocalAgentChangePaths) Up(db migration.Actions) (err error) {
	if runtime.GOOS != windowsRuntime {
		return nil // nothing to do
	}

	switch db.GetDialect() {
	case migration.SQLite:
		err = db.Exec(`UPDATE local_agents SET 
			root = REPLACE(LTRIM(root, '/'), '/', '\'),
			in_dir = REPLACE(LTRIM(in_dir, '/'), '/', '\'),
			out_dir = REPLACE(LTRIM(out_dir, '/'), '/', '\'),
			work_dir = REPLACE(LTRIM(work_dir, '/'), '/', '\')`)
	case migration.PostgreSQL:
		err = db.Exec(`UPDATE local_agents SET
			root = replace(trim(leading '/' from root), '/', '\'),
			in_dir = replace(trim(leading '/' from in_dir), '/', '\'),
			out_dir = replace(trim(leading '/' from out_dir), '/', '\'),
			work_dir = replace(trim(leading '/' from work_dir), '/', '\')`)
	case migration.MySQL:
		err = db.Exec(`UPDATE local_agents SET
			root = REPLACE(TRIM(LEADING '/' FROM root), '/', '\\'),
			in_dir = REPLACE(TRIM(LEADING '/' FROM in_dir), '/', '\\'),
			out_dir = REPLACE(TRIM(LEADING '/' FROM out_dir), '/', '\\'),
			work_dir = REPLACE(TRIM(LEADING '/' FROM work_dir), '/', '\\')`)
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	if err != nil {
		return fmt.Errorf("failed to change server paths: %w", err)
	}

	return nil
}

func (ver0_5_0LocalAgentChangePaths) Down(db migration.Actions) (err error) {
	if runtime.GOOS != windowsRuntime {
		return nil // nothing to do
	}

	switch db.GetDialect() {
	case migration.SQLite, migration.PostgreSQL:
		if err = db.Exec(`UPDATE local_agents SET root = ('/' || root) WHERE root LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the server root directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET in_dir = ('/' || in_dir) 
			WHERE in_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the server in directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET out_dir = ('/' || out_dir)
			WHERE out_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the server out directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET work_dir = ('/' || work_dir)
			WHERE work_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the server tmp directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET root = REPLACE(root, '\', '/'), 
			in_dir = REPLACE(in_dir, '\', '/'),	out_dir = REPLACE(out_dir, '\', '/'), 
			work_dir = REPLACE(work_dir, '\', '/')`); err != nil {
			return fmt.Errorf("failed to change server paths: %w", err)
		}
	case migration.MySQL:
		if err = db.Exec(`UPDATE local_agents SET root = CONCAT('/', root) WHERE root LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the server root directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET in_dir = CONCAT('/', in_dir) 
			WHERE in_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the server in directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET out_dir = CONCAT('/', out_dir) 
			WHERE out_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the server out directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET work_dir = CONCAT('/', work_dir) 
			WHERE work_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the server tmp directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET root = REPLACE(root, '\\', '/'),
			in_dir = REPLACE(in_dir, '\\', '/'), out_dir = REPLACE(out_dir, '\\', '/'),
			work_dir = REPLACE(work_dir, '\\', '/')`); err != nil {
			return fmt.Errorf("failed to change server paths: %w", err)
		}
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	return nil
}

type ver0_5_0LocalAgentDisallowReservedNames struct{}

func (ver0_5_0LocalAgentDisallowReservedNames) Up(db migration.Actions) error {
	rows, err := db.Query(`SELECT name FROM local_agents`)
	if err != nil {
		return fmt.Errorf("failed to retrieve local servers list: %w", err)
	}

	defer rows.Close() //nolint:errcheck //this error is irrelevant

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("failed to retrieve server name: %w", err)
		}

		if name == "Database" || name == "Admin" || name == "Controller" {
			//nolint:goerr113 //this is a base error
			return fmt.Errorf("'%s' is a reserved service name, this server should be renamed",
				name)
		}
	}

	return nil
}

func (ver0_5_0LocalAgentDisallowReservedNames) Down(migration.Actions) error {
	return nil // nothing to do
}

type ver0_5_0RuleNewPathCols struct{}

func (ver0_5_0RuleNewPathCols) Up(db migration.Actions) error {
	if err := db.RenameColumn("rules", "in_path", "local_dir"); err != nil {
		return fmt.Errorf("failed to rename the rule 'in_path' column: %w", err)
	}

	if err := db.RenameColumn("rules", "out_path", "remote_dir"); err != nil {
		return fmt.Errorf("failed to rename the rule 'out_path' column: %w", err)
	}

	if err := db.RenameColumn("rules", "work_path", "local_tmp_dir"); err != nil {
		return fmt.Errorf("failed to rename the rule 'work_dir' column: %w", err)
	}

	if err := db.SwapColumns("rules", "local_dir", "remote_dir", "send=true"); err != nil {
		return fmt.Errorf("failed to swap the rule path columns values: %w", err)
	}

	return nil
}

func (ver0_5_0RuleNewPathCols) Down(db migration.Actions) error {
	if err := db.SwapColumns("rules", "local_dir", "remote_dir", "send=true"); err != nil {
		return fmt.Errorf("failed to re-swap the rule path columns values: %w", err)
	}

	if err := db.RenameColumn("rules", "local_dir", "in_path"); err != nil {
		return fmt.Errorf("failed to revert renaming the rule 'in_path' column: %w", err)
	}

	if err := db.RenameColumn("rules", "remote_dir", "out_path"); err != nil {
		return fmt.Errorf("failed to revert renaming the rule 'out_path' column: %w", err)
	}

	if err := db.RenameColumn("rules", "local_tmp_dir", "work_path"); err != nil {
		return fmt.Errorf("failed to revert renaming the rule 'work_dir' column: %w", err)
	}

	return nil
}

type ver0_5_0RulePathChanges struct{}

func (ver0_5_0RulePathChanges) Up(db migration.Actions) (err error) {
	if runtime.GOOS != windowsRuntime {
		return nil // nothing to do
	}

	switch db.GetDialect() {
	case migration.SQLite:
		err = db.Exec(`UPDATE rules SET 
			local_dir = REPLACE(LTRIM(local_dir, '/'), '/', '\'),
			local_tmp_dir = REPLACE(LTRIM(local_tmp_dir, '/'), '/', '\')`)
	case migration.PostgreSQL:
		err = db.Exec(`UPDATE rules SET
			local_dir = replace(trim(leading '/' from local_dir), '/', '\'),
			local_tmp_dir = replace(trim(leading '/' from local_tmp_dir), '/', '\')`)
	case migration.MySQL:
		err = db.Exec(`UPDATE rules SET
			local_dir = REPLACE(TRIM(LEADING '/' FROM local_dir), '/', '\\'),
			local_tmp_dir = REPLACE(TRIM(LEADING '/' FROM local_tmp_dir), '/', '\\')`)
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	if err != nil {
		return fmt.Errorf("failed to change rule paths: %w", err)
	}

	return nil
}

func (ver0_5_0RulePathChanges) Down(db migration.Actions) (err error) {
	if runtime.GOOS != windowsRuntime {
		return nil // nothing to do
	}

	switch db.GetDialect() {
	case migration.MySQL:
		if err = db.Exec(`UPDATE rules SET local_dir = CONCAT('/', local_dir) 
			WHERE local_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to restore rule local directory: %w", err)
		}

		if err = db.Exec(`UPDATE rules SET local_tmp_dir = CONCAT('/', local_tmp_dir) 
			WHERE local_tmp_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to restore rule tmp directory: %w", err)
		}
	case migration.SQLite, migration.PostgreSQL:
		if err = db.Exec(`UPDATE rules SET local_dir = ('/' || local_dir) 
			WHERE local_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to restore rule local directory: %w", err)
		}

		if err = db.Exec(`UPDATE rules SET local_tmp_dir = ('/' || local_tmp_dir) 
			WHERE local_tmp_dir LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to restore rule tmp directory: %w", err)
		}
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	switch db.GetDialect() {
	case migration.SQLite:
		err = db.Exec(`UPDATE rules SET 
			local_dir = REPLACE(local_dir, '\', '/'),
			local_tmp_dir = REPLACE(local_tmp_dir, '\', '/')`)
	case migration.PostgreSQL:
		err = db.Exec(`UPDATE rules SET
			local_dir = replace(local_dir, '\', '/'),
			local_tmp_dir = replace(local_tmp_dir, '\', '/')`)
	case migration.MySQL:
		err = db.Exec(`UPDATE rules SET
			local_dir = REPLACE(local_dir, '\\', '/'),
			local_tmp_dir = REPLACE(local_tmp_dir, '\\', '/')`)
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	if err != nil {
		return fmt.Errorf("failed to revert rule paths: %w", err)
	}

	return nil
}

type ver0_5_0AddFilesize struct{}

func (ver0_5_0AddFilesize) Up(db migration.Actions) error {
	if err := db.AddColumn("transfers", "filesize", migration.BigInt,
		migration.NotNull, migration.Default(int64(-1))); err != nil {
		return fmt.Errorf("failed to add transfer 'filesize' column: %w", err)
	}

	if err := db.AddColumn("transfer_history", "filesize", migration.BigInt,
		migration.NotNull, migration.Default(int64(-1))); err != nil {
		return fmt.Errorf("failed to add history 'filesize' column: %w", err)
	}

	return nil
}

func (ver0_5_0AddFilesize) Down(db migration.Actions) error {
	if err := db.DropColumn("transfers", "filesize"); err != nil {
		return fmt.Errorf("failed to drop transfer 'filesize' column: %w", err)
	}

	if err := db.DropColumn("transfer_history", "filesize"); err != nil {
		return fmt.Errorf("failed to drop history 'filesize' column: %w", err)
	}

	return nil
}

type ver0_5_0TransferChangePaths struct{}

func (ver0_5_0TransferChangePaths) Up(db migration.Actions) error {
	if err := db.RenameColumn("transfers", "true_filepath", "local_path"); err != nil {
		return fmt.Errorf("failed to rename the 'true_filepath' transfer column: %w", err)
	}

	if err := db.RenameColumn("transfers", "source_file", "remote_path"); err != nil {
		return fmt.Errorf("failed to rename the 'source_file' transfer column: %w", err)
	}

	if err := db.Exec(`UPDATE transfers SET remote_path = dest_file WHERE 
		rule_id IN (SELECT id FROM rules WHERE send=true)`); err != nil {
		return fmt.Errorf("failed to fill the new 'remote_path' transfer column: %w", err)
	}

	var err error

	switch db.GetDialect() {
	case migration.SQLite, migration.PostgreSQL:
		err = db.Exec(`UPDATE transfers SET remote_path = (SELECT remote_dir FROM 
			rules WHERE id = transfers.rule_id) || '/' || transfers.remote_path`)
	case migration.MySQL:
		err = db.Exec(`UPDATE transfers SET remote_path = CONCAT((SELECT remote_dir 
			FROM rules WHERE id = transfers.rule_id), '/', transfers.remote_path)`)
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	if err != nil {
		return fmt.Errorf("failed to format the `remote_path` transfer values: %w", err)
	}

	if err := db.DropColumn("transfers", "dest_file"); err != nil {
		return fmt.Errorf("failed to drop the 'dest_file' transfer column: %w", err)
	}

	return nil
}

func (ver0_5_0TransferChangePaths) makeTransList(db migration.Actions, off uint64) ([]struct {
	id             uint64
	full, src, dst string
	send           bool
}, error) {
	const buffSize = 100

	rows, err := db.Query(fmt.Sprintf(`SELECT transfers.id, transfers.true_filepath, 
		transfers.source_file, transfers.dest_file, rules.send FROM transfers 
		INNER JOIN rules ON	rules.id = transfers.rule_id LIMIT %d OFFSET %d`,
		buffSize, off))
	if err != nil || rows.Err() != nil {
		return nil, fmt.Errorf("failed to retrieve transfer entries: %w", err)
	}

	defer rows.Close() //nolint:errcheck //this error is unimportant

	list := make([]struct {
		id             uint64
		full, src, dst string
		send           bool
	}, 0, buffSize)

	for rows.Next() {
		row := struct {
			id             uint64
			full, src, dst string
			send           bool
		}{}

		if err := rows.Scan(&row.id, &row.full, &row.src, &row.dst, &row.send); err != nil {
			return nil, fmt.Errorf("failed to parse transfer row: %w", err)
		}

		list = append(list, row)
	}

	return list, nil
}

func (v ver0_5_0TransferChangePaths) Down(db migration.Actions) error {
	if err := db.RenameColumn("transfers", "local_path", "true_filepath"); err != nil {
		return fmt.Errorf("failed to revert renaming the 'true_filepath' transfer column: %w", err)
	}

	if err := db.RenameColumn("transfers", "remote_path", "source_file"); err != nil {
		return fmt.Errorf("failed to revert renaming the 'source_file' transfer column: %w", err)
	}

	if err := db.AddColumn("transfers", "dest_file", migration.Varchar(255), //nolint:gomnd //unnecessary here
		migration.NotNull, migration.Default("")); err != nil {
		return fmt.Errorf("failed to restore the 'dest_file' transfer column: %w", err)
	}

	var off uint64

	for {
		trans, err := v.makeTransList(db, off)
		if err != nil {
			return err
		}

		if len(trans) == 0 {
			return nil
		}

		for i := range trans {
			if trans[i].send {
				trans[i].dst = path.Base(trans[i].src)
				trans[i].src = path.Base(trans[i].full)
			} else {
				trans[i].dst = path.Base(trans[i].full)
				trans[i].src = path.Base(trans[i].src)
			}

			if err := db.Exec(`UPDATE transfers SET source_file='%s', dest_file='%s'
				WHERE id=%d`, trans[i].src, trans[i].dst, trans[i].id); err != nil {
				return fmt.Errorf("failed to revert transfer paths formatting: %w", err)
			}
		}

		off += uint64(len(trans))
	}
}

type ver0_5_0TransferFormatLocalPath struct{}

func (ver0_5_0TransferFormatLocalPath) Up(db migration.Actions) (err error) {
	if runtime.GOOS != windowsRuntime {
		return nil // nothing to do
	}

	switch db.GetDialect() {
	case migration.SQLite:
		err = db.Exec(`UPDATE transfers SET 
			local_path = REPLACE(LTRIM(local_path, '/'), '/', '\')`)
	case migration.PostgreSQL:
		err = db.Exec(`UPDATE transfers SET
			local_path = replace(trim(leading '/' from local_path), '/', '\')`)
	case migration.MySQL:
		err = db.Exec(`UPDATE transfers SET
			local_path = REPLACE(TRIM(LEADING '/' FROM local_path), '/', '\\')`)
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	if err != nil {
		return fmt.Errorf("failed to format transfer local path: %w", err)
	}

	return nil
}

func (ver0_5_0TransferFormatLocalPath) Down(db migration.Actions) (err error) {
	if runtime.GOOS != windowsRuntime {
		return nil // nothing to do
	}

	switch db.GetDialect() {
	case migration.SQLite, migration.PostgreSQL:
		if err = db.Exec(`UPDATE transfers SET local_path = ('/' || local_path) 
			WHERE local_path LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the transfer local path: %w", err)
		}

		if err = db.Exec(`UPDATE transfers SET local_path = REPLACE(local_path, '\', '/')`); err != nil {
			return fmt.Errorf("failed to rechange the transfer local path: %w", err)
		}
	case migration.MySQL:
		if err = db.Exec(`UPDATE transfers SET local_path = CONCAT('/', local_path)
			WHERE local_path LIKE '_:%%'`); err != nil {
			return fmt.Errorf("failed to reformat the transfer local path: %w", err)
		}

		if err = db.Exec(`UPDATE transfers SET local_path = REPLACE(local_path, '\\', '/')`); err != nil {
			return fmt.Errorf("failed to rechange the transfer local path: %w", err)
		}
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	return nil
}

type ver0_5_0HistoryPathsChange struct{}

func (ver0_5_0HistoryPathsChange) Up(db migration.Actions) error {
	if err := db.RenameColumn("transfer_history", "dest_filename", "local_path"); err != nil {
		return fmt.Errorf("failed to rename the history 'dest_filename' column: %w", err)
	}

	if err := db.RenameColumn("transfer_history", "source_filename", "remote_path"); err != nil {
		return fmt.Errorf("failed to rename the history 'source_filename' column: %w", err)
	}

	if err := db.SwapColumns("transfer_history", "local_path", "remote_path", "is_send=true"); err != nil {
		return fmt.Errorf("failed to swap the new history path columns: %w", err)
	}

	if runtime.GOOS != windowsRuntime {
		return nil // nothing more to do
	}

	var err error

	switch db.GetDialect() {
	case migration.SQLite:
		err = db.Exec(`UPDATE transfer_history SET 
			local_path = REPLACE(LTRIM(local_path, '/'), '/', '\')`)
	case migration.PostgreSQL:
		err = db.Exec(`UPDATE transfer_history SET
			local_path = replace(trim(leading '/' from local_path), '/', '\')`)
	case migration.MySQL:
		err = db.Exec(`UPDATE transfer_history SET
			local_path = REPLACE(TRIM(LEADING '/' FROM local_path), '/', '\\')`)
	default:
		return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
	}

	if err != nil {
		return fmt.Errorf("failed to format the history local path: %w", err)
	}

	return nil
}

func (ver0_5_0HistoryPathsChange) Down(db migration.Actions) (err error) {
	if runtime.GOOS == windowsRuntime {
		switch db.GetDialect() {
		case migration.SQLite:
			err = db.Exec(`UPDATE transfer_history SET 
			local_path = REPLACE(LTRIM(local_path, '/'), '/', '\')`)
		case migration.PostgreSQL:
			err = db.Exec(`UPDATE transfer_history SET
			local_path = replace(trim(leading '/' from local_path), '/', '\')`)
		case migration.MySQL:
			err = db.Exec(`UPDATE transfer_history SET
			local_path = REPLACE(TRIM(LEADING '/' FROM local_path), '/', '\\')`)
		default:
			return fmt.Errorf("unknown dialect engine %T: %w", db, errUnsuportedDB)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to reformat the history local path: %w", err)
	}

	if err := db.SwapColumns("transfer_history", "local_path", "remote_path", "is_send=true"); err != nil {
		return fmt.Errorf("failed to reswap the new history path columns: %w", err)
	}

	if err := db.RenameColumn("transfer_history", "local_path", "dest_filename"); err != nil {
		return fmt.Errorf("failed to restore the history 'dest_filename' column: %w", err)
	}

	if err := db.RenameColumn("transfer_history", "remote_path", "source_filename"); err != nil {
		return fmt.Errorf("failed to restore the history 'source_filename' column: %w", err)
	}

	return nil
}

type ver0_5_0LocalAccountPasswordHash struct{}

func (ver0_5_0LocalAccountPasswordHash) Up(db migration.Actions) error {
	if err := db.RenameColumn("local_accounts", "password", "password_hash"); err != nil {
		return fmt.Errorf("failed to rename the local account 'password' column: %w", err)
	}

	return nil
}

func (ver0_5_0LocalAccountPasswordHash) Down(db migration.Actions) error {
	if err := db.RenameColumn("local_accounts", "password_hash", "password"); err != nil {
		return fmt.Errorf("failed to restore the local account 'password' column: %w", err)
	}

	return nil
}

type ver0_5_0CertificatesRename struct{}

func (ver0_5_0CertificatesRename) Up(db migration.Actions) error {
	if err := db.RenameTable("certificates", "crypto_credentials"); err != nil {
		return fmt.Errorf("failed to rename the 'certificates' table: %w", err)
	}

	return nil
}

func (ver0_5_0CertificatesRename) Down(db migration.Actions) error {
	if err := db.RenameTable("crypto_credentials", "certificates"); err != nil {
		return fmt.Errorf("failed to restore the 'certificates' table: %w", err)
	}

	return nil
}
