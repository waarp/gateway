package migrations

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
)

func ver0_5_0RemoveRulePathSlashUp(db Actions) error {
	var err error

	switch dial := db.GetDialect(); dial {
	case SQLite:
		err = db.Exec(`UPDATE rules SET path=LTRIM(path, '/')`)
	case PostgreSQL, MySQL:
		err = db.Exec(`UPDATE rules SET path=TRIM(LEADING '/' FROM path)`)
	default:
		return errUnknownDialect(dial)
	}

	if err != nil {
		return fmt.Errorf("failed to trim the rules' paths: %w", err)
	}

	return nil
}

func ver0_5_0RemoveRulePathSlashDown(db Actions) error {
	var err error

	switch dial := db.GetDialect(); dial {
	case SQLite, PostgreSQL:
		err = db.Exec(`UPDATE rules SET path='/' || path`)
	case MySQL:
		err = db.Exec(`UPDATE rules SET path=CONCAT("/", path)`)
	default:
		return errUnknownDialect(dial)
	}

	if err != nil {
		return fmt.Errorf("failed to restore the rules' paths: %w", err)
	}

	return nil
}

func ver0_5_0CheckRulePathParentUp(db Actions) error {
	var query string

	switch dial := db.GetDialect(); dial {
	case SQLite, PostgreSQL:
		query = `SELECT A.name, A.path, B.name, B.path FROM rules A, rules B WHERE B.path LIKE A.path || '/%'`
	case MySQL:
		query = `SELECT A.name, A.path, B.name, B.path FROM rules A, rules B WHERE B.path LIKE CONCAT(A.path, '/%')`
	default:
		return errUnknownDialect(dial)
	}

	rows, err := db.Query(query)
	if err != nil || rows.Err() != nil {
		return fmt.Errorf("failed to retrieve the rule entries: %w", err)
	}

	defer rows.Close() //nolint:errcheck //this error is unimportant

	if rows.Next() {
		var aName, aPath, bName, bPath string
		if err = rows.Scan(&aName, &aPath, &bName, &bPath); err != nil {
			return fmt.Errorf("failed to parse the rule entry: %w", err)
		}

		//nolint:err113 //this is a base error
		return fmt.Errorf("the path of the rule '%s' (%s) must be changed so "+
			"that it is no longer a parent of the path of rule '%s' (%s)",
			aName, aPath, bName, bPath)
	}

	return nil
}

func ver0_5_0LocalAgentDenormalizePathsUp(db Actions) (err error) {
	if !isWindowsRuntime() {
		return nil // nothing to do
	}

	switch dial := db.GetDialect(); dial {
	case SQLite:
		err = db.Exec(`UPDATE local_agents SET 
			root = REPLACE(LTRIM(root, '/'), '/', '\'),
			in_dir = REPLACE(LTRIM(in_dir, '/'), '/', '\'),
			out_dir = REPLACE(LTRIM(out_dir, '/'), '/', '\'),
			work_dir = REPLACE(LTRIM(work_dir, '/'), '/', '\')`)
	case PostgreSQL:
		err = db.Exec(`UPDATE local_agents SET
			root = replace(trim(leading '/' from root), '/', '\'),
			in_dir = replace(trim(leading '/' from in_dir), '/', '\'),
			out_dir = replace(trim(leading '/' from out_dir), '/', '\'),
			work_dir = replace(trim(leading '/' from work_dir), '/', '\')`)
	case MySQL:
		err = db.Exec(`UPDATE local_agents SET
			root = REPLACE(TRIM(LEADING '/' FROM root), '/', '\\'),
			in_dir = REPLACE(TRIM(LEADING '/' FROM in_dir), '/', '\\'),
			out_dir = REPLACE(TRIM(LEADING '/' FROM out_dir), '/', '\\'),
			work_dir = REPLACE(TRIM(LEADING '/' FROM work_dir), '/', '\\')`)
	default:
		return errUnknownDialect(dial)
	}

	if err != nil {
		return fmt.Errorf("failed to change server paths: %w", err)
	}

	return nil
}

func ver0_5_0LocalAgentDenormalizePathsDown(db Actions) (err error) {
	if !isWindowsRuntime() {
		return nil // nothing to do
	}

	switch dial := db.GetDialect(); dial {
	case SQLite, PostgreSQL:
		if err = db.Exec(`UPDATE local_agents SET root = ('/' || root) WHERE root LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the server root directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET in_dir = ('/' || in_dir) 
			WHERE in_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the server in directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET out_dir = ('/' || out_dir)
			WHERE out_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the server out directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET work_dir = ('/' || work_dir)
			WHERE work_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the server tmp directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET root = REPLACE(root, '\', '/'), 
			in_dir = REPLACE(in_dir, '\', '/'),	out_dir = REPLACE(out_dir, '\', '/'), 
			work_dir = REPLACE(work_dir, '\', '/')`); err != nil {
			return fmt.Errorf("failed to change server paths: %w", err)
		}
	case MySQL:
		if err = db.Exec(`UPDATE local_agents SET root = CONCAT('/', root) WHERE root LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the server root directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET in_dir = CONCAT('/', in_dir) 
			WHERE in_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the server in directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET out_dir = CONCAT('/', out_dir) 
			WHERE out_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the server out directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET work_dir = CONCAT('/', work_dir) 
			WHERE work_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the server tmp directory: %w", err)
		}

		if err = db.Exec(`UPDATE local_agents SET root = REPLACE(root, '\\', '/'),
			in_dir = REPLACE(in_dir, '\\', '/'), out_dir = REPLACE(out_dir, '\\', '/'),
			work_dir = REPLACE(work_dir, '\\', '/')`); err != nil {
			return fmt.Errorf("failed to change server paths: %w", err)
		}
	default:
		return errUnknownDialect(dial)
	}

	return nil
}

func ver0_5_0LocalAgentsPathsRenameUp(db Actions) error {
	if err := db.AlterTable("local_agents",
		RenameColumn{OldName: "root", NewName: "root_dir"},
		RenameColumn{OldName: "in_dir", NewName: "receive_dir"},
		RenameColumn{OldName: "out_dir", NewName: "send_dir"},
		RenameColumn{OldName: "work_dir", NewName: "tmp_receive_dir"},
	); err != nil {
		return fmt.Errorf("failed to rename the local agent columns: %w", err)
	}

	return nil
}

func ver0_5_0LocalAgentsPathsRenameDown(db Actions) error {
	if err := db.AlterTable("local_agents",
		RenameColumn{OldName: "root_dir", NewName: "root"},
		RenameColumn{OldName: "receive_dir", NewName: "in_dir"},
		RenameColumn{OldName: "send_dir", NewName: "out_dir"},
		RenameColumn{OldName: "tmp_receive_dir", NewName: "work_dir"},
	); err != nil {
		return fmt.Errorf("failed to restore the local agent column names: %w", err)
	}

	return nil
}

func ver0_5_0LocalAgentsDisallowReservedNamesUp(db Actions) error {
	rows, err := db.Query(`SELECT name FROM local_agents`)
	if err != nil {
		return fmt.Errorf("failed to retrieve local servers list: %w", err)
	}

	defer rows.Close() //nolint:errcheck //this error is irrelevant

	if rows.Err() != nil {
		return fmt.Errorf("failed to retrieve local servers list: %w", rows.Err())
	}

	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return fmt.Errorf("failed to retrieve server name: %w", err)
		}

		if name == "Database" || name == "Admin" || name == "Controller" {
			//nolint:err113 //this is a base error
			return fmt.Errorf("%q is a reserved service name, this server should be renamed",
				name)
		}
	}

	return nil
}

func ver0_5_0RulesPathsRenameUp(db Actions) error {
	if err := db.AlterTable("rules",
		RenameColumn{OldName: "in_path", NewName: "local_dir"},
		RenameColumn{OldName: "out_path", NewName: "remote_dir"},
		RenameColumn{OldName: "work_path", NewName: "tmp_local_receive_dir"},
	); err != nil {
		return fmt.Errorf("failed to rename the rule columns: %w", err)
	}

	if err := db.SwapColumns("rules", "local_dir", "remote_dir", "send=true"); err != nil {
		return fmt.Errorf("failed to swap the rule path columns values: %w", err)
	}

	return nil
}

func ver0_5_0RulesPathsRenameDown(db Actions) error {
	if err := db.SwapColumns("rules", "local_dir", "remote_dir", "send=true"); err != nil {
		return fmt.Errorf("failed to re-swap the rule path columns values: %w", err)
	}

	if err := db.AlterTable("rules",
		RenameColumn{OldName: "local_dir", NewName: "in_path"},
		RenameColumn{OldName: "remote_dir", NewName: "out_path"},
		RenameColumn{OldName: "tmp_local_receive_dir", NewName: "work_path"},
	); err != nil {
		return fmt.Errorf("failed to restore the rule column names: %w", err)
	}

	return nil
}

func ver0_5_0RulePathChangesUp(db Actions) (err error) {
	if !isWindowsRuntime() {
		return nil // nothing to do
	}

	switch dial := db.GetDialect(); dial {
	case SQLite:
		err = db.Exec(`UPDATE rules SET 
			local_dir = REPLACE(LTRIM(local_dir, '/'), '/', '\'),
			tmp_local_receive_dir = REPLACE(LTRIM(tmp_local_receive_dir, '/'), '/', '\')`)
	case PostgreSQL:
		err = db.Exec(`UPDATE rules SET
			local_dir = replace(trim(leading '/' from local_dir), '/', '\'),
			tmp_local_receive_dir = replace(trim(leading '/' from tmp_local_receive_dir), '/', '\')`)
	case MySQL:
		err = db.Exec(`UPDATE rules SET
			local_dir = REPLACE(TRIM(LEADING '/' FROM local_dir), '/', '\\'),
			tmp_local_receive_dir = REPLACE(TRIM(LEADING '/' FROM tmp_local_receive_dir), '/', '\\')`)
	default:
		return errUnknownDialect(dial)
	}

	if err != nil {
		return fmt.Errorf("failed to change rule paths: %w", err)
	}

	return nil
}

func ver0_5_0RulePathChangesDown(db Actions) (err error) {
	if !isWindowsRuntime() {
		return nil // nothing to do
	}

	switch dial := db.GetDialect(); dial {
	case MySQL:
		if err = db.Exec(`UPDATE rules SET local_dir = CONCAT('/', local_dir) 
			WHERE local_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to restore rule local directory: %w", err)
		}

		if err = db.Exec(`UPDATE rules SET tmp_local_receive_dir = CONCAT('/', tmp_local_receive_dir) 
			WHERE tmp_local_receive_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to restore rule tmp directory: %w", err)
		}
	default:
		if err = db.Exec(`UPDATE rules SET local_dir = ('/' || local_dir) 
			WHERE local_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to restore rule local directory: %w", err)
		}

		if err = db.Exec(`UPDATE rules SET tmp_local_receive_dir = ('/' || tmp_local_receive_dir) 
			WHERE tmp_local_receive_dir LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to restore rule tmp directory: %w", err)
		}
	}

	switch dial := db.GetDialect(); dial {
	case SQLite:
		err = db.Exec(`UPDATE rules SET 
			local_dir = REPLACE(local_dir, '\', '/'),
			tmp_local_receive_dir = REPLACE(tmp_local_receive_dir, '\', '/')`)
	case PostgreSQL:
		err = db.Exec(`UPDATE rules SET
			local_dir = replace(local_dir, '\', '/'),
			tmp_local_receive_dir = replace(tmp_local_receive_dir, '\', '/')`)
	default: // MySQL
		err = db.Exec(`UPDATE rules SET
			local_dir = REPLACE(local_dir, '\\', '/'),
			tmp_local_receive_dir = REPLACE(tmp_local_receive_dir, '\\', '/')`)
	}

	if err != nil {
		return fmt.Errorf("failed to revert rule paths: %w", err)
	}

	return nil
}

func ver0_5_0AddFilesizeUp(db Actions) error {
	if err := db.AlterTable("transfers",
		AddColumn{Name: "filesize", Type: BigInt{}, NotNull: true, Default: -1},
	); err != nil {
		return fmt.Errorf("failed to add transfer 'filesize' column: %w", err)
	}

	if err := db.AlterTable("transfer_history",
		AddColumn{Name: "filesize", Type: BigInt{}, NotNull: true, Default: -1},
	); err != nil {
		return fmt.Errorf("failed to add history 'filesize' column: %w", err)
	}

	return nil
}

func ver0_5_0AddFilesizeDown(db Actions) error {
	if err := db.AlterTable("transfers", DropColumn{Name: "filesize"}); err != nil {
		return fmt.Errorf("failed to drop transfer 'filesize' column: %w", err)
	}

	if err := db.AlterTable("transfer_history", DropColumn{Name: "filesize"}); err != nil {
		return fmt.Errorf("failed to drop history 'filesize' column: %w", err)
	}

	return nil
}

func ver0_5_0TransferChangePathsUp(db Actions) error {
	if err := db.AlterTable("transfers",
		RenameColumn{OldName: "true_filepath", NewName: "local_path"},
		RenameColumn{OldName: "source_file", NewName: "remote_path"},
	); err != nil {
		return fmt.Errorf("failed to rename the transfer columns: %w", err)
	}

	if err := db.Exec(`UPDATE transfers SET remote_path = dest_file WHERE 
		rule_id IN (SELECT id FROM rules WHERE send=true)`); err != nil {
		return fmt.Errorf("failed to fill the new 'remote_path' transfer column: %w", err)
	}

	var err error

	switch dial := db.GetDialect(); dial {
	case MySQL:
		err = db.Exec(`UPDATE transfers SET remote_path = CONCAT((SELECT remote_dir 
			FROM rules WHERE id = transfers.rule_id), '/', transfers.remote_path)`)
	default:
		err = db.Exec(`UPDATE transfers SET remote_path = (SELECT remote_dir FROM 
			rules WHERE id = transfers.rule_id) || '/' || transfers.remote_path`)
	}

	if err != nil {
		return fmt.Errorf("failed to format the `remote_path` transfer values: %w", err)
	}

	if err = db.AlterTable("transfers", DropColumn{Name: "dest_file"}); err != nil {
		return fmt.Errorf("failed to drop the 'dest_file' transfer column: %w", err)
	}

	return nil
}

func ver0_5_0TransferChangePathsDown(db Actions) error {
	if err := db.AlterTable("transfers",
		RenameColumn{OldName: "local_path", NewName: "true_filepath"},
		RenameColumn{OldName: "remote_path", NewName: "source_file"},
		AddColumn{Name: "dest_file", Type: Varchar(255), NotNull: true, Default: ""},
	); err != nil {
		return fmt.Errorf("failed to restore the transfer columns: %w", err)
	}

	if db.GetDialect() == MySQL {
		if err := db.Exec(`UPDATE transfers SET 
        	source_file=(@temp:=source_file),
    		source_file=(SUBSTRING_INDEX(true_filepath, '/', -1)),
    		dest_file  =(SUBSTRING_INDEX(@temp, '/', -1))
		WHERE (SELECT send FROM rules WHERE id=rule_id)`); err != nil {
			return fmt.Errorf("failed to restore the transfer paths: %w", err)
		}

		if err := db.Exec(`UPDATE transfers SET 
    		dest_file  =(SUBSTRING_INDEX(true_filepath, '/', -1)),
    		source_file=(SUBSTRING_INDEX(source_file, '/', -1))
		WHERE NOT (SELECT send FROM rules WHERE id=rule_id)`); err != nil {
			return fmt.Errorf("failed to restore the transfer paths: %w", err)
		}

		return nil
	}

	if err := db.Exec(`UPDATE transfers SET 
    	source_file=(SELECT REPLACE(true_filepath, RTRIM(true_filepath, REPLACE(true_filepath, '/', '')), '')),
    	dest_file  =(SELECT REPLACE(source_file, RTRIM(source_file, REPLACE(source_file, '/', '')), ''))
	WHERE (SELECT send FROM rules WHERE id=rule_id)`); err != nil {
		return fmt.Errorf("failed to restore the transfer paths: %w", err)
	}

	if err := db.Exec(`UPDATE transfers SET 
    	dest_file  =(SELECT REPLACE(true_filepath, RTRIM(true_filepath, REPLACE(true_filepath, '/', '')), '')),
    	source_file=(SELECT REPLACE(source_file, RTRIM(source_file, REPLACE(source_file, '/', '')), ''))
	WHERE NOT (SELECT send FROM rules WHERE id=rule_id)`); err != nil {
		return fmt.Errorf("failed to restore the transfer paths: %w", err)
	}

	return nil
}

func ver0_5_0TransferFormatLocalPathUp(db Actions) (err error) {
	if !isWindowsRuntime() {
		return nil // nothing to do
	}

	switch dial := db.GetDialect(); dial {
	case SQLite:
		err = db.Exec(`UPDATE transfers SET 
			local_path = REPLACE(LTRIM(local_path, '/'), '/', '\')`)
	case PostgreSQL:
		err = db.Exec(`UPDATE transfers SET
			local_path = replace(trim(leading '/' from local_path), '/', '\')`)
	case MySQL:
		err = db.Exec(`UPDATE transfers SET
			local_path = REPLACE(TRIM(LEADING '/' FROM local_path), '/', '\\')`)
	default:
		return errUnknownDialect(dial)
	}

	if err != nil {
		return fmt.Errorf("failed to format transfer local path: %w", err)
	}

	return nil
}

func ver0_5_0TransferFormatLocalPathDown(db Actions) (err error) {
	if !isWindowsRuntime() {
		return nil // nothing to do
	}

	switch dial := db.GetDialect(); dial {
	case SQLite, PostgreSQL:
		if err = db.Exec(`UPDATE transfers SET local_path = ('/' || local_path) 
			WHERE local_path LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the transfer local path: %w", err)
		}

		if err = db.Exec(`UPDATE transfers SET local_path = REPLACE(local_path, '\', '/')`); err != nil {
			return fmt.Errorf("failed to rechange the transfer local path: %w", err)
		}
	case MySQL:
		if err = db.Exec(`UPDATE transfers SET local_path = CONCAT('/', local_path)
			WHERE local_path LIKE '_:%'`); err != nil {
			return fmt.Errorf("failed to reformat the transfer local path: %w", err)
		}

		if err = db.Exec(`UPDATE transfers SET local_path = REPLACE(local_path, '\\', '/')`); err != nil {
			return fmt.Errorf("failed to rechange the transfer local path: %w", err)
		}
	default:
		return errUnknownDialect(dial)
	}

	return nil
}

func ver0_5_0HistoryPathsChangeUp(db Actions) error {
	if err := db.AlterTable("transfer_history",
		RenameColumn{OldName: "dest_filename", NewName: "local_path"},
		RenameColumn{OldName: "source_filename", NewName: "remote_path"},
	); err != nil {
		return fmt.Errorf("failed to rename the history columns: %w", err)
	}

	if err := db.SwapColumns("transfer_history", "local_path", "remote_path", "is_send=true"); err != nil {
		return fmt.Errorf("failed to swap the new history path columns: %w", err)
	}

	if !isWindowsRuntime() {
		return nil // nothing more to do
	}

	var err error

	switch dial := db.GetDialect(); dial {
	case SQLite:
		err = db.Exec(`UPDATE transfer_history SET 
			local_path = REPLACE(LTRIM(local_path, '/'), '/', '\')`)
	case PostgreSQL:
		err = db.Exec(`UPDATE transfer_history SET
			local_path = replace(trim(leading '/' from local_path), '/', '\')`)
	case MySQL:
		err = db.Exec(`UPDATE transfer_history SET
			local_path = REPLACE(TRIM(LEADING '/' FROM local_path), '/', '\\')`)
	default:
		return errUnknownDialect(dial)
	}

	if err != nil {
		return fmt.Errorf("failed to format the history local path: %w", err)
	}

	return nil
}

func ver0_5_0HistoryPathsChangeDown(db Actions) (err error) {
	if isWindowsRuntime() {
		switch dial := db.GetDialect(); dial {
		case SQLite:
			err = db.Exec(`UPDATE transfer_history SET 
			local_path = REPLACE(LTRIM(local_path, '/'), '/', '\')`)
		case PostgreSQL:
			err = db.Exec(`UPDATE transfer_history SET
			local_path = replace(trim(leading '/' from local_path), '/', '\')`)
		case MySQL:
			err = db.Exec(`UPDATE transfer_history SET
			local_path = REPLACE(TRIM(LEADING '/' FROM local_path), '/', '\\')`)
		default:
			return errUnknownDialect(dial)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to reformat the history local path: %w", err)
	}

	if err = db.SwapColumns("transfer_history", "local_path", "remote_path", "is_send=true"); err != nil {
		return fmt.Errorf("failed to reswap the new history path columns: %w", err)
	}

	if err = db.AlterTable("transfer_history",
		RenameColumn{OldName: "local_path", NewName: "dest_filename"},
		RenameColumn{OldName: "remote_path", NewName: "source_filename"},
	); err != nil {
		return fmt.Errorf("failed to restore the history columns: %w", err)
	}

	return nil
}

func ver0_5_0LocalAccountsPasswordDecodeUp(db Actions) error {
	for i := 0; true; i++ {
		row := db.QueryRow(`SELECT id,password_hash FROM local_accounts 
			ORDER BY id LIMIT 1 OFFSET ?`, i)

		var (
			id   int64
			hash string
		)

		if err := row.Scan(&id, &hash); errors.Is(err, sql.ErrNoRows) {
			break
		} else if err != nil {
			return fmt.Errorf("failed to parse account information: %w", err)
		}

		dec, decErr := base64.StdEncoding.DecodeString(hash)
		if decErr != nil {
			return fmt.Errorf("failed to decode password hash: %w", decErr)
		}

		if err := db.Exec("UPDATE local_accounts SET password_hash=? WHERE id=?",
			dec, id); err != nil {
			return fmt.Errorf("failed to update account entry: %w", err)
		}
	}

	return nil
}

func ver0_5_0LocalAccountsPasswordDecodeDown(db Actions) error {
	for i := 0; true; i++ {
		row := db.QueryRow(`SELECT id,password_hash FROM local_accounts
			ORDER BY id LIMIT 1 OFFSET ?`, i)

		var (
			id   int64
			hash string
		)

		if err := row.Scan(&id, &hash); errors.Is(err, sql.ErrNoRows) {
			break
		} else if err != nil {
			return fmt.Errorf("failed to parse account information: %w", err)
		}

		if err := db.Exec("UPDATE local_accounts SET password_hash=? WHERE id=?",
			base64.StdEncoding.EncodeToString([]byte(hash)), id); err != nil {
			return fmt.Errorf("failed to update account entry: %w", err)
		}
	}

	return nil
}

func ver0_5_0UserPasswordChangeUp(db Actions) error {
	if err := db.AlterTable("users",
		AddColumn{Name: "password_hash", Type: Text{}, NotNull: true, Default: ""},
	); err != nil {
		return fmt.Errorf("failed to add the user 'password_hash' column: %w", err)
	}

	if db.GetDialect() == PostgreSQL {
		if err := db.Exec("UPDATE users SET password_hash=encode(password, 'escape')"); err != nil {
			return fmt.Errorf("failed to update user entries: %w", err)
		}
	} else {
		if err := db.Exec("UPDATE users SET password_hash=password"); err != nil {
			return fmt.Errorf("failed to update user entries: %w", err)
		}
	}

	if err := db.AlterTable("users", DropColumn{Name: "password"}); err != nil {
		return fmt.Errorf("failed to drop the user 'password' column: %w", err)
	}

	return nil
}

func ver0_5_0UserPasswordChangeDown(db Actions) error {
	if err := db.AlterTable("users",
		AddColumn{Name: "password", Type: Blob{}, NotNull: true, Default: []byte{}},
	); err != nil {
		return fmt.Errorf("failed to add the user 'password' column: %w", err)
	}

	if db.GetDialect() == PostgreSQL {
		if err := db.Exec("UPDATE users SET password=decode(password_hash, 'escape')"); err != nil {
			return fmt.Errorf("failed to update user entries: %w", err)
		}
	} else {
		if err := db.Exec("UPDATE users SET password=password_hash"); err != nil {
			return fmt.Errorf("failed to update user entries: %w", err)
		}
	}

	if err := db.AlterTable("users",
		AlterColumn{Name: "password", Type: Blob{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to modify the user 'password' column: %w", err)
	}

	if err := db.AlterTable("users", DropColumn{Name: "password_hash"}); err != nil {
		return fmt.Errorf("failed to drop the user 'password_hash' column: %w", err)
	}

	return nil
}
