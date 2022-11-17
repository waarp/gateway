package migrations

import (
	"encoding/binary"
	"fmt"

	"code.waarp.fr/lib/migration"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

type ver0_7_0AddLocalAgentEnabledColumn struct{}

func (ver0_7_0AddLocalAgentEnabledColumn) Up(db migration.Actions) error {
	if err := db.AddColumn("local_agents", "enabled", migration.Boolean,
		migration.NotNull, migration.Default(true)); err != nil {
		return fmt.Errorf("failed to add the local agent 'enabled' column: %w", err)
	}

	return nil
}

func (ver0_7_0AddLocalAgentEnabledColumn) Down(db migration.Actions) error {
	if err := db.DropColumn("local_agents", "enabled"); err != nil {
		return fmt.Errorf("failed to drop the local agent 'enabled' column: %w", err)
	}

	return nil
}

type ver0_7_0RevampUsersTable struct{}

func (ver0_7_0RevampUsersTable) getUpUsersList(db migration.Actions,
) ([]struct{ id, perms int64 }, error) {
	rows, err := db.Query(`SELECT id, permissions FROM users`)
	if err != nil || rows.Err() != nil {
		return nil, fmt.Errorf("failed to retrieve the user table content: %w", err)
	}

	defer rows.Close() //nolint:errcheck //error is irrelevant here

	var users []struct{ id, perms int64 }

	for rows.Next() {
		var (
			id        int64
			permBytes []byte
		)

		if err := rows.Scan(&id, &permBytes); err != nil {
			return nil, fmt.Errorf("failed to parse user values: %w", err)
		}

		perms := int64(binary.LittleEndian.Uint32(permBytes))

		users = append(users, struct{ id, perms int64 }{id: id, perms: perms})
	}

	return users, nil
}

func (v ver0_7_0RevampUsersTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("users_new",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("owner", migration.Varchar(100), migration.NotNull),
		migration.Col("username", migration.Varchar(100), migration.NotNull),
		migration.Col("password_hash", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("permissions", migration.BigInt, migration.NotNull, migration.Default(0)),
		migration.MultiUnique("unique_username", "owner", "username"),
	); err != nil {
		return fmt.Errorf("failed to create the new users table: %w", err)
	}

	if err := db.Exec(`INSERT INTO users_new (id, owner, username, password_hash)
		SELECT id, owner, username, password_hash FROM users`); err != nil {
		return fmt.Errorf("failed to copy the content of the users table: %w", err)
	}

	users, err := v.getUpUsersList(db)
	if err != nil {
		return err
	}

	for i := range users {
		if err := db.Exec(`UPDATE users_new SET permissions=? WHERE id=?`,
			users[i].perms, users[i].id); err != nil {
			return fmt.Errorf("failed to update the user permissions: %w", err)
		}
	}

	if err := db.DropTable("users"); err != nil {
		return fmt.Errorf("failed to drop the users table: %w", err)
	}

	if err := db.RenameTable("users_new", "users"); err != nil {
		return fmt.Errorf("failed to rename the users table: %w", err)
	}

	return nil
}

func (ver0_7_0RevampUsersTable) getDownUsersList(db migration.Actions) (users []struct {
	id                int64
	owner, name, hash string
	permMask          []byte
}, _ error,
) {
	rows, err := db.Query(`SELECT id, owner, username, password_hash, permissions FROM users`)
	if err != nil || rows.Err() != nil {
		return nil, fmt.Errorf("failed to retrieve the user table content: %w", err)
	}

	defer rows.Close() //nolint:errcheck //error is irrelevant here

	for rows.Next() {
		var (
			id, perms         int64
			owner, name, hash string
		)

		if err := rows.Scan(&id, &owner, &name, &hash, &perms); err != nil {
			return nil, fmt.Errorf("failed to parse user values: %w", err)
		}

		const permSize = 4
		permMask := make([]byte, permSize)
		binary.LittleEndian.PutUint32(permMask, uint32(perms))

		users = append(users, struct {
			id                int64
			owner, name, hash string
			permMask          []byte
		}{
			id:       id,
			owner:    owner,
			name:     name,
			hash:     hash,
			permMask: permMask,
		})
	}

	return users, nil
}

func (v ver0_7_0RevampUsersTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("users_old",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("owner", migration.Varchar(255), migration.NotNull),
		migration.Col("username", migration.Varchar(255), migration.NotNull),
		migration.Col("password_hash", migration.Text, migration.NotNull),
		migration.Col("permissions", migration.Binary(4), migration.NotNull),
		migration.MultiUnique("UQE_users_name", "owner", "username"),
	); err != nil {
		return fmt.Errorf("failed to create the new users table: %w", err)
	}

	users, err := v.getDownUsersList(db)
	if err != nil {
		return err
	}

	for i := range users {
		if err := db.Exec(`INSERT INTO users_old (id, owner, username, password_hash,
			permissions) VALUES (?,?,?,?,?)`, users[i].id, users[i].owner,
			users[i].name, users[i].hash, users[i].permMask); err != nil {
			return fmt.Errorf("failed to update the user permissions: %w", err)
		}
	}

	if err := db.DropTable("users"); err != nil {
		return fmt.Errorf("failed to drop the users table: %w", err)
	}

	if err := db.RenameTable("users_old", "users"); err != nil {
		return fmt.Errorf("failed to rename the users table: %w", err)
	}

	return nil
}

type ver0_7_0RevampLocalAgentsTable struct{}

func (ver0_7_0RevampLocalAgentsTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("local_agents_new",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("owner", migration.Varchar(100), migration.NotNull),
		migration.Col("name", migration.Varchar(100), migration.NotNull),
		migration.Col("protocol", migration.Varchar(50), migration.NotNull),
		migration.Col("enabled", migration.Boolean, migration.NotNull, migration.Default(true)),
		migration.Col("address", migration.Varchar(260), migration.NotNull),
		migration.Col("root_dir", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("receive_dir", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("send_dir", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("tmp_receive_dir", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("proto_config", migration.Text, migration.NotNull, migration.Default("{}")),
		migration.MultiUnique("unique_local_agent_name", "owner", "name"),
	); err != nil {
		return fmt.Errorf("failed to create the new local_agents table: %w", err)
	}

	protoConf := "proto_config" //nolint:goconst //adding a constant here would be a bad idea
	if db.GetDialect() == PostgreSQL {
		protoConf = "ENCODE(proto_config, 'escape')"
	}

	if err := db.Exec(`INSERT INTO local_agents_new (id, owner, name, protocol, 
		address, root_dir, receive_dir, send_dir, tmp_receive_dir, proto_config)
		SELECT id, owner, name, protocol, address, root_dir, receive_dir, 
		send_dir, tmp_receive_dir, ` + protoConf + ` FROM local_agents`); err != nil {
		return fmt.Errorf("failed to copy the content of the local_agents table: %w", err)
	}

	if err := db.DropTable("local_agents"); err != nil {
		return fmt.Errorf("failed to drop the local_agents table: %w", err)
	}

	if err := db.RenameTable("local_agents_new", "local_agents"); err != nil {
		return fmt.Errorf("failed to rename the local_agents table: %w", err)
	}

	return nil
}

func (ver0_7_0RevampLocalAgentsTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("local_agents_old",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("owner", migration.Varchar(255), migration.NotNull),
		migration.Col("name", migration.Varchar(255), migration.NotNull),
		migration.Col("protocol", migration.Varchar(255), migration.NotNull),
		migration.Col("root_dir", migration.Varchar(255), migration.NotNull),
		migration.Col("receive_dir", migration.Varchar(255), migration.NotNull),
		migration.Col("send_dir", migration.Varchar(255), migration.NotNull),
		migration.Col("tmp_receive_dir", migration.Varchar(255), migration.NotNull),
		migration.Col("proto_config", migration.Blob, migration.NotNull),
		migration.Col("address", migration.Varchar(255), migration.NotNull),
		migration.MultiUnique("UQE_local_agents_loc_ag", "owner", "name"),
	); err != nil {
		return fmt.Errorf("failed to recreate the old local_agents table: %w", err)
	}

	protoConf := "proto_config" //nolint:goconst //adding a constant here would be a bad idea
	if db.GetDialect() == PostgreSQL {
		protoConf = "DECODE(proto_config, 'escape')"
	}

	if err := db.Exec(`INSERT INTO local_agents_old (id, owner, name, protocol, 
		address, root_dir, receive_dir, send_dir, tmp_receive_dir, proto_config)
		SELECT id, owner, name, protocol, address, root_dir, receive_dir, 
		send_dir, tmp_receive_dir, ` + protoConf + ` FROM local_agents`); err != nil {
		return fmt.Errorf("failed to copy the content of the local_agents table: %w", err)
	}

	if err := db.DropTable("local_agents"); err != nil {
		return fmt.Errorf("failed to drop the local_agents table: %w", err)
	}

	if err := db.RenameTable("local_agents_old", "local_agents"); err != nil {
		return fmt.Errorf("failed to rename the local_agents table: %w", err)
	}

	return nil
}

type ver0_7_0RevampRemoteAgentsTable struct{}

func (ver0_7_0RevampRemoteAgentsTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("remote_agents_new",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("name", migration.Varchar(100), migration.NotNull, migration.Unique),
		migration.Col("protocol", migration.Varchar(50), migration.NotNull),
		migration.Col("address", migration.Varchar(260), migration.NotNull),
		migration.Col("proto_config", migration.Text, migration.NotNull, migration.Default("{}")),
	); err != nil {
		return fmt.Errorf("failed to create the new remote_agents table: %w", err)
	}

	protoConf := "proto_config" //nolint:goconst //adding a constant here would be a bad idea
	if db.GetDialect() == PostgreSQL {
		protoConf = "ENCODE(proto_config, 'escape')"
	}

	if err := db.Exec(`INSERT INTO remote_agents_new (id, name, protocol, address, 
		proto_config) SELECT id, name, protocol, address, ` + protoConf +
		` FROM remote_agents`); err != nil {
		return fmt.Errorf("failed to copy the content of the remote_agents table: %w", err)
	}

	if err := db.DropTable("remote_agents"); err != nil {
		return fmt.Errorf("failed to drop the remote_agents table: %w", err)
	}

	if err := db.RenameTable("remote_agents_new", "remote_agents"); err != nil {
		return fmt.Errorf("failed to rename the remote_agents table: %w", err)
	}

	return nil
}

func (ver0_7_0RevampRemoteAgentsTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("remote_agents_old",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("name", migration.Varchar(255), migration.NotNull, migration.Unique),
		migration.Col("protocol", migration.Varchar(255), migration.NotNull),
		migration.Col("proto_config", migration.Blob, migration.NotNull),
		migration.Col("address", migration.Varchar(255), migration.NotNull),
	); err != nil {
		return fmt.Errorf("failed to recreate the old remote_agents table: %w", err)
	}

	protoConf := "proto_config" //nolint:goconst //adding a constant here would be a bad idea
	if db.GetDialect() == PostgreSQL {
		protoConf = "DECODE(proto_config, 'escape')"
	}

	if err := db.Exec(`INSERT INTO remote_agents_old (id, name, protocol, address, 
		proto_config) SELECT id, name, protocol, address, ` + protoConf +
		` FROM remote_agents`); err != nil {
		return fmt.Errorf("failed to copy the content of the remote_agents table: %w", err)
	}

	if err := db.DropTable("remote_agents"); err != nil {
		return fmt.Errorf("failed to drop the remote_agents table: %w", err)
	}

	if err := db.RenameTable("remote_agents_old", "remote_agents"); err != nil {
		return fmt.Errorf("failed to rename the remote_agents table: %w", err)
	}

	return nil
}

type ver0_7_0RevampLocalAccountsTable struct{}

//nolint:dupl //factorizing would greatly hurt readability
func (ver0_7_0RevampLocalAccountsTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("local_accounts_new",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("local_agent_id", migration.BigInt, migration.NotNull, migration.ForeignKey("local_agents", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("login", migration.Varchar(100), migration.NotNull),
		migration.Col("password_hash", migration.Text, migration.NotNull, migration.Default("")),
		migration.MultiUnique("unique_local_account_login", "local_agent_id", "login"),
	); err != nil {
		return fmt.Errorf("failed to create the new local_accounts table: %w", err)
	}

	if err := db.Exec(`INSERT INTO local_accounts_new (id, local_agent_id, login, password_hash) 
		SELECT id, local_agent_id, login, password_hash FROM local_accounts`); err != nil {
		return fmt.Errorf("failed to copy the content of the local_accounts table: %w", err)
	}

	if err := db.DropTable("local_accounts"); err != nil {
		return fmt.Errorf("failed to drop the local_accounts table: %w", err)
	}

	if err := db.RenameTable("local_accounts_new", "local_accounts"); err != nil {
		return fmt.Errorf("failed to rename the local_accounts table: %w", err)
	}

	return nil
}

//nolint:dupl //factorizing would greatly hurt readability
func (ver0_7_0RevampLocalAccountsTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("local_accounts_old",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("local_agent_id", migration.BigInt, migration.NotNull),
		migration.Col("login", migration.Varchar(255), migration.NotNull),
		migration.Col("password_hash", migration.Text),
		migration.MultiUnique("uqe_local_accounts_new_loc_ac", "local_agent_id", "login"),
	); err != nil {
		return fmt.Errorf("failed to create the new local_accounts table: %w", err)
	}

	if err := db.Exec(`INSERT INTO local_accounts_old (id, local_agent_id, login, password_hash) 
		SELECT id, local_agent_id, login, password_hash FROM local_accounts`); err != nil {
		return fmt.Errorf("failed to copy the content of the local_accounts table: %w", err)
	}

	if err := db.DropTable("local_accounts"); err != nil {
		return fmt.Errorf("failed to drop the local_accounts table: %w", err)
	}

	if err := db.RenameTable("local_accounts_old", "local_accounts"); err != nil {
		return fmt.Errorf("failed to rename the local_accounts table: %w", err)
	}

	return nil
}

type ver0_7_0RevampRemoteAccountsTable struct{}

//nolint:dupl //factorizing would greatly hurt readability
func (ver0_7_0RevampRemoteAccountsTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("remote_accounts_new",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("remote_agent_id", migration.BigInt, migration.NotNull,
			migration.ForeignKey("remote_agents", "id").OnUpdate(migration.Restrict).
				OnDelete(migration.Cascade)),
		migration.Col("login", migration.Varchar(100), migration.NotNull),
		migration.Col("password", migration.Text, migration.NotNull, migration.Default("")),
		migration.MultiUnique("unique_remote_account_login", "remote_agent_id", "login"),
	); err != nil {
		return fmt.Errorf("failed to create the new remote_accounts table: %w", err)
	}

	if err := db.Exec(`INSERT INTO remote_accounts_new (id, remote_agent_id, login, password) 
		SELECT id, remote_agent_id, login, password FROM remote_accounts`); err != nil {
		return fmt.Errorf("failed to copy the content of the remote_accounts table: %w", err)
	}

	if err := db.DropTable("remote_accounts"); err != nil {
		return fmt.Errorf("failed to drop the remote_accounts table: %w", err)
	}

	if err := db.RenameTable("remote_accounts_new", "remote_accounts"); err != nil {
		return fmt.Errorf("failed to rename the remote_accounts table: %w", err)
	}

	return nil
}

//nolint:dupl //factorizing would greatly hurt readability
func (ver0_7_0RevampRemoteAccountsTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("remote_accounts_old",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("remote_agent_id", migration.BigInt, migration.NotNull),
		migration.Col("login", migration.Varchar(255), migration.NotNull),
		migration.Col("password", migration.Text),
		migration.MultiUnique("UQE_remote_accounts_old_rem_ac", "remote_agent_id", "login"),
	); err != nil {
		return fmt.Errorf("failed to create the new remote_accounts table: %w", err)
	}

	if err := db.Exec(`INSERT INTO remote_accounts_old (id, remote_agent_id, login, password) 
		SELECT id, remote_agent_id, login, password FROM remote_accounts`); err != nil {
		return fmt.Errorf("failed to copy the content of the remote_accounts table: %w", err)
	}

	if err := db.DropTable("remote_accounts"); err != nil {
		return fmt.Errorf("failed to drop the remote_accounts table: %w", err)
	}

	if err := db.RenameTable("remote_accounts_old", "remote_accounts"); err != nil {
		return fmt.Errorf("failed to rename the remote_accounts table: %w", err)
	}

	return nil
}

type ver0_7_0RevampRulesTable struct{}

func (ver0_7_0RevampRulesTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("rules_new",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("name", migration.Varchar(100), migration.NotNull),
		migration.Col("is_send", migration.Boolean, migration.NotNull),
		migration.Col("comment", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("path", migration.Varchar(255), migration.NotNull),
		migration.Col("local_dir", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("remote_dir", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("tmp_local_receive_dir", migration.Text, migration.NotNull, migration.Default("")),
		migration.MultiUnique("unique_rule_name", "is_send", "name"),
		migration.MultiUnique("unique_rule_path", "is_send", "path"),
	); err != nil {
		return fmt.Errorf("failed to create the new rules table: %w", err)
	}

	if err := db.Exec(`INSERT INTO rules_new (id, name, is_send, comment, path,
		local_dir, remote_dir, tmp_local_receive_dir) SELECT id, name, send, 
		comment, path, local_dir, remote_dir, tmp_local_receive_dir FROM rules`); err != nil {
		return fmt.Errorf("failed to copy the content of the rules table: %w", err)
	}

	if err := db.DropTable("rules"); err != nil {
		return fmt.Errorf("failed to drop the rules table: %w", err)
	}

	if err := db.RenameTable("rules_new", "rules"); err != nil {
		return fmt.Errorf("failed to rename the rules table: %w", err)
	}

	return nil
}

func (ver0_7_0RevampRulesTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("rules_old",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("name", migration.Varchar(255), migration.NotNull),
		migration.Col("comment", migration.Varchar(255), migration.NotNull),
		migration.Col("send", migration.Boolean, migration.NotNull),
		migration.Col("path", migration.Varchar(255), migration.NotNull),
		migration.Col("local_dir", migration.Varchar(255), migration.NotNull),
		migration.Col("remote_dir", migration.Varchar(255), migration.NotNull),
		migration.Col("tmp_local_receive_dir", migration.Varchar(255), migration.NotNull),
		migration.MultiUnique("UQE_rules_dir", "name", "send"),
		migration.MultiUnique("UQE_rules_path", "path", "send"),
	); err != nil {
		return fmt.Errorf("failed to create the new remote_accounts table: %w", err)
	}

	if err := db.Exec(`INSERT INTO rules_old (id, name, send, comment, path,
		local_dir, remote_dir, tmp_local_receive_dir) SELECT id, name, is_send, 
		comment, path, local_dir, remote_dir, tmp_local_receive_dir FROM rules`); err != nil {
		return fmt.Errorf("failed to copy the content of the rules table: %w", err)
	}

	if err := db.DropTable("rules"); err != nil {
		return fmt.Errorf("failed to drop the rules table: %w", err)
	}

	if err := db.RenameTable("rules_old", "rules"); err != nil {
		return fmt.Errorf("failed to rename the rules table: %w", err)
	}

	return nil
}

type ver0_7_0RevampTasksTable struct{}

func (ver0_7_0RevampTasksTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("tasks_new",
		migration.Col("rule_id", migration.BigInt, migration.NotNull, migration.ForeignKey("rules", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("chain", migration.Varchar(10), migration.NotNull),
		migration.Col("rank", migration.SmallInt, migration.NotNull),
		migration.Col("type", migration.Varchar(50), migration.NotNull),
		migration.Col("args", migration.Text, migration.NotNull, migration.Default("{}")),
		migration.Check("chain = 'PRE' OR chain = 'POST' OR chain = 'ERROR'"),
		migration.MultiUnique("unique_task_nb", "rule_id", "chain", "rank"),
	); err != nil {
		return fmt.Errorf("failed to create the new tasks table: %w", err)
	}

	args := "args"
	if db.GetDialect() == PostgreSQL {
		args = "ENCODE(args, 'escape')"
	}

	if err := db.Exec(`INSERT INTO tasks_new (rule_id, chain, rank, type, args) 
		SELECT rule_id, chain, rank, type, ` + args + ` FROM tasks`); err != nil {
		return fmt.Errorf("failed to copy the content of the tasks table: %w", err)
	}

	if err := db.DropTable("tasks"); err != nil {
		return fmt.Errorf("failed to drop the tasks table: %w", err)
	}

	if err := db.RenameTable("tasks_new", "tasks"); err != nil {
		return fmt.Errorf("failed to rename the tasks table: %w", err)
	}

	return nil
}

func (ver0_7_0RevampTasksTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("tasks_old",
		migration.Col("rule_id", migration.BigInt, migration.NotNull),
		migration.Col("chain", migration.Varchar(255), migration.NotNull),
		migration.Col("rank", migration.Integer, migration.NotNull),
		migration.Col("type", migration.Varchar(255), migration.NotNull),
		migration.Col("args", migration.Blob, migration.NotNull),
	); err != nil {
		return fmt.Errorf("failed to create the new tasks table: %w", err)
	}

	args := "args"
	if db.GetDialect() == PostgreSQL {
		args = "DECODE(args, 'escape')"
	}

	if err := db.Exec(`INSERT INTO tasks_old (rule_id, chain, rank, type, args) 
		SELECT rule_id, chain, rank, type, ` + args + ` FROM tasks`); err != nil {
		return fmt.Errorf("failed to copy the content of the tasks table: %w", err)
	}

	if err := db.DropTable("tasks"); err != nil {
		return fmt.Errorf("failed to drop the tasks table: %w", err)
	}

	if err := db.RenameTable("tasks_old", "tasks"); err != nil {
		return fmt.Errorf("failed to rename the tasks table: %w", err)
	}

	return nil
}

type ver0_7_0RevampHistoryTable struct{}

func (ver0_7_0RevampHistoryTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("transfer_history_new",
		migration.Col("id", migration.BigInt, migration.PrimaryKey),
		migration.Col("owner", migration.Varchar(100), migration.NotNull),
		migration.Col("remote_transfer_id", migration.Varchar(100), migration.NotNull),
		migration.Col("is_server", migration.Boolean, migration.NotNull),
		migration.Col("is_send", migration.Boolean, migration.NotNull),
		migration.Col("rule", migration.Varchar(100), migration.NotNull),
		migration.Col("account", migration.Varchar(100), migration.NotNull),
		migration.Col("agent", migration.Varchar(100), migration.NotNull),
		migration.Col("protocol", migration.Varchar(50), migration.NotNull),
		migration.Col("local_path", migration.Text, migration.NotNull),
		migration.Col("remote_path", migration.Text, migration.NotNull),
		migration.Col("filesize", migration.BigInt, migration.NotNull, migration.Default(-1)),
		migration.Col("start", migration.DateTime, migration.NotNull),
		migration.Col("stop", migration.DateTime),
		migration.Col("status", migration.Varchar(50), migration.NotNull),
		migration.Col("step", migration.Varchar(50), migration.NotNull),
		migration.Col("progress", migration.BigInt, migration.NotNull, migration.Default(0)),
		migration.Col("task_number", migration.SmallInt, migration.NotNull, migration.Default(0)),
		migration.Col("error_code", migration.Varchar(50), migration.NotNull, migration.Default("TeOk")),
		migration.Col("error_details", migration.Text, migration.NotNull, migration.Default("")),
		migration.MultiUnique("unique_history_id", "remote_transfer_id",
			"is_server", "account", "agent"),
	); err != nil {
		return fmt.Errorf("failed to create the new transfer_history table: %w", err)
	}

	if db.GetDialect() == PostgreSQL {
		if err := db.Exec("SET TimeZone = 'UTC'"); err != nil {
			return fmt.Errorf("failed to set the PostgreSQL time zone: %w", err)
		}
	}

	start, stop := "start", "stop" //nolint:goconst //other instances are about different matters

	if db.GetDialect() == MySQL || db.GetDialect() == SQLite {
		start = "REPLACE(REPLACE(start, 'T', ' '), 'Z', '')"
		stop = "REPLACE(REPLACE(stop, 'T', ' '), 'Z', '')"
	}

	if err := db.Exec(`INSERT INTO transfer_history_new (id, owner, is_server,
		is_send, remote_transfer_id, rule, account, agent, protocol, local_path,
		remote_path, filesize, start, stop, status, step, progress, task_number,
		error_code, error_details) SELECT id, owner, is_server,	is_send, remote_transfer_id,
		rule, account, agent, protocol, local_path,	remote_path, filesize, ` +
		start + `, ` + stop + `, status, step, progression, task_number, error_code,
		error_details FROM transfer_history`); err != nil {
		return fmt.Errorf("failed to copy the content of the transfer_history table: %w", err)
	}

	if err := db.DropTable("transfer_history"); err != nil {
		return fmt.Errorf("failed to drop the transfer_history table: %w", err)
	}

	if err := db.RenameTable("transfer_history_new", "transfer_history"); err != nil {
		return fmt.Errorf("failed to rename the transfer_history table: %w", err)
	}

	return nil
}

//nolint:funlen //splitting hurts readability
func (ver0_7_0RevampHistoryTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("transfer_history_old",
		migration.Col("id", migration.BigInt, migration.PrimaryKey),
		migration.Col("owner", migration.Varchar(255), migration.NotNull),
		migration.Col("remote_transfer_id", migration.Varchar(255), migration.NotNull),
		migration.Col("is_server", migration.Boolean, migration.NotNull),
		migration.Col("is_send", migration.Boolean, migration.NotNull),
		migration.Col("rule", migration.Varchar(255), migration.NotNull),
		migration.Col("account", migration.Varchar(255), migration.NotNull),
		migration.Col("agent", migration.Varchar(255), migration.NotNull),
		migration.Col("protocol", migration.Varchar(255), migration.NotNull),
		migration.Col("local_path", migration.Varchar(255), migration.NotNull),
		migration.Col("remote_path", migration.Varchar(255), migration.NotNull),
		migration.Col("filesize", migration.BigInt, migration.NotNull, migration.Default(-1)),
		migration.Col("start", migration.Timestampz, migration.NotNull),
		migration.Col("stop", migration.Timestampz),
		migration.Col("status", migration.Varchar(50), migration.NotNull),
		migration.Col("step", migration.Varchar(50), migration.NotNull),
		migration.Col("progression", migration.BigInt, migration.NotNull),
		migration.Col("task_number", migration.BigInt, migration.NotNull),
		migration.Col("error_code", migration.Varchar(50), migration.NotNull),
		migration.Col("error_details", migration.Varchar(255), migration.NotNull),
	); err != nil {
		return fmt.Errorf("failed to create the new transfer_history table: %w", err)
	}

	if db.GetDialect() == PostgreSQL {
		if err := db.Exec("SET TimeZone = 'UTC'"); err != nil {
			return fmt.Errorf("failed to set the PostgreSQL time zone: %w", err)
		}
	}

	start, stop := "start", "stop"

	if db.GetDialect() == MySQL {
		start = "CONCAT(REPLACE(start, ' ', 'T'), 'Z')"
		stop = "CONCAT(REPLACE(stop, ' ', 'T'), 'Z')"
	} else if db.GetDialect() == SQLite {
		start = "REPLACE(start, ' ', 'T') || 'Z'"
		stop = "REPLACE(stop, ' ', 'T') || 'Z'"
	}

	if err := db.Exec(`INSERT INTO transfer_history_old (id, owner, is_server,
		is_send, remote_transfer_id, rule, account, agent, protocol, local_path,
		remote_path, filesize, start, stop, status, step, progression, task_number,
		error_code, error_details) SELECT id, owner, is_server,	is_send, remote_transfer_id,
		rule, account, agent, protocol, local_path,	remote_path, filesize, ` +
		start + `, ` + stop + `, status, step, progress, task_number, error_code,
		error_details FROM transfer_history`); err != nil {
		return fmt.Errorf("failed to copy the content of the transfer_history table: %w", err)
	}

	if err := db.DropTable("transfer_history"); err != nil {
		return fmt.Errorf("failed to drop the transfer_history table: %w", err)
	}

	if err := db.RenameTable("transfer_history_old", "transfer_history"); err != nil {
		return fmt.Errorf("failed to rename the transfer_history table: %w", err)
	}

	return nil
}

type ver0_7_0RevampTransfersTable struct{}

func (ver0_7_0RevampTransfersTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("transfers_new",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("owner", migration.Varchar(100), migration.NotNull),
		migration.Col("remote_transfer_id", migration.Varchar(100), migration.NotNull, migration.Default("")),
		migration.Col("rule_id", migration.BigInt, migration.NotNull, migration.ForeignKey("rules", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Restrict)),
		migration.Col("local_account_id", migration.BigInt, migration.ForeignKey("local_accounts", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Restrict)),
		migration.Col("remote_account_id", migration.BigInt, migration.ForeignKey("remote_accounts", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Restrict)),
		migration.Col("local_path", migration.Text, migration.NotNull),
		migration.Col("remote_path", migration.Text, migration.NotNull),
		migration.Col("filesize", migration.BigInt, migration.NotNull, migration.Default(-1)),
		migration.Col("start", migration.DateTime, migration.NotNull, migration.Default(migration.CurrentTimestamp)),
		migration.Col("status", migration.Varchar(50), migration.NotNull, migration.Default("PLANNED")),
		migration.Col("step", migration.Varchar(50), migration.NotNull, migration.Default("StepNone")),
		migration.Col("progress", migration.BigInt, migration.NotNull, migration.Default(0)),
		migration.Col("task_number", migration.SmallInt, migration.NotNull, migration.Default(0)),
		migration.Col("error_code", migration.Varchar(50), migration.NotNull, migration.Default("TeOk")),
		migration.Col("error_details", migration.Text, migration.NotNull, migration.Default("")),
		migration.MultiUnique("unique_transfer_local_account", "remote_transfer_id", "local_account_id"),
		migration.MultiUnique("unique_transfer_remote_account", "remote_transfer_id", "remote_account_id"),
		migration.Check(utils.CheckOnlyOneNotNull(db.GetDialect(), "local_account_id", "remote_account_id")),
	); err != nil {
		return fmt.Errorf("failed to create the new transfers table: %w", err)
	}

	if db.GetDialect() == PostgreSQL {
		if err := db.Exec("SET TimeZone = 'UTC'"); err != nil {
			return fmt.Errorf("failed to set the PostgreSQL time zone: %w", err)
		}
	}

	start := "start"

	if db.GetDialect() == MySQL || db.GetDialect() == SQLite {
		start = "REPLACE(REPLACE(start, 'T', ' '), 'Z', '')"
	}

	if err := db.Exec(`INSERT INTO transfers_new (id, owner, remote_transfer_id,
		rule_id, local_account_id, remote_account_id, local_path, remote_path, 
		filesize, start, status, step, progress, task_number, error_code, 
		error_details) SELECT id, owner, remote_transfer_id, rule_id, 
		(CASE WHEN is_server THEN account_id END),
		(CASE WHEN NOT is_server THEN account_id END), 
		local_path, remote_path, filesize, ` + start + `, status, step,	progression, 
		task_number, error_code, error_details FROM transfers`); err != nil {
		return fmt.Errorf("failed to copy the content of the transfers table: %w", err)
	}

	if err := db.DropTable("transfers"); err != nil {
		return fmt.Errorf("failed to drop the transfers table: %w", err)
	}

	if err := db.RenameTable("transfers_new", "transfers"); err != nil {
		return fmt.Errorf("failed to rename the transfers table: %w", err)
	}

	return nil
}

//nolint:funlen //splitting hurts readability
func (ver0_7_0RevampTransfersTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("transfers_old",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("owner", migration.Varchar(255), migration.NotNull),
		migration.Col("remote_transfer_id", migration.Varchar(255), migration.NotNull),
		migration.Col("rule_id", migration.BigInt, migration.NotNull),
		migration.Col("is_server", migration.Boolean, migration.NotNull),
		migration.Col("agent_id", migration.BigInt, migration.NotNull),
		migration.Col("account_id", migration.BigInt, migration.NotNull),
		migration.Col("local_path", migration.Varchar(255), migration.NotNull),
		migration.Col("remote_path", migration.Varchar(255), migration.NotNull),
		migration.Col("filesize", migration.BigInt, migration.NotNull, migration.Default(-1)),
		migration.Col("start", migration.Timestampz, migration.NotNull),
		migration.Col("status", migration.Varchar(255), migration.NotNull),
		migration.Col("step", migration.Varchar(50), migration.NotNull),
		migration.Col("progression", migration.BigInt, migration.NotNull),
		migration.Col("task_number", migration.BigInt, migration.NotNull),
		migration.Col("error_code", migration.Varchar(50), migration.NotNull),
		migration.Col("error_details", migration.Varchar(255), migration.NotNull),
	); err != nil {
		return fmt.Errorf("failed to create the new transfers table: %w", err)
	}

	if db.GetDialect() == PostgreSQL {
		if err := db.Exec("SET TimeZone = 'UTC'"); err != nil {
			return fmt.Errorf("failed to set the PostgreSQL time zone: %w", err)
		}
	}

	start := "start"

	if db.GetDialect() == MySQL {
		start = "CONCAT(REPLACE(start, ' ', 'T'), 'Z')"
	} else if db.GetDialect() == SQLite {
		start = "REPLACE(start, ' ', 'T') || 'Z'"
	}

	if err := db.Exec(`INSERT INTO transfers_old (id, owner, remote_transfer_id,
		rule_id, is_server, agent_id, account_id, local_path, remote_path, filesize,
		start, status, step, progression, task_number, error_code, error_details)
		SELECT id, owner, remote_transfer_id, rule_id, 
		(CASE WHEN local_account_id IS NULL THEN FALSE ELSE TRUE END),
		(CASE WHEN local_account_id IS NOT NULL 
			THEN (SELECT local_agent_id FROM local_accounts WHERE id=local_account_id)
			ELSE (SELECT remote_agent_id FROM remote_accounts WHERE id=remote_account_id) 
		END), 
		(CASE WHEN local_account_id IS NULL THEN remote_account_id ELSE local_account_id END),
		local_path, remote_path, filesize, ` + start + `, status, step, progress, 
		task_number, error_code, error_details FROM transfers`); err != nil {
		return fmt.Errorf("failed to copy the content of the transfers table: %w", err)
	}

	if err := db.DropTable("transfers"); err != nil {
		return fmt.Errorf("failed to drop the transfers table: %w", err)
	}

	if err := db.RenameTable("transfers_old", "transfers"); err != nil {
		return fmt.Errorf("failed to rename the transfers table: %w", err)
	}

	return nil
}

type ver0_7_0RevampTransferInfoTable struct{}

func (ver0_7_0RevampTransferInfoTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("transfer_info_new",
		migration.Col("transfer_id", migration.BigInt, migration.ForeignKey("transfers", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("history_id", migration.BigInt, migration.ForeignKey("transfer_history", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("name", migration.Varchar(100), migration.NotNull),
		migration.Col("value", migration.Text, migration.NotNull, migration.Default("null")),
		migration.MultiUnique("unique_transfer_info_tran", "transfer_id", "name"),
		migration.MultiUnique("unique_transfer_info_hist", "history_id", "name"),
		migration.Check(utils.CheckOnlyOneNotNull(db.GetDialect(), "transfer_id", "history_id")),
	); err != nil {
		return fmt.Errorf("failed to create the new transfer_info table: %w", err)
	}

	if err := db.Exec(`INSERT INTO transfer_info_new (transfer_id, history_id,
		name, value) SELECT 
		(CASE WHEN NOT is_history THEN transfer_id END), 
		(CASE WHEN is_history THEN transfer_id END), 
		name, value FROM transfer_info`); err != nil {
		return fmt.Errorf("failed to copy the content of the transfer_info table: %w", err)
	}

	if err := db.DropTable("transfer_info"); err != nil {
		return fmt.Errorf("failed to drop the transfer_info table: %w", err)
	}

	if err := db.RenameTable("transfer_info_new", "transfer_info"); err != nil {
		return fmt.Errorf("failed to rename the transfer_info table: %w", err)
	}

	return nil
}

func (ver0_7_0RevampTransferInfoTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("transfer_info_old",
		migration.Col("transfer_id", migration.BigInt, migration.NotNull),
		migration.Col("is_history", migration.Boolean, migration.NotNull),
		migration.Col("name", migration.Varchar(255), migration.NotNull),
		migration.Col("value", migration.Text, migration.NotNull),
		migration.MultiUnique("UQE_transfer_info_old_infoName", "transfer_id", "name"),
	); err != nil {
		return fmt.Errorf("failed to create the new transfer_info table: %w", err)
	}

	if err := db.Exec(`INSERT INTO transfer_info_old (transfer_id, is_history,
		name, value) SELECT 
		(CASE WHEN transfer_id IS NULL THEN history_id ELSE transfer_id END), 
		(CASE WHEN transfer_id IS NULL THEN true ELSE false END), 
		name, value FROM transfer_info`); err != nil {
		return fmt.Errorf("failed to copy the content of the transfer_info table: %w", err)
	}

	if err := db.DropTable("transfer_info"); err != nil {
		return fmt.Errorf("failed to drop the transfer_info table: %w", err)
	}

	if err := db.RenameTable("transfer_info_old", "transfer_info"); err != nil {
		return fmt.Errorf("failed to rename the transfer_info table: %w", err)
	}

	return nil
}

type ver0_7_0RevampCryptoTable struct{}

func (ver0_7_0RevampCryptoTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("crypto_credentials_new",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("name", migration.Varchar(100), migration.NotNull),
		migration.Col("local_agent_id", migration.BigInt, migration.ForeignKey("local_agents", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("remote_agent_id", migration.BigInt, migration.ForeignKey("remote_agents", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("local_account_id", migration.BigInt, migration.ForeignKey("local_accounts", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("remote_account_id", migration.BigInt, migration.ForeignKey("remote_accounts", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("private_key", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("certificate", migration.Text, migration.NotNull, migration.Default("")),
		migration.Col("ssh_public_key", migration.Text, migration.NotNull, migration.Default("")),
		migration.MultiUnique("unique_crypto_credentials_loc_agent", "local_agent_id", "name"),
		migration.MultiUnique("unique_crypto_credentials_rem_agent", "remote_agent_id", "name"),
		migration.MultiUnique("unique_crypto_credentials_loc_account", "local_account_id", "name"),
		migration.MultiUnique("unique_crypto_credentials_rem_account", "remote_account_id", "name"),
		migration.Check(utils.CheckOnlyOneNotNull(db.GetDialect(), "local_agent_id",
			"remote_agent_id", "local_account_id", "remote_account_id")),
	); err != nil {
		return fmt.Errorf("failed to create the new crypto_credentials table: %w", err)
	}

	if err := db.Exec(`INSERT INTO crypto_credentials_new (id, name, local_agent_id, 
		remote_agent_id, local_account_id, remote_account_id, private_key, 
		certificate, ssh_public_key) SELECT id, name,
		(CASE WHEN owner_type='local_agents' THEN owner_id END),
		(CASE WHEN owner_type='remote_agents' THEN owner_id END),
		(CASE WHEN owner_type='local_accounts' THEN owner_id END),
		(CASE WHEN owner_type='remote_accounts' THEN owner_id END),
		private_key, certificate, ssh_public_key FROM crypto_credentials`); err != nil {
		return fmt.Errorf("failed to copy the content of the crypto_credentials table: %w", err)
	}

	if err := db.DropTable("crypto_credentials"); err != nil {
		return fmt.Errorf("failed to drop the crypto_credentials table: %w", err)
	}

	if err := db.RenameTable("crypto_credentials_new", "crypto_credentials"); err != nil {
		return fmt.Errorf("failed to rename the crypto_credentials table: %w", err)
	}

	return nil
}

func (ver0_7_0RevampCryptoTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("crypto_credentials_old",
		migration.Col("id", migration.BigInt, migration.PrimaryKey, migration.AutoIncr),
		migration.Col("name", migration.Varchar(100), migration.NotNull),
		migration.Col("owner_type", migration.Varchar(255), migration.NotNull),
		migration.Col("owner_id", migration.BigInt, migration.NotNull),
		migration.Col("private_key", migration.Text),
		migration.Col("certificate", migration.Text),
		migration.Col("ssh_public_key", migration.Text),
		migration.MultiUnique("UQE_crypto_credentials_old_cert", "name", "owner_type", "owner_id"),
	); err != nil {
		return fmt.Errorf("failed to create the new crypto_credentials table: %w", err)
	}

	if err := db.Exec(`INSERT INTO crypto_credentials_old (id, name, owner_type,
		owner_id, private_key, certificate, ssh_public_key) SELECT id, name,
		(CASE WHEN local_agent_id IS NOT NULL THEN 'local_agents'
			  WHEN remote_agent_id IS NOT NULL THEN 'remote_agents'
			  WHEN local_account_id IS NOT NULL THEN 'local_accounts'
			  WHEN remote_account_id IS NOT NULL THEN 'remote_accounts' END),
		(CASE WHEN local_agent_id IS NOT NULL THEN local_agent_id
			  WHEN remote_agent_id IS NOT NULL THEN remote_agent_id
			  WHEN local_account_id IS NOT NULL THEN local_account_id
			  WHEN remote_account_id IS NOT NULL THEN remote_account_id END),
		private_key, certificate, ssh_public_key FROM crypto_credentials`); err != nil {
		return fmt.Errorf("failed to copy the content of the crypto_credentials table: %w", err)
	}

	if err := db.DropTable("crypto_credentials"); err != nil {
		return fmt.Errorf("failed to drop the crypto_credentials table: %w", err)
	}

	if err := db.RenameTable("crypto_credentials_old", "crypto_credentials"); err != nil {
		return fmt.Errorf("failed to rename the crypto_credentials table: %w", err)
	}

	return nil
}

type ver0_7_0RevampRuleAccessTable struct{}

func (ver0_7_0RevampRuleAccessTable) Up(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("rule_access_new",
		migration.Col("rule_id", migration.BigInt, migration.NotNull, migration.ForeignKey("rules", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("local_agent_id", migration.BigInt, migration.ForeignKey("local_agents", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("remote_agent_id", migration.BigInt, migration.ForeignKey("remote_agents", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("local_account_id", migration.BigInt, migration.ForeignKey("local_accounts", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.Col("remote_account_id", migration.BigInt, migration.ForeignKey("remote_accounts", "id").
			OnUpdate(migration.Restrict).OnDelete(migration.Cascade)),
		migration.MultiUnique("unique_rule_access_loc_agent", "rule_id", "local_agent_id"),
		migration.MultiUnique("unique_rule_access_rem_agent", "rule_id", "remote_agent_id"),
		migration.MultiUnique("unique_rule_access_loc_account", "rule_id", "local_account_id"),
		migration.MultiUnique("unique_rule_access_rem_account", "rule_id", "remote_account_id"),
		migration.Check(utils.CheckOnlyOneNotNull(db.GetDialect(), "local_agent_id",
			"remote_agent_id", "local_account_id", "remote_account_id")),
	); err != nil {
		return fmt.Errorf("failed to create the new rule_access table: %w", err)
	}

	if err := db.Exec(`INSERT INTO rule_access_new (rule_id, local_agent_id, 
		remote_agent_id, local_account_id, remote_account_id) SELECT rule_id,
		(CASE WHEN object_type='local_agents' THEN object_id END),
		(CASE WHEN object_type='remote_agents' THEN object_id END),
		(CASE WHEN object_type='local_accounts' THEN object_id END),
		(CASE WHEN object_type='remote_accounts' THEN object_id END)
		FROM rule_access`); err != nil {
		return fmt.Errorf("failed to copy the content of the rule_access table: %w", err)
	}

	if err := db.DropTable("rule_access"); err != nil {
		return fmt.Errorf("failed to drop the rule_access table: %w", err)
	}

	if err := db.RenameTable("rule_access_new", "rule_access"); err != nil {
		return fmt.Errorf("failed to rename the rule_access table: %w", err)
	}

	return nil
}

func (ver0_7_0RevampRuleAccessTable) Down(db migration.Actions) error {
	//nolint:gomnd //magic numbers are required here for variable length types
	if err := db.CreateTable("rule_access_old",
		migration.Col("rule_id", migration.BigInt, migration.NotNull),
		migration.Col("object_type", migration.Varchar(255), migration.NotNull),
		migration.Col("object_id", migration.BigInt, migration.NotNull),
		migration.MultiUnique("UQE_rule_access_old_perm", "rule_id", "object_type", "object_id"),
	); err != nil {
		return fmt.Errorf("failed to create the new rule_access table: %w", err)
	}

	if err := db.Exec(`INSERT INTO rule_access_old (rule_id, object_type, 
		object_id) SELECT rule_id,
		(CASE WHEN local_agent_id IS NOT NULL THEN 'local_agents'
			  WHEN remote_agent_id IS NOT NULL THEN 'remote_agents'
			  WHEN local_account_id IS NOT NULL THEN 'local_accounts'
			  WHEN remote_account_id IS NOT NULL THEN 'remote_accounts' END),
		(CASE WHEN local_agent_id IS NOT NULL THEN local_agent_id
			  WHEN remote_agent_id IS NOT NULL THEN remote_agent_id
			  WHEN local_account_id IS NOT NULL THEN local_account_id
			  WHEN remote_account_id IS NOT NULL THEN remote_account_id END)
		FROM rule_access`); err != nil {
		return fmt.Errorf("failed to copy the content of the rule_access table: %w", err)
	}

	if err := db.DropTable("rule_access"); err != nil {
		return fmt.Errorf("failed to drop the rule_access table: %w", err)
	}

	if err := db.RenameTable("rule_access_old", "rule_access"); err != nil {
		return fmt.Errorf("failed to rename the rule_access table: %w", err)
	}

	return nil
}
