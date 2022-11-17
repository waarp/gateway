package migrations

func initTableList() []string {
	return []string{
		"crypto_credentials", "rule_access", "transfer_info", "transfer_history",
		"transfers", "local_accounts", "local_agents", "remote_accounts",
		"remote_agents", "tasks", "rules", "users", "version",
	}
}

//nolint:lll // this won't change a lot, readability might not be that important
const SqliteCreationScript = `
CREATE TABLE version (current TEXT NOT NULL);
INSERT INTO version (current) VALUES ('0.4.0');
CREATE TABLE crypto_credentials (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, owner_type TEXT NOT NULL, owner_id INTEGER NOT NULL, name TEXT NOT NULL, private_key TEXT NULL, certificate TEXT NULL, ssh_public_key TEXT NULL);
CREATE UNIQUE INDEX UQE_crypto_credentials_cert ON crypto_credentials (owner_type,owner_id,name);
CREATE TABLE transfer_history (id INTEGER PRIMARY KEY NOT NULL, owner TEXT NOT NULL, remote_transfer_id TEXT NULL, is_server INTEGER NOT NULL, is_send INTEGER NOT NULL, account TEXT NOT NULL, agent TEXT NOT NULL, protocol TEXT NOT NULL, source_filename TEXT NOT NULL, dest_filename TEXT NOT NULL, rule TEXT NOT NULL, start TEXT NOT NULL, stop TEXT NULL, status TEXT NOT NULL, error_code TEXT NOT NULL, error_details TEXT NOT NULL, step TEXT NOT NULL, progression INTEGER NOT NULL, task_number INTEGER NOT NULL);
CREATE UNIQUE INDEX UQE_transfer_history_histRemID ON transfer_history (remote_transfer_id,account,agent);
CREATE TABLE local_accounts (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, local_agent_id INTEGER NOT NULL, login TEXT NOT NULL, password_hash TEXT NULL);
CREATE UNIQUE INDEX UQE_local_accounts_loc_ac ON local_accounts (local_agent_id,login);
CREATE TABLE local_agents (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, owner TEXT NOT NULL, name TEXT NOT NULL, protocol TEXT NOT NULL, root TEXT NOT NULL, in_dir TEXT NOT NULL, out_dir TEXT NOT NULL, work_dir TEXT NOT NULL, proto_config BLOB NOT NULL, address TEXT NOT NULL);
CREATE UNIQUE INDEX UQE_local_agents_loc_ag ON local_agents (owner,name);
CREATE TABLE remote_accounts (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, remote_agent_id INTEGER NOT NULL, login TEXT NOT NULL, password TEXT NULL);
CREATE UNIQUE INDEX UQE_remote_accounts_rem_ac ON remote_accounts (remote_agent_id,login);
CREATE TABLE remote_agents (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, name TEXT NOT NULL, protocol TEXT NOT NULL, proto_config BLOB NOT NULL, address TEXT NOT NULL);
CREATE UNIQUE INDEX UQE_remote_agents_name ON remote_agents (name);
CREATE TABLE rules (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, name TEXT NOT NULL, comment TEXT NOT NULL, send INTEGER NOT NULL, path TEXT NOT NULL, in_path TEXT NOT NULL, out_path TEXT NOT NULL, work_path TEXT NOT NULL);
CREATE UNIQUE INDEX UQE_rules_path ON rules (send,path);
CREATE UNIQUE INDEX UQE_rules_dir ON rules (name,send);
CREATE TABLE rule_access (rule_id INTEGER NOT NULL, object_id INTEGER NOT NULL, object_type TEXT NOT NULL);
CREATE UNIQUE INDEX UQE_rule_access_perm ON rule_access (rule_id,object_id,object_type);
CREATE TABLE tasks (rule_id INTEGER NOT NULL, chain TEXT NOT NULL, rank INTEGER NOT NULL, type TEXT NOT NULL, args BLOB NOT NULL);
CREATE TABLE transfers (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, remote_transfer_id TEXT NOT NULL, rule_id INTEGER NOT NULL, is_server INTEGER NOT NULL, agent_id INTEGER NOT NULL, account_id INTEGER NOT NULL, true_filepath TEXT NOT NULL, source_file TEXT NOT NULL, dest_file TEXT NOT NULL, start TEXT NOT NULL, step TEXT NOT NULL, status TEXT NOT NULL, owner TEXT NOT NULL, progression INTEGER NOT NULL, task_number INTEGER NOT NULL, error_code TEXT NOT NULL, error_details TEXT NOT NULL);
CREATE TABLE transfer_info (transfer_id INTEGER NOT NULL, name TEXT NOT NULL, value TEXT NOT NULL);
CREATE UNIQUE INDEX UQE_transfer_info_infoName ON transfer_info (transfer_id,name);
CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, owner TEXT NOT NULL, username TEXT NOT NULL, password BLOB NOT NULL, permissions BLOB NOT NULL);
CREATE UNIQUE INDEX UQE_users_name ON users (owner,username);`

//nolint:lll // this won't change a lot, readability might not be that important
const PostgresCreationScript = `
CREATE TABLE "version" ("current" TEXT NOT NULL);
INSERT INTO "version" ("current") VALUES ('0.4.0');
CREATE TABLE "crypto_credentials" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "owner_type" VARCHAR(255) NOT NULL, "owner_id" BIGINT NOT NULL, "name" VARCHAR(255) NOT NULL, "private_key" TEXT NULL, "certificate" TEXT NULL, "ssh_public_key" TEXT NULL);
CREATE UNIQUE INDEX "UQE_crypto_credentials_cert" ON "crypto_credentials" ("owner_type","owner_id","name");
CREATE TABLE "transfer_history" ("id" BIGINT PRIMARY KEY NOT NULL, "owner" VARCHAR(255) NOT NULL, "remote_transfer_id" VARCHAR(255) NULL, "is_server" BOOL NOT NULL, "is_send" BOOL NOT NULL, "account" VARCHAR(255) NOT NULL, "agent" VARCHAR(255) NOT NULL, "protocol" VARCHAR(255) NOT NULL, "source_filename" VARCHAR(255) NOT NULL, "dest_filename" VARCHAR(255) NOT NULL, "rule" VARCHAR(255) NOT NULL, "start" timestamp with time zone NOT NULL, "stop" timestamp with time zone NULL, "status" VARCHAR(50) NOT NULL, "error_code" VARCHAR(50) NOT NULL, "error_details" VARCHAR(255) NOT NULL, "step" VARCHAR(50) NOT NULL, "progression" BIGINT NOT NULL, "task_number" BIGINT NOT NULL);
CREATE UNIQUE INDEX "UQE_transfer_history_histRemID" ON "transfer_history" ("remote_transfer_id","account","agent");
CREATE TABLE "local_accounts" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "local_agent_id" BIGINT NOT NULL, "login" VARCHAR(255) NOT NULL, "password_hash" TEXT NULL);
CREATE UNIQUE INDEX "UQE_local_accounts_loc_ac" ON "local_accounts" ("local_agent_id","login");
CREATE TABLE "local_agents" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "owner" VARCHAR(255) NOT NULL, "name" VARCHAR(255) NOT NULL, "protocol" VARCHAR(255) NOT NULL, "root" VARCHAR(255) NOT NULL, "in_dir" VARCHAR(255) NOT NULL, "out_dir" VARCHAR(255) NOT NULL, "work_dir" VARCHAR(255) NOT NULL, "proto_config" BYTEA NOT NULL, "address" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_local_agents_loc_ag" ON "local_agents" ("owner","name");
CREATE TABLE "remote_accounts" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "remote_agent_id" BIGINT NOT NULL, "login" VARCHAR(255) NOT NULL, "password" TEXT NULL);
CREATE UNIQUE INDEX "UQE_remote_accounts_rem_ac" ON "remote_accounts" ("remote_agent_id","login");
CREATE TABLE "remote_agents" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "name" VARCHAR(255) NOT NULL, "protocol" VARCHAR(255) NOT NULL, "proto_config" BYTEA NOT NULL, "address" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_remote_agents_name" ON "remote_agents" ("name");
CREATE TABLE "rules" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "name" VARCHAR(255) NOT NULL, "comment" VARCHAR(255) NOT NULL, "send" BOOL NOT NULL, "path" VARCHAR(255) NOT NULL, "in_path" VARCHAR(255) NOT NULL, "out_path" VARCHAR(255) NOT NULL, "work_path" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_rules_dir" ON "rules" ("name","send");
CREATE UNIQUE INDEX "UQE_rules_path" ON "rules" ("send","path");
CREATE TABLE "rule_access" ("rule_id" BIGINT NOT NULL, "object_id" BIGINT NOT NULL, "object_type" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_rule_access_perm" ON "rule_access" ("rule_id","object_id","object_type");
CREATE TABLE "tasks" ("rule_id" BIGINT NOT NULL, "chain" VARCHAR(255) NOT NULL, "rank" BIGINT NOT NULL, "type" VARCHAR(255) NOT NULL, "args" BYTEA NOT NULL);
CREATE TABLE "transfers" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "remote_transfer_id" VARCHAR(255) NOT NULL, "rule_id" BIGINT NOT NULL, "is_server" BOOL NOT NULL, "agent_id" BIGINT NOT NULL, "account_id" BIGINT NOT NULL, "true_filepath" VARCHAR(255) NOT NULL, "source_file" VARCHAR(255) NOT NULL, "dest_file" VARCHAR(255) NOT NULL, "start" timestamp with time zone NOT NULL, "step" VARCHAR(50) NOT NULL, "status" VARCHAR(255) NOT NULL, "owner" VARCHAR(255) NOT NULL, "progression" BIGINT NOT NULL, "task_number" BIGINT NOT NULL, "error_code" VARCHAR(50) NOT NULL, "error_details" VARCHAR(255) NOT NULL);
CREATE TABLE "transfer_info" ("transfer_id" BIGINT NOT NULL, "name" VARCHAR(255) NOT NULL, "value" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_transfer_info_infoName" ON "transfer_info" ("transfer_id","name");
CREATE TABLE "users" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "owner" VARCHAR(255) NOT NULL, "username" VARCHAR(255) NOT NULL, "password" BYTEA NOT NULL, "permissions" BYTEA NOT NULL);
CREATE UNIQUE INDEX "UQE_users_name" ON "users" ("owner","username");`

//nolint:lll // this won't change a lot, readability might not be that important
const MysqlCreationScript = `
CREATE TABLE version (current TEXT NOT NULL);
INSERT INTO version (current) VALUES ('0.4.0');
CREATE TABLE crypto_credentials (id BIGINT(20) UNSIGNED PRIMARY KEY AUTO_INCREMENT NOT NULL, owner_type VARCHAR(255) NOT NULL, owner_id BIGINT(20) UNSIGNED NOT NULL, name VARCHAR(255) NOT NULL, private_key TEXT NULL, certificate TEXT NULL, ssh_public_key TEXT NULL);
CREATE UNIQUE INDEX UQE_crypto_credentials_cert ON crypto_credentials (owner_type,owner_id,name);
CREATE TABLE transfer_history (id BIGINT(20) UNSIGNED PRIMARY KEY NOT NULL, owner VARCHAR(255) NOT NULL, remote_transfer_id VARCHAR(255) NULL, is_server TINYINT(1) NOT NULL, is_send TINYINT(1) NOT NULL, account VARCHAR(255) NOT NULL, agent VARCHAR(255) NOT NULL, protocol VARCHAR(255) NOT NULL, source_filename VARCHAR(255) NOT NULL, dest_filename VARCHAR(255) NOT NULL, rule VARCHAR(255) NOT NULL, start CHAR(64) NOT NULL, stop CHAR(64) NULL, status VARCHAR(50) NOT NULL, error_code VARCHAR(50) NOT NULL, error_details VARCHAR(255) NOT NULL, step VARCHAR(50) NOT NULL, progression BIGINT(20) UNSIGNED NOT NULL, task_number BIGINT(20) UNSIGNED NOT NULL);
CREATE UNIQUE INDEX UQE_transfer_history_histRemID ON transfer_history (remote_transfer_id,account,agent);
CREATE TABLE local_accounts (id BIGINT(20) UNSIGNED PRIMARY KEY AUTO_INCREMENT NOT NULL, local_agent_id BIGINT(20) UNSIGNED NOT NULL, login VARCHAR(255) NOT NULL, password_hash TEXT NULL);
CREATE UNIQUE INDEX UQE_local_accounts_loc_ac ON local_accounts (local_agent_id,login);
CREATE TABLE local_agents (id BIGINT(20) UNSIGNED PRIMARY KEY AUTO_INCREMENT NOT NULL, owner VARCHAR(255) NOT NULL, name VARCHAR(255) NOT NULL, protocol VARCHAR(255) NOT NULL, root VARCHAR(255) NOT NULL, in_dir VARCHAR(255) NOT NULL, out_dir VARCHAR(255) NOT NULL, work_dir VARCHAR(255) NOT NULL, proto_config BLOB NOT NULL, address VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_local_agents_loc_ag ON local_agents (owner,name);
CREATE TABLE remote_accounts (id BIGINT(20) UNSIGNED PRIMARY KEY AUTO_INCREMENT NOT NULL, remote_agent_id BIGINT(20) UNSIGNED NOT NULL, login VARCHAR(255) NOT NULL, password TEXT NULL);
CREATE UNIQUE INDEX UQE_remote_accounts_rem_ac ON remote_accounts (remote_agent_id,login);
CREATE TABLE remote_agents (id BIGINT(20) UNSIGNED PRIMARY KEY AUTO_INCREMENT NOT NULL, name VARCHAR(255) NOT NULL, protocol VARCHAR(255) NOT NULL, proto_config BLOB NOT NULL, address VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_remote_agents_name ON remote_agents (name);
CREATE TABLE rules (id BIGINT(20) UNSIGNED PRIMARY KEY AUTO_INCREMENT NOT NULL, name VARCHAR(255) NOT NULL, comment VARCHAR(255) NOT NULL, send TINYINT(1) NOT NULL, path VARCHAR(255) NOT NULL, in_path VARCHAR(255) NOT NULL, out_path VARCHAR(255) NOT NULL, work_path VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_rules_dir ON rules (name,send);
CREATE UNIQUE INDEX UQE_rules_path ON rules (send,path);
CREATE TABLE rule_access (rule_id BIGINT(20) UNSIGNED NOT NULL, object_id BIGINT(20) UNSIGNED NOT NULL, object_type VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_rule_access_perm ON rule_access (rule_id,object_id,object_type);
CREATE TABLE tasks (rule_id BIGINT(20) UNSIGNED NOT NULL, chain VARCHAR(255) NOT NULL, rank INT UNSIGNED NOT NULL, type VARCHAR(255) NOT NULL, args BLOB NOT NULL);
CREATE TABLE transfers (id BIGINT(20) UNSIGNED PRIMARY KEY AUTO_INCREMENT NOT NULL, remote_transfer_id VARCHAR(255) NOT NULL, rule_id BIGINT(20) UNSIGNED NOT NULL, is_server TINYINT(1) NOT NULL, agent_id BIGINT(20) UNSIGNED NOT NULL, account_id BIGINT(20) UNSIGNED NOT NULL, true_filepath VARCHAR(255) NOT NULL, source_file VARCHAR(255) NOT NULL, dest_file VARCHAR(255) NOT NULL, start CHAR(64) NOT NULL, step VARCHAR(50) NOT NULL, status VARCHAR(255) NOT NULL, owner VARCHAR(255) NOT NULL, progression BIGINT(20) UNSIGNED NOT NULL, task_number BIGINT(20) UNSIGNED NOT NULL, error_code VARCHAR(50) NOT NULL, error_details VARCHAR(255) NOT NULL);
CREATE TABLE transfer_info (transfer_id BIGINT(20) UNSIGNED NOT NULL, name VARCHAR(255) NOT NULL, value VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_transfer_info_infoName ON transfer_info (transfer_id,name);
CREATE TABLE users (id BIGINT(20) UNSIGNED PRIMARY KEY AUTO_INCREMENT NOT NULL, owner VARCHAR(255) NOT NULL, username VARCHAR(255) NOT NULL, password BLOB NOT NULL, permissions BINARY(4) NOT NULL);
CREATE UNIQUE INDEX UQE_users_name ON users (owner,username);`
