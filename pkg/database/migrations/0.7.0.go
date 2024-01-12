package migrations

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func ver0_7_0AddLocalAgentEnabledColumnUp(db Actions) error {
	if err := db.AlterTable("local_agents",
		AddColumn{Name: "enabled", Type: Boolean{}, NotNull: true, Default: true},
	); err != nil {
		return fmt.Errorf("failed to add the local agent 'enabled' column: %w", err)
	}

	return nil
}

func ver0_7_0AddLocalAgentEnabledColumnDown(db Actions) error {
	if err := db.AlterTable("local_agents", DropColumn{Name: "enabled"}); err != nil {
		return fmt.Errorf("failed to drop the local agent 'enabled' column: %w", err)
	}

	return nil
}

func ver0_7_0RevampUsersTableUp(db Actions) error {
	if err := db.DropIndex(quote(db, "UQE_users_name"), "users"); err != nil {
		return fmt.Errorf("failed to drop the user index: %w", err)
	}

	if err := db.AlterTable("users",
		RenameColumn{OldName: "permissions", NewName: "bytes_permissions"},
		AlterColumn{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "owner", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "username", Type: Varchar(100), NotNull: true},
		AddColumn{Name: "int_permissions", Type: Integer{}, NotNull: true, Default: 0},
		AddUnique{Name: "unique_user", Cols: []string{"owner", "username"}},
	); err != nil {
		return fmt.Errorf("failed to alter the users table: %w", err)
	}

	for i := 0; true; i++ {
		var (
			id        int64
			permBytes []byte
		)

		row := db.QueryRow(`SELECT id, bytes_permissions FROM users 
								ORDER BY id LIMIT 1 OFFSET ?`, i)
		if err := row.Scan(&id, &permBytes); errors.Is(err, sql.ErrNoRows) {
			break
		} else if err != nil {
			return fmt.Errorf("failed to parse the user permissions: %w", err)
		}

		permUint := bits.Reverse32(binary.LittleEndian.Uint32(permBytes))

		if err := db.Exec("UPDATE users SET int_permissions=? WHERE id=?",
			int32(permUint), id); err != nil {
			return fmt.Errorf("failed to update the user permissions: %w", err)
		}
	}

	if err := db.AlterTable("users",
		DropColumn{Name: "bytes_permissions"},
		RenameColumn{OldName: "int_permissions", NewName: "permissions"},
	); err != nil {
		return fmt.Errorf("failed to alter the users table: %w", err)
	}

	return nil
}

func ver0_7_0RevampUsersTableDown(db Actions) error {
	if err := db.AlterTable("users",
		RenameColumn{OldName: "permissions", NewName: "int_permissions"},
		DropConstraint{Name: "unique_user"},
		AlterColumn{Name: "id", Type: UnsignedBigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "owner", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "username", Type: Varchar(255), NotNull: true},
		AddColumn{Name: "bytes_permissions", Type: Binary(4)},
	); err != nil {
		return fmt.Errorf("failed to restore the users table: %w", err)
	}

	for {
		var (
			id      int64
			permInt int32
		)

		row := db.QueryRow(`SELECT id, int_permissions FROM users 
                           		WHERE bytes_permissions IS NULL`)
		if err := row.Scan(&id, &permInt); errors.Is(err, sql.ErrNoRows) {
			break
		} else if err != nil {
			return fmt.Errorf("failed to parse the user permissions: %w", err)
		}

		permBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(permBytes, bits.Reverse32(uint32(permInt)))

		if err := db.Exec("UPDATE users SET bytes_permissions=? WHERE id=?",
			permBytes, id); err != nil {
			return fmt.Errorf("failed to update the user permissions: %w", err)
		}
	}

	if err := db.AlterTable("users",
		DropColumn{Name: "int_permissions"},
		AlterColumn{Name: "bytes_permissions", NewName: "permissions", Type: Binary(4), NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to alter the users table: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_users_name"), Unique: true,
		On: "users", Cols: []string{"owner", "username"},
	}); err != nil {
		return fmt.Errorf("failed to restore the user index: %w", err)
	}

	return nil
}

func ver0_7_0RevampLocalAgentsTableUp(db Actions) error {
	if err := db.DropIndex(quote(db, "UQE_local_agents_loc_ag"), "local_agents"); err != nil {
		return fmt.Errorf("failed to drop the old local_agent index: %w", err)
	}

	if err := db.AlterTable("local_agents",
		RenameColumn{OldName: "proto_config", NewName: "blob_proto_config"},
		AlterColumn{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "owner", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "name", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "protocol", Type: Varchar(50), NotNull: true},
		AlterColumn{Name: "root_dir", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "receive_dir", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "send_dir", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "tmp_receive_dir", Type: Text{}, NotNull: true, Default: ""},
		AddColumn{Name: "text_proto_config", Type: Text{}, NotNull: true, Default: "{}"},
		AddUnique{Name: "unique_local_agent", Cols: []string{"owner", "name"}},
	); err != nil {
		return fmt.Errorf("failed to alter the local_agents table: %w", err)
	}

	protoConfQuery := utils.If(db.GetDialect() == PostgreSQL,
		"UPDATE local_agents SET text_proto_config=ENCODE(blob_proto_config, 'escape')",
		"UPDATE local_agents SET text_proto_config=blob_proto_config")
	if err := db.Exec(protoConfQuery); err != nil {
		return fmt.Errorf("failed to update the proto_config: %w", err)
	}

	if err := db.AlterTable("local_agents",
		DropColumn{Name: "blob_proto_config"},
		RenameColumn{OldName: "text_proto_config", NewName: "proto_config"},
	); err != nil {
		return fmt.Errorf("failed to alter the local_agents table: %w", err)
	}

	return nil
}

func ver0_7_0RevampLocalAgentsTableDown(db Actions) error {
	if err := db.AlterTable("local_agents",
		DropConstraint{Name: "unique_local_agent"},
		RenameColumn{OldName: "proto_config", NewName: "text_proto_config"},
		AlterColumn{Name: "id", Type: UnsignedBigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "owner", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "name", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "protocol", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "root_dir", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "receive_dir", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "send_dir", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "tmp_receive_dir", Type: Varchar(255), NotNull: true},
		AddColumn{Name: "blob_proto_config", Type: Blob{} /*NotNull: true*/},
	); err != nil {
		return fmt.Errorf("failed to alter the local_agents table: %w", err)
	}

	protoConfQuery := utils.If(db.GetDialect() == PostgreSQL,
		"UPDATE local_agents SET blob_proto_config=DECODE(text_proto_config, 'escape')",
		"UPDATE local_agents SET blob_proto_config=text_proto_config")
	if err := db.Exec(protoConfQuery); err != nil {
		return fmt.Errorf("failed to update the proto_config: %w", err)
	}

	if err := db.AlterTable("local_agents",
		DropColumn{Name: "text_proto_config"},
		AlterColumn{Name: "blob_proto_config", NewName: "proto_config", Type: Blob{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to alter the local_agents table: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_local_agents_loc_ag"), Unique: true,
		On: "local_agents", Cols: []string{"owner", "name"},
	}); err != nil {
		return fmt.Errorf("failed to restore the old local_agent index: %w", err)
	}

	return nil
}

func ver0_7_0RevampRemoteAgentsTableUp(db Actions) error {
	if err := db.DropIndex(quote(db, "UQE_remote_agents_name"), "remote_agents"); err != nil {
		return fmt.Errorf("failed to drop the old remote_agent index: %w", err)
	}

	if err := db.AlterTable("remote_agents",
		RenameColumn{OldName: "proto_config", NewName: "blob_proto_config"},
		AlterColumn{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "name", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "protocol", Type: Varchar(50), NotNull: true},
		AddColumn{Name: "text_proto_config", Type: Text{}, NotNull: true, Default: "{}"},
		AddUnique{Name: "unique_remote_agent", Cols: []string{"name"}},
	); err != nil {
		return fmt.Errorf("failed to alter the remote_agents table: %w", err)
	}

	protoConfQuery := utils.If(db.GetDialect() == PostgreSQL,
		"UPDATE remote_agents SET text_proto_config=ENCODE(blob_proto_config, 'escape')",
		"UPDATE remote_agents SET text_proto_config=blob_proto_config")
	if err := db.Exec(protoConfQuery); err != nil {
		return fmt.Errorf("failed to update the proto_config: %w", err)
	}

	if err := db.AlterTable("remote_agents",
		DropColumn{Name: "blob_proto_config"},
		RenameColumn{OldName: "text_proto_config", NewName: "proto_config"},
	); err != nil {
		return fmt.Errorf("failed to alter the remote_agents table: %w", err)
	}

	return nil
}

func ver0_7_0RevampRemoteAgentsTableDown(db Actions) error {
	if err := db.AlterTable("remote_agents",
		DropConstraint{Name: "unique_remote_agent"},
		RenameColumn{OldName: "proto_config", NewName: "text_proto_config"},
		AlterColumn{Name: "id", Type: UnsignedBigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "name", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "protocol", Type: Varchar(255), NotNull: true},
		AddColumn{Name: "blob_proto_config", Type: Blob{} /*NotNull: true*/},
	); err != nil {
		return fmt.Errorf("failed to recreate the old remote_agents table: %w", err)
	}

	protoConfQuery := utils.If(db.GetDialect() == PostgreSQL,
		"UPDATE remote_agents SET blob_proto_config=DECODE(text_proto_config, 'escape')",
		"UPDATE remote_agents SET blob_proto_config=text_proto_config")
	if err := db.Exec(protoConfQuery); err != nil {
		return fmt.Errorf("failed to update the proto_config: %w", err)
	}

	if err := db.AlterTable("remote_agents",
		DropColumn{Name: "text_proto_config"},
		AlterColumn{
			Name: "blob_proto_config", NewName: "proto_config",
			Type: Blob{}, NotNull: true,
		},
	); err != nil {
		return fmt.Errorf("failed to alter the remote_agents table: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_remote_agents_name"), Unique: true,
		On: "remote_agents", Cols: []string{"name"},
	}); err != nil {
		return fmt.Errorf("failed to restore the old remote_agent index: %w", err)
	}

	return nil
}

func ver0_7_0RevampLocalAccountsTableUp(db Actions) error {
	if err := db.DropIndex(quote(db, "UQE_local_accounts_loc_ac"), "local_accounts"); err != nil {
		return fmt.Errorf("failed to drop the old local_account index: %w", err)
	}

	if err := db.AlterTable("local_accounts",
		AlterColumn{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "local_agent_id", Type: BigInt{}, NotNull: true},
		AlterColumn{Name: "login", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "password_hash", Type: Text{}, NotNull: true, Default: ""},
		AddForeignKey{
			Name: "local_accounts_agent_fkey", Cols: []string{"local_agent_id"},
			RefTbl: "local_agents", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddUnique{Name: "unique_local_account", Cols: []string{"local_agent_id", "login"}},
	); err != nil {
		return fmt.Errorf("failed to alter the local_accounts table: %w", err)
	}

	return nil
}

func ver0_7_0RevampLocalAccountsTableDown(db Actions) error {
	if err := db.AlterTable("local_accounts",
		DropConstraint{Name: "local_accounts_agent_fkey"},
		DropConstraint{Name: "unique_local_account"},
		AlterColumn{Name: "id", Type: UnsignedBigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "local_agent_id", Type: UnsignedBigInt{}, NotNull: true},
		AlterColumn{Name: "login", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "password_hash", Type: Text{}},
	); err != nil {
		return fmt.Errorf("failed to alter the local_accounts table: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_local_accounts_loc_ac"), Unique: true,
		On: "local_accounts", Cols: []string{"local_agent_id", "login"},
	}); err != nil {
		return fmt.Errorf("failed to restore the old local_account index: %w", err)
	}

	return nil
}

func ver0_7_0RevampRemoteAccountsTableUp(db Actions) error {
	if err := db.DropIndex(quote(db, "UQE_remote_accounts_rem_ac"), "remote_accounts"); err != nil {
		return fmt.Errorf("failed to drop the old remote_account index: %w", err)
	}

	if err := db.Exec(`UPDATE remote_accounts SET password='' WHERE password IS NULL`); err != nil {
		return fmt.Errorf("failed to update the remote_accounts passwords: %w", err)
	}

	if err := db.AlterTable("remote_accounts",
		AlterColumn{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "remote_agent_id", Type: BigInt{}, NotNull: true},
		AlterColumn{Name: "login", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "password", Type: Text{}, NotNull: true, Default: ""},
		AddForeignKey{
			Name: "remote_accounts_agent_fkey", Cols: []string{"remote_agent_id"},
			RefTbl: "remote_agents", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddUnique{Name: "unique_remote_account", Cols: []string{"remote_agent_id", "login"}},
	); err != nil {
		return fmt.Errorf("failed to alter the remote_accounts table: %w", err)
	}

	return nil
}

func ver0_7_0RevampRemoteAccountsTableDown(db Actions) error {
	if err := db.AlterTable("remote_accounts",
		DropConstraint{Name: "remote_accounts_agent_fkey"},
		DropConstraint{Name: "unique_remote_account"},
		AlterColumn{Name: "id", Type: UnsignedBigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "remote_agent_id", Type: UnsignedBigInt{}, NotNull: true},
		AlterColumn{Name: "login", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "password", Type: Text{}},
	); err != nil {
		return fmt.Errorf("failed to alter the remote_accounts table: %w", err)
	}

	if err := db.Exec(`UPDATE remote_accounts SET password=NULL WHERE password=''`); err != nil {
		return fmt.Errorf("failed to update the remote_accounts passwords: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_remote_accounts_rem_ac"), Unique: true,
		On: "remote_accounts", Cols: []string{"remote_agent_id", "login"},
	}); err != nil {
		return fmt.Errorf("failed to restore the old remote_account index: %w", err)
	}

	return nil
}

func ver0_7_0RevampRulesTableUp(db Actions) error {
	if err := db.DropIndex(quote(db, "UQE_rules_dir"), "rules"); err != nil {
		return fmt.Errorf("failed to drop the old rule name index: %w", err)
	}

	if err := db.DropIndex(quote(db, "UQE_rules_path"), "rules"); err != nil {
		return fmt.Errorf("failed to drop the old rule path index: %w", err)
	}

	if err := db.AlterTable("rules",
		RenameColumn{OldName: "send", NewName: "is_send"},
		AlterColumn{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "name", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "comment", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "path", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "local_dir", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "remote_dir", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "tmp_local_receive_dir", Type: Text{}, NotNull: true, Default: ""},
		AddUnique{Name: "unique_rule_name", Cols: []string{"is_send", "name"}},
		AddUnique{Name: "unique_rule_path", Cols: []string{"is_send", "path"}},
	); err != nil {
		return fmt.Errorf("failed to alter the rules table: %w", err)
	}

	return nil
}

func ver0_7_0RevampRulesTableDown(db Actions) error {
	if err := db.AlterTable("rules",
		DropConstraint{Name: "unique_rule_name"},
		DropConstraint{Name: "unique_rule_path"},
		RenameColumn{OldName: "is_send", NewName: "send"},
		AlterColumn{Name: "id", Type: UnsignedBigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "name", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "comment", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "path", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "local_dir", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "remote_dir", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "tmp_local_receive_dir", Type: Text{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to create the new remote_accounts table: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_rules_dir"), Unique: true,
		On: "rules", Cols: []string{"send", "name"},
	}); err != nil {
		return fmt.Errorf("failed to restore the old rule name index: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_rules_path"), Unique: true,
		On: "rules", Cols: []string{"send", "path"},
	}); err != nil {
		return fmt.Errorf("failed to restore the old rule path index: %w", err)
	}

	return nil
}

func ver0_7_0RevampTasksTableUp(db Actions) error {
	if err := db.AlterTable("tasks",
		RenameColumn{OldName: "args", NewName: "blob_args"},
		AlterColumn{Name: "rule_id", Type: BigInt{}, NotNull: true},
		AlterColumn{Name: "chain", Type: Varchar(10), NotNull: true},
		AlterColumn{Name: "rank", Type: TinyInt{}, NotNull: true},
		AlterColumn{Name: "type", Type: Varchar(50), NotNull: true},
		AddColumn{Name: "text_args", Type: Text{}, NotNull: true, Default: "{}"},
		AddForeignKey{
			Name: "tasks_rule_id_fkey", Cols: []string{"rule_id"},
			RefTbl: "rules", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddCheck{Name: "task_rank_check", Expr: "rank >= 0"},
		AddCheck{Name: "task_chain_check", Expr: "chain = 'PRE' OR chain = 'POST' OR chain = 'ERROR'"},
		AddUnique{Name: "unique_task_nb", Cols: []string{"rule_id", "chain", "rank"}},
	); err != nil {
		return fmt.Errorf("failed to alter the tasks table: %w", err)
	}

	argsQuery := utils.If(db.GetDialect() == PostgreSQL,
		"UPDATE tasks SET text_args=ENCODE(blob_args, 'escape')",
		"UPDATE tasks SET text_args=blob_args")
	if err := db.Exec(argsQuery); err != nil {
		return fmt.Errorf("failed to update the task arguments: %w", err)
	}

	if err := db.AlterTable("tasks",
		DropColumn{Name: "blob_args"},
		RenameColumn{OldName: "text_args", NewName: "args"},
	); err != nil {
		return fmt.Errorf("failed to alter the tasks table: %w", err)
	}

	return nil
}

func ver0_7_0RevampTasksTableDown(db Actions) error {
	if err := db.AlterTable("tasks",
		DropConstraint{Name: "tasks_rule_id_fkey"},
		DropConstraint{Name: "task_rank_check"},
		DropConstraint{Name: "task_chain_check"},
		DropConstraint{Name: "unique_task_nb"},
		RenameColumn{OldName: "args", NewName: "text_args"},
		AlterColumn{Name: "rule_id", Type: UnsignedBigInt{}, NotNull: true},
		AlterColumn{Name: "chain", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "rank", Type: Integer{}, NotNull: true},
		AlterColumn{Name: "type", Type: Varchar(255), NotNull: true},
		AddColumn{Name: "blob_args", Type: Blob{} /*NotNull: true*/},
	); err != nil {
		return fmt.Errorf("failed to alter the tasks table: %w", err)
	}

	argsQuery := utils.If(db.GetDialect() == PostgreSQL,
		"UPDATE tasks SET blob_args=DECODE(text_args, 'escape')",
		"UPDATE tasks SET blob_args=text_args")
	if err := db.Exec(argsQuery); err != nil {
		return fmt.Errorf("failed to update the task arguments: %w", err)
	}

	if err := db.AlterTable("tasks",
		DropColumn{Name: "text_args"},
		AlterColumn{Name: "blob_args", NewName: "args", Type: Blob{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to alter the tasks table: %w", err)
	}

	return nil
}

func ver0_7_0RevampHistoryTableUp(db Actions) error {
	if db.GetDialect() == PostgreSQL { // set the db time zone to UTC
		if err := db.Exec("SET TimeZone = 'UTC'"); err != nil {
			return fmt.Errorf("failed to set the PostgreSQL time zone: %w", err)
		}
	} else if db.GetDialect() == MySQL { // convert timestamps from the ISO-8601 to the SQL format
		if err := db.Exec(`UPDATE transfer_history SET
			start = REPLACE(REPLACE(start, 'T', ' '), 'Z', ''),
			stop = REPLACE(REPLACE(stop, 'T', ' '), 'Z', '')`); err != nil {
			return fmt.Errorf("failed to format the MySQL timestamps: %w", err)
		}
	}

	if err := db.AlterTable("transfer_history",
		AlterColumn{Name: "id", Type: BigInt{}, NotNull: true},
		AlterColumn{Name: "owner", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "remote_transfer_id", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "rule", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "account", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "agent", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "protocol", Type: Varchar(50), NotNull: true},
		AlterColumn{Name: "local_path", Type: Text{}, NotNull: true},
		AlterColumn{Name: "remote_path", Type: Text{}, NotNull: true},
		AlterColumn{Name: "start", Type: DateTime{}, NotNull: true},
		AlterColumn{Name: "stop", Type: DateTime{}},
		AlterColumn{Name: "progression", NewName: "progress", Type: BigInt{}, NotNull: true, Default: 0},
		AlterColumn{Name: "task_number", Type: TinyInt{}, NotNull: true, Default: 0},
		AlterColumn{Name: "error_code", Type: Varchar(50), NotNull: true, Default: "TeOk"},
		AlterColumn{Name: "error_details", Type: Text{}, NotNull: true, Default: ""},
		AddUnique{Name: "unique_history", Cols: []string{
			"remote_transfer_id",
			"is_server", "account", "agent",
		}},
	); err != nil {
		return fmt.Errorf("failed to alter the transfer_history table: %w", err)
	}

	return nil
}

func ver0_7_0RevampHistoryTableDown(db Actions) error {
	if db.GetDialect() == PostgreSQL { // set the db time zone to UTC
		if err := db.Exec("SET TimeZone = 'UTC'"); err != nil {
			return fmt.Errorf("failed to set the PostgreSQL time zone: %w", err)
		}
	}

	if err := db.AlterTable("transfer_history",
		DropConstraint{Name: "unique_history"},
		AlterColumn{Name: "id", Type: UnsignedBigInt{}, NotNull: true},
		AlterColumn{Name: "owner", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "remote_transfer_id", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "rule", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "account", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "agent", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "protocol", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "local_path", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "remote_path", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "start", Type: DateTimeOffset{}, NotNull: true},
		AlterColumn{Name: "stop", Type: DateTimeOffset{}},
		AlterColumn{Name: "progress", NewName: "progression", Type: UnsignedBigInt{}, NotNull: true},
		AlterColumn{Name: "task_number", Type: UnsignedBigInt{}, NotNull: true},
		AlterColumn{Name: "error_code", Type: Varchar(50), NotNull: true},
		AlterColumn{Name: "error_details", Type: Varchar(255), NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to alter the transfer_history table: %w", err)
	}

	if db.GetDialect() == MySQL { // reconvert timestamps to the ISO-8601 from the SQL format
		if err := db.Exec(`UPDATE transfer_history SET
			start = CONCAT(REPLACE(start, ' ', 'T'), 'Z'),
			stop = CONCAT(REPLACE(stop, ' ', 'T'), 'Z')`); err != nil {
			return fmt.Errorf("failed to format the MySQL timestamps: %w", err)
		}
	}

	return nil
}

func ver0_7_0RevampTransfersTableUp(db Actions) error {
	if db.GetDialect() == PostgreSQL { // set the db time zone to UTC
		if err := db.Exec("SET TimeZone = 'UTC'"); err != nil {
			return fmt.Errorf("failed to set the PostgreSQL time zone: %w", err)
		}
	} else if db.GetDialect() == MySQL { // convert timestamps from the ISO-8601 to the SQL format
		if err := db.Exec(`UPDATE transfers SET
			start = REPLACE(REPLACE(start, 'T', ' '), 'Z', '')`); err != nil {
			return fmt.Errorf("failed to format the MySQL timestamps: %w", err)
		}
	}

	if err := db.AlterTable("transfers",
		DropColumn{Name: "agent_id"},
		AlterColumn{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "owner", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "remote_transfer_id", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "rule_id", Type: BigInt{}, NotNull: true},
		AlterColumn{Name: "local_path", Type: Text{}, NotNull: true},
		AlterColumn{Name: "remote_path", Type: Text{}, NotNull: true},
		AlterColumn{Name: "start", Type: DateTime{}, NotNull: true, Default: CurrentTimestamp{}},
		AlterColumn{Name: "status", Type: Varchar(50), NotNull: true, Default: "PLANNED"},
		AlterColumn{Name: "step", Type: Varchar(50), NotNull: true, Default: "StepNone"},
		AlterColumn{Name: "progression", NewName: "progress", Type: BigInt{}, NotNull: true, Default: 0},
		AlterColumn{Name: "task_number", Type: TinyInt{}, NotNull: true, Default: 0},
		AlterColumn{Name: "error_code", Type: Varchar(50), NotNull: true, Default: "TeOk"},
		AlterColumn{Name: "error_details", Type: Text{}, NotNull: true, Default: ""},
		AddColumn{Name: "local_account_id", Type: BigInt{}},
		AddColumn{Name: "remote_account_id", Type: BigInt{}},
		AddUnique{Name: "unique_transfer_local", Cols: []string{"remote_transfer_id", "local_account_id"}},
		AddUnique{Name: "unique_transfer_remote", Cols: []string{"remote_transfer_id", "remote_account_id"}},
		AddForeignKey{
			Name: "transfers_rule_id_fkey", Cols: []string{"rule_id"},
			RefTbl: "rules", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Restrict,
		},
		AddForeignKey{
			Name: "transfers_local_account_id_fkey", Cols: []string{"local_account_id"},
			RefTbl: "local_accounts", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Restrict,
		},
		AddForeignKey{
			Name: "transfers_remote_account_id_fkey", Cols: []string{"remote_account_id"},
			RefTbl: "remote_accounts", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Restrict,
		},
	); err != nil {
		return fmt.Errorf("failed to alter the transfers table: %w", err)
	}

	if err := db.Exec(`UPDATE transfers SET local_account_id=account_id WHERE is_server`); err != nil {
		return fmt.Errorf("failed to update the local account ids: %w", err)
	}

	if err := db.Exec(`UPDATE transfers SET remote_account_id=account_id WHERE NOT is_server`); err != nil {
		return fmt.Errorf("failed to update the local account ids: %w", err)
	}

	if err := db.AlterTable("transfers",
		DropColumn{Name: "account_id"},
		DropColumn{Name: "is_server"},
		AddCheck{
			Name: "transfer_check_requester",
			Expr: checkOnlyOneNotNull("local_account_id", "remote_account_id"),
		},
	); err != nil {
		return fmt.Errorf("failed to alter the transfers table: %w", err)
	}

	return nil
}

func ver0_7_0RevampTransfersTableDown(db Actions) error {
	if db.GetDialect() == PostgreSQL { // set the db time zone to UTC
		if err := db.Exec("SET TimeZone = 'UTC'"); err != nil {
			return fmt.Errorf("failed to set the PostgreSQL time zone: %w", err)
		}
	}

	if err := db.AlterTable("transfers",
		DropConstraint{Name: "unique_transfer_local"},
		DropConstraint{Name: "unique_transfer_remote"},
		DropConstraint{Name: "transfer_check_requester"},
		DropConstraint{Name: "transfers_rule_id_fkey"},
		DropConstraint{Name: "transfers_local_account_id_fkey"},
		DropConstraint{Name: "transfers_remote_account_id_fkey"},
		AlterColumn{Name: "id", Type: UnsignedBigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "owner", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "remote_transfer_id", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "rule_id", Type: UnsignedBigInt{}, NotNull: true},
		AlterColumn{Name: "local_path", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "remote_path", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "start", Type: DateTimeOffset{}, NotNull: true},
		AlterColumn{Name: "status", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "step", Type: Varchar(50), NotNull: true},
		AlterColumn{Name: "progress", NewName: "progression", Type: UnsignedBigInt{}, NotNull: true},
		AlterColumn{Name: "task_number", Type: UnsignedBigInt{}, NotNull: true},
		AlterColumn{Name: "error_code", Type: Varchar(50), NotNull: true},
		AlterColumn{Name: "error_details", Type: Varchar(255), NotNull: true},
		AddColumn{Name: "is_server", Type: Boolean{} /*NotNull: true*/},
		AddColumn{Name: "agent_id", Type: UnsignedBigInt{} /*NotNull: true*/},
		AddColumn{Name: "account_id", Type: UnsignedBigInt{} /*NotNull: true*/},
	); err != nil {
		return fmt.Errorf("failed to alter the transfers table: %w", err)
	}

	if db.GetDialect() == MySQL { // reconvert timestamps to the ISO-8601 from the SQL format
		if err := db.Exec(`UPDATE transfers SET
			start = CONCAT(REPLACE(start, ' ', 'T'), 'Z')`); err != nil {
			return fmt.Errorf("failed to format the MySQL timestamps: %w", err)
		}
	}

	if err := db.Exec(`UPDATE transfers SET is_server=true, account_id=local_account_id, 
			    agent_id=(SELECT local_agent_id FROM local_accounts WHERE id=local_account_id)
            WHERE local_account_id IS NOT NULL`); err != nil {
		return fmt.Errorf("failed to update the local account ids: %w", err)
	}

	if err := db.Exec(`UPDATE transfers SET is_server=false, account_id=remote_account_id,
            	agent_id=(SELECT remote_agent_id FROM remote_accounts WHERE id=remote_account_id)
            WHERE remote_account_id IS NOT NULL`); err != nil {
		return fmt.Errorf("failed to update the remote account ids: %w", err)
	}

	if err := db.AlterTable("transfers",
		DropColumn{Name: "local_account_id"},
		DropColumn{Name: "remote_account_id"},
		AlterColumn{Name: "is_server", Type: Boolean{}, NotNull: true},
		AlterColumn{Name: "agent_id", Type: BigInt{}, NotNull: true},
		AlterColumn{Name: "account_id", Type: BigInt{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to alter the transfers table: %w", err)
	}

	return nil
}

func ver0_7_0RevampTransferInfoTableUp(db Actions) error {
	if err := db.DropIndex(quote(db, "UQE_transfer_info_infoName"), "transfer_info"); err != nil {
		return fmt.Errorf("failed to drop the old transfer_info index: %w", err)
	}

	if err := db.Exec(`DELETE FROM transfer_info WHERE is_history=true AND
		transfer_id NOT IN (SELECT id FROM transfer_history)`); err != nil {
		return fmt.Errorf("failed to delete the orphaned transfer info entries: %w", err)
	}

	if err := db.AlterTable("transfer_info",
		AddColumn{Name: "history_id", Type: BigInt{}},
		AlterColumn{Name: "transfer_id", Type: BigInt{}},
		AlterColumn{Name: "name", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "value", Type: Text{}, NotNull: true, Default: "null"},
		AddUnique{Name: "unique_transfer_info", Cols: []string{"transfer_id", "name"}},
		AddUnique{Name: "unique_history_info", Cols: []string{"history_id", "name"}},
	); err != nil {
		return fmt.Errorf("failed to alter the transfer_info table: %w", err)
	}

	if err := db.Exec(`UPDATE transfer_info SET history_id=transfer_id, transfer_id=null
		WHERE is_history=true`); err != nil {
		return fmt.Errorf("failed to update the transfer info table: %w", err)
	}

	if err := db.AlterTable("transfer_info",
		DropColumn{Name: "is_history"},
		AddForeignKey{
			Name: "info_transfer_fkey", Cols: []string{"transfer_id"},
			RefTbl: "transfers", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddForeignKey{
			Name: "info_history_fkey", Cols: []string{"history_id"},
			RefTbl: "transfer_history", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddCheck{
			Name: "transfer_info_check_owner",
			Expr: checkOnlyOneNotNull("transfer_id", "history_id"),
		},
	); err != nil {
		return fmt.Errorf("failed to alter the transfer_info table: %w", err)
	}

	return nil
}

func ver0_7_0RevampTransferInfoTableDown(db Actions) error {
	if err := db.AlterTable("transfer_info",
		DropConstraint{Name: "info_transfer_fkey"},
		DropConstraint{Name: "info_history_fkey"},
		DropConstraint{Name: "unique_transfer_info"},
		DropConstraint{Name: "unique_history_info"},
		DropConstraint{Name: "transfer_info_check_owner"},
		AddColumn{Name: "is_history", Type: Boolean{}, NotNull: true, Default: true},
		AlterColumn{Name: "name", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "value", Type: Varchar(255), NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to alter the transfer_info table: %w", err)
	}

	if err := db.Exec(`UPDATE transfer_info	SET transfer_id=history_id
		WHERE history_id IS NOT NULL`); err != nil {
		return fmt.Errorf("failed to update the transfer info table: %w", err)
	}

	if err := db.Exec(`UPDATE transfer_info	SET is_history=false
		WHERE history_id IS NULL`); err != nil {
		return fmt.Errorf("failed to update the transfer info table: %w", err)
	}

	if err := db.AlterTable("transfer_info",
		DropColumn{Name: "history_id"},
		AlterColumn{Name: "transfer_id", Type: UnsignedBigInt{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to alter the transfer_info table: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_transfer_info_infoName"), Unique: true,
		On: "transfer_info", Cols: []string{"transfer_id", "name"},
	}); err != nil {
		return fmt.Errorf("failed to restore the old transfer_info index: %w", err)
	}

	return nil
}

func ver0_7_0RevampCryptoTableUp(db Actions) error {
	if err := db.DropIndex(quote(db, "UQE_crypto_credentials_cert"),
		"crypto_credentials"); err != nil {
		return fmt.Errorf("failed to drop the old crypto_credentials index: %w", err)
	}

	if err := db.Exec(`UPDATE crypto_credentials SET private_key='' WHERE private_key IS NULL`); err != nil {
		return fmt.Errorf("failed to update the crypto_credentials private keys: %w", err)
	}

	if err := db.AlterTable("crypto_credentials",
		AlterColumn{Name: "id", Type: BigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "name", Type: Varchar(100), NotNull: true},
		AlterColumn{Name: "private_key", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "certificate", Type: Text{}, NotNull: true, Default: ""},
		AlterColumn{Name: "ssh_public_key", Type: Text{}, NotNull: true, Default: ""},
		AddColumn{Name: "local_agent_id", Type: BigInt{}},
		AddColumn{Name: "remote_agent_id", Type: BigInt{}},
		AddColumn{Name: "local_account_id", Type: BigInt{}},
		AddColumn{Name: "remote_account_id", Type: BigInt{}},
		AddUnique{Name: "unique_crypto_loc_agent", Cols: []string{"local_agent_id", "name"}},
		AddUnique{Name: "unique_crypto_rem_agent", Cols: []string{"remote_agent_id", "name"}},
		AddUnique{Name: "unique_crypto_loc_account", Cols: []string{"local_account_id", "name"}},
		AddUnique{Name: "unique_crypto_rem_account", Cols: []string{"remote_account_id", "name"}},
		AddForeignKey{
			Name: "crypto_local_agent_fkey", Cols: []string{"local_agent_id"},
			RefTbl: "local_agents", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddForeignKey{
			Name: "crypto_remote_agent_fkey", Cols: []string{"remote_agent_id"},
			RefTbl: "remote_agents", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddForeignKey{
			Name: "crypto_local_account_fkey", Cols: []string{"local_account_id"},
			RefTbl: "local_accounts", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddForeignKey{
			Name: "crypto_remote_account_fkey", Cols: []string{"remote_account_id"},
			RefTbl: "remote_accounts", RefCols: []string{"id"},
			OnUpdate: Restrict, OnDelete: Cascade,
		},
	); err != nil {
		return fmt.Errorf("failed to alter the crypto_credentials table: %w", err)
	}

	if err := db.Exec(`UPDATE crypto_credentials SET
		local_agent_id=(CASE WHEN owner_type='local_agents' THEN owner_id END),
		remote_agent_id=(CASE WHEN owner_type='remote_agents' THEN owner_id END),
		local_account_id=(CASE WHEN owner_type='local_accounts' THEN owner_id END),
		remote_account_id=(CASE WHEN owner_type='remote_accounts' THEN owner_id END)`); err != nil {
		return fmt.Errorf("failed to update the crypto_credentials table: %w", err)
	}

	if err := db.AlterTable("crypto_credentials",
		DropColumn{Name: "owner_type"},
		DropColumn{Name: "owner_id"},
		AddCheck{
			Name: "crypto_check_owner",
			Expr: checkOnlyOneNotNull("local_agent_id", "remote_agent_id",
				"local_account_id", "remote_account_id"),
		},
	); err != nil {
		return fmt.Errorf("failed to alter the crypto_credentials table: %w", err)
	}

	return nil
}

func ver0_7_0RevampCryptoTableDown(db Actions) error {
	if err := db.AlterTable("crypto_credentials",
		DropConstraint{Name: "crypto_local_agent_fkey"},
		DropConstraint{Name: "crypto_remote_agent_fkey"},
		DropConstraint{Name: "crypto_local_account_fkey"},
		DropConstraint{Name: "crypto_remote_account_fkey"},
		DropConstraint{Name: "unique_crypto_loc_agent"},
		DropConstraint{Name: "unique_crypto_rem_agent"},
		DropConstraint{Name: "unique_crypto_loc_account"},
		DropConstraint{Name: "unique_crypto_rem_account"},
		DropConstraint{Name: "crypto_check_owner"},
		AlterColumn{Name: "id", Type: UnsignedBigInt{}, NotNull: true, Default: AutoIncr{}},
		AlterColumn{Name: "name", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "private_key", Type: Text{}},
		AlterColumn{Name: "certificate", Type: Text{}},
		AlterColumn{Name: "ssh_public_key", Type: Text{}},
		AddColumn{Name: "owner_type", Type: Varchar(255) /*NotNull: true*/},
		AddColumn{Name: "owner_id", Type: UnsignedBigInt{} /*NotNull: true*/},
	); err != nil {
		return fmt.Errorf("failed to alter the crypto_credentials table: %w", err)
	}

	if err := db.Exec(`UPDATE crypto_credentials SET private_key=NULL WHERE private_key=''`); err != nil {
		return fmt.Errorf("failed to update the crypto_credentials private keys: %w", err)
	}

	if err := db.Exec(`UPDATE crypto_credentials SET
		owner_type=(CASE WHEN local_agent_id IS NOT NULL THEN 'local_agents'
						 WHEN remote_agent_id IS NOT NULL THEN 'remote_agents'
						 WHEN local_account_id IS NOT NULL THEN 'local_accounts'
						 WHEN remote_account_id IS NOT NULL THEN 'remote_accounts' END),
		owner_id=COALESCE(local_agent_id, remote_agent_id, local_account_id, remote_account_id)`,
	); err != nil {
		return fmt.Errorf("failed to copy the content of the crypto_credentials table: %w", err)
	}

	if err := db.AlterTable("crypto_credentials",
		DropColumn{Name: "local_agent_id"},
		DropColumn{Name: "remote_agent_id"},
		DropColumn{Name: "local_account_id"},
		DropColumn{Name: "remote_account_id"},
		AlterColumn{Name: "owner_type", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "owner_id", Type: UnsignedBigInt{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to alter the crypto_credentials table: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_crypto_credentials_cert"), Unique: true,
		On: "crypto_credentials", Cols: []string{"name", "owner_type", "owner_id"},
	}); err != nil {
		return fmt.Errorf("failed to restore the old crypto_credentials index: %w", err)
	}

	return nil
}

func ver0_7_0RevampRuleAccessTableUp(db Actions) error {
	if err := db.DropIndex(quote(db, "UQE_rule_access_perm"), "rule_access"); err != nil {
		return fmt.Errorf("failed to drop the old rule_access index: %w", err)
	}

	if err := db.AlterTable("rule_access",
		AlterColumn{Name: "rule_id", Type: BigInt{}, NotNull: true},
		AddColumn{Name: "local_agent_id", Type: BigInt{}},
		AddColumn{Name: "remote_agent_id", Type: BigInt{}},
		AddColumn{Name: "local_account_id", Type: BigInt{}},
		AddColumn{Name: "remote_account_id", Type: BigInt{}},
		AddUnique{Name: "unique_access_loc_agent", Cols: []string{"rule_id", "local_agent_id"}},
		AddUnique{Name: "unique_access_rem_agent", Cols: []string{"rule_id", "remote_agent_id"}},
		AddUnique{Name: "unique_access_loc_account", Cols: []string{"rule_id", "local_account_id"}},
		AddUnique{Name: "unique_access_rem_account", Cols: []string{"rule_id", "remote_account_id"}},
		AddForeignKey{
			Name: "access_rule_id_fkey", Cols: []string{"rule_id"},
			RefTbl: "rules", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddForeignKey{
			Name: "access_local_agent_id_fkey", Cols: []string{"local_agent_id"},
			RefTbl: "local_agents", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddForeignKey{
			Name: "access_remote_agent_id_fkey", Cols: []string{"remote_agent_id"},
			RefTbl: "remote_agents", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddForeignKey{
			Name: "access_local_account_id_fkey", Cols: []string{"local_account_id"},
			RefTbl: "local_accounts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		},
		AddForeignKey{
			Name: "access_remote_account_id_fkey", Cols: []string{"remote_account_id"},
			RefTbl: "remote_accounts", RefCols: []string{"id"}, OnUpdate: Restrict, OnDelete: Cascade,
		},
	); err != nil {
		return fmt.Errorf("failed to alter the rule_access table: %w", err)
	}

	if err := db.Exec(`UPDATE rule_access SET
		local_agent_id=(CASE WHEN object_type='local_agents' THEN object_id END),
		remote_agent_id=(CASE WHEN object_type='remote_agents' THEN object_id END),
		local_account_id=(CASE WHEN object_type='local_accounts' THEN object_id END),
		remote_account_id=(CASE WHEN object_type='remote_accounts' THEN object_id END)`,
	); err != nil {
		return fmt.Errorf("failed to copy the content of the rule_access table: %w", err)
	}

	if err := db.AlterTable("rule_access",
		DropColumn{Name: "object_id"},
		DropColumn{Name: "object_type"},
		AddCheck{
			Name: "rule_access_target_check",
			Expr: checkOnlyOneNotNull("local_agent_id", "remote_agent_id",
				"local_account_id", "remote_account_id"),
		},
	); err != nil {
		return fmt.Errorf("failed to alter the rule_access table: %w", err)
	}

	return nil
}

func ver0_7_0RevampRuleAccessTableDown(db Actions) error {
	if err := db.AlterTable("rule_access",
		DropConstraint{Name: "access_rule_id_fkey"},
		DropConstraint{Name: "access_local_agent_id_fkey"},
		DropConstraint{Name: "access_remote_agent_id_fkey"},
		DropConstraint{Name: "access_local_account_id_fkey"},
		DropConstraint{Name: "access_remote_account_id_fkey"},
		DropConstraint{Name: "unique_access_loc_agent"},
		DropConstraint{Name: "unique_access_rem_agent"},
		DropConstraint{Name: "unique_access_loc_account"},
		DropConstraint{Name: "unique_access_rem_account"},
		DropConstraint{Name: "rule_access_target_check"},
		AlterColumn{Name: "rule_id", Type: UnsignedBigInt{}, NotNull: true},
		AddColumn{Name: "object_type", Type: Varchar(255) /*NotNull: true*/},
		AddColumn{Name: "object_id", Type: UnsignedBigInt{} /*NotNull: true*/},
	); err != nil {
		return fmt.Errorf("failed to create the new rule_access table: %w", err)
	}

	if err := db.Exec(`UPDATE rule_access SET
		object_type=(CASE WHEN local_agent_id IS NOT NULL THEN 'local_agents'
						  WHEN remote_agent_id IS NOT NULL THEN 'remote_agents'
						  WHEN local_account_id IS NOT NULL THEN 'local_accounts'
						  WHEN remote_account_id IS NOT NULL THEN 'remote_accounts' END),
		object_id=(COALESCE(local_agent_id, remote_agent_id, local_account_id, remote_account_id))`,
	); err != nil {
		return fmt.Errorf("failed to copy the content of the rule_access table: %w", err)
	}

	if err := db.AlterTable("rule_access",
		DropColumn{Name: "local_agent_id"},
		DropColumn{Name: "remote_agent_id"},
		DropColumn{Name: "local_account_id"},
		DropColumn{Name: "remote_account_id"},
		AlterColumn{Name: "object_type", Type: Varchar(255), NotNull: true},
		AlterColumn{Name: "object_id", Type: UnsignedBigInt{}, NotNull: true},
	); err != nil {
		return fmt.Errorf("failed to create the new rule_access table: %w", err)
	}

	if err := db.CreateIndex(&Index{
		Name: quote(db, "UQE_rule_access_perm"), Unique: true,
		On: "rule_access", Cols: []string{"rule_id", "object_type", "object_id"},
	}); err != nil {
		return fmt.Errorf("failed to drop the old rule_access index: %w", err)
	}

	return nil
}

func ver0_7_0AddLocalAgentsAddressUniqueUp(db Actions) error {
	if err := db.AlterTable("local_agents",
		AddUnique{Name: "unique_server_addr", Cols: []string{"owner", "address"}},
	); err != nil {
		return fmt.Errorf("failed to add the local agent 'address' unique constraint: %w", err)
	}

	return nil
}

func ver0_7_0AddLocalAgentsAddressUniqueDown(db Actions) error {
	if err := db.AlterTable("local_agents",
		DropConstraint{Name: "unique_server_addr"},
	); err != nil {
		return fmt.Errorf("failed to drop the local agent 'address' unique index: %w", err)
	}

	return nil
}

func ver0_7_0AddNormalizedTransfersViewUp(db Actions) error {
	transStop := utils.If(db.GetDialect() == PostgreSQL,
		"null::timestamp", "null")

	if err := db.CreateView(&View{
		Name: "normalized_transfers",
		As: `WITH transfers_as_history(id, owner, remote_transfer_id, is_server,
				is_send, rule, account, agent, protocol, local_path, remote_path, 
				filesize, start, stop, status, step, progress, task_number, error_code, 
				error_details, is_transfer) AS (
					SELECT t.id, t.owner, t.remote_transfer_id, 
						t.local_account_id IS NOT NULL, r.is_send, r.name,
						(CASE WHEN t.local_account_id IS NULL THEN ra.login ELSE la.login END),
						(CASE WHEN t.local_account_id IS NULL THEN p.name ELSE s.name END),
						(CASE WHEN t.local_account_id IS NULL THEN p.protocol ELSE s.protocol END),
						t.local_path, t.remote_path, t.filesize, t.start, ` + transStop + `, t.status,
						t.step, t.progress, t.task_number, t.error_code, t.error_details, true
					FROM transfers AS t
					LEFT JOIN rules AS r ON t.rule_id = r.id
					LEFT JOIN local_accounts  AS la ON  t.local_account_id = la.id
					LEFT JOIN remote_accounts AS ra ON t.remote_account_id = ra.id
					LEFT JOIN local_agents    AS s ON la.local_agent_id = s.id 
					LEFT JOIN remote_agents   AS p ON ra.remote_agent_id = p.id
				)
			SELECT id, owner, remote_transfer_id, is_server, is_send, rule, account,
				agent, protocol, local_path, remote_path, filesize, start, stop, 
				status,	step, progress, task_number, error_code, error_details,
				false AS is_transfer
			FROM transfer_history UNION
			SELECT * FROM transfers_as_history`,
	}); err != nil {
		return fmt.Errorf("failed to create the normalized transfer view: %w", err)
	}

	return nil
}

func ver0_7_0AddNormalizedTransfersViewDown(db Actions) error {
	if err := db.DropView("normalized_transfers"); err != nil {
		return fmt.Errorf("failed to drop the normalized transfer view: %w", err)
	}

	return nil
}
