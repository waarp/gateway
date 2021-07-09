package migrations

const sqliteCreationScript = `
CREATE TABLE IF NOT EXISTS 'version' ('current' TEXT NOT NULL);
INSERT INTO 'version' ('current') VALUES ('0.0.0');
CREATE TABLE IF NOT EXISTS 'certificates' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'owner_type' TEXT NOT NULL, 'owner_id' INTEGER NOT NULL, 'name' TEXT NOT NULL, 'private_key' BLOB NULL, 'public_key' BLOB NULL, 'cert' BLOB NULL);
CREATE UNIQUE INDEX 'UQE_certificates_cert' ON 'certificates' ('owner_type','owner_id','name');
CREATE TABLE IF NOT EXISTS 'transfer_history' ('id' INTEGER PRIMARY KEY NOT NULL, 'owner' TEXT NOT NULL, 'remote_transfer_id' TEXT NULL, 'is_server' INTEGER NOT NULL, 'is_send' INTEGER NOT NULL, 'account' TEXT NOT NULL, 'agent' TEXT NOT NULL, 'protocol' TEXT NOT NULL, 'source_filename' TEXT NOT NULL, 'dest_filename' TEXT NOT NULL, 'rule' TEXT NOT NULL, 'start' TEXT NOT NULL, 'stop' TEXT NOT NULL, 'status' TEXT NOT NULL, 'error_code' TEXT NOT NULL, 'error_details' TEXT NOT NULL, 'step' TEXT NOT NULL, 'progression' INTEGER NOT NULL, 'task_number' INTEGER NOT NULL);
CREATE UNIQUE INDEX 'UQE_transfer_history_histRemID' ON 'transfer_history' ('remote_transfer_id','account','agent');
CREATE TABLE IF NOT EXISTS 'local_accounts' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'local_agent_id' INTEGER NOT NULL, 'login' TEXT NOT NULL, 'password' BLOB NULL);
CREATE UNIQUE INDEX 'UQE_local_accounts_loc_ac' ON 'local_accounts' ('local_agent_id','login');
CREATE TABLE IF NOT EXISTS 'local_agents' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'owner' TEXT NOT NULL, 'name' TEXT NOT NULL, 'protocol' TEXT NOT NULL, 'root' TEXT NOT NULL, 'in_dir' TEXT NOT NULL, 'out_dir' TEXT NOT NULL, 'work_dir' TEXT NOT NULL, 'proto_config' BLOB NOT NULL, 'address' TEXT NOT NULL);
CREATE UNIQUE INDEX 'UQE_local_agents_loc_ag' ON 'local_agents' ('owner','name');
CREATE TABLE IF NOT EXISTS 'remote_accounts' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'remote_agent_id' INTEGER NOT NULL, 'login' TEXT NOT NULL, 'password' BLOB NULL);
CREATE UNIQUE INDEX 'UQE_remote_accounts_rem_ac' ON 'remote_accounts' ('remote_agent_id','login');
CREATE TABLE IF NOT EXISTS 'remote_agents' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'name' TEXT NOT NULL, 'protocol' TEXT NOT NULL, 'proto_config' BLOB NOT NULL, 'address' TEXT NOT NULL);
CREATE UNIQUE INDEX 'UQE_remote_agents_name' ON 'remote_agents' ('name');
CREATE TABLE IF NOT EXISTS 'rules' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'name' TEXT NOT NULL, 'comment' TEXT NOT NULL, 'send' INTEGER NOT NULL, 'path' TEXT NOT NULL, 'in_path' TEXT NOT NULL, 'out_path' TEXT NOT NULL, 'work_path' TEXT NOT NULL);
CREATE UNIQUE INDEX 'UQE_rules_dir' ON 'rules' ('name','send');
CREATE UNIQUE INDEX 'UQE_rules_path' ON 'rules' ('send','path');
CREATE TABLE IF NOT EXISTS 'rule_access' ('rule_id' INTEGER NOT NULL, 'object_id' INTEGER NOT NULL, 'object_type' TEXT NOT NULL);
CREATE UNIQUE INDEX 'UQE_rule_access_perm' ON 'rule_access' ('rule_id','object_id','object_type');
CREATE TABLE IF NOT EXISTS 'tasks' ('rule_id' INTEGER NOT NULL, 'chain' TEXT NOT NULL, 'rank' INTEGER NOT NULL, 'type' TEXT NOT NULL, 'args' BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS 'transfers' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'remote_transfer_id' TEXT NULL, 'rule_id' INTEGER NOT NULL, 'is_server' INTEGER NOT NULL, 'agent_id' INTEGER NOT NULL, 'account_id' INTEGER NOT NULL, 'true_filepath' TEXT NOT NULL, 'source_file' TEXT NOT NULL, 'dest_file' TEXT NOT NULL, 'start' TEXT NOT NULL, 'step' TEXT NOT NULL, 'status' TEXT NOT NULL, 'owner' TEXT NOT NULL, 'progression' INTEGER NOT NULL, 'task_number' INTEGER NOT NULL, 'error_code' TEXT NOT NULL, 'error_details' TEXT NOT NULL);
CREATE UNIQUE INDEX 'UQE_transfers_transRemID' ON 'transfers' ('remote_transfer_id','account_id');
CREATE TABLE IF NOT EXISTS 'transfer_info' ('transfer_id' INTEGER NOT NULL, 'name' TEXT NOT NULL, 'value' TEXT NOT NULL);
CREATE UNIQUE INDEX 'UQE_transfer_info_infoName' ON 'transfer_info' ('transfer_id','name');
CREATE TABLE IF NOT EXISTS 'users' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'owner' TEXT NOT NULL, 'username' TEXT NOT NULL, 'password' BLOB NOT NULL, 'permissions' BLOB NOT NULL);
CREATE UNIQUE INDEX 'UQE_users_name' ON 'users' ('owner','username');
INSERT INTO 'users' ('owner','username','password','permissions') VALUES ('test_gateway', 'admin', X'243261243034244E7052683552343875795038504E336A73524C4C5775773734776A54514E394972715A70596A354E384A594D2E7268597877727665', X'DFFFFFFF');
`

const postgresCreationScript = `
CREATE TABLE IF NOT EXISTS "version" ("current" TEXT NOT NULL);
INSERT INTO "version" ("current") VALUES ('0.0.0');
CREATE TABLE IF NOT EXISTS "certificates" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "owner_type" VARCHAR(255) NOT NULL, "owner_id" BIGINT NOT NULL, "name" VARCHAR(255) NOT NULL, "private_key" BYTEA NULL, "public_key" BYTEA NULL, "cert" BYTEA NULL);
CREATE UNIQUE INDEX "UQE_certificates_cert" ON "certificates" ("owner_type","owner_id","name");
CREATE TABLE IF NOT EXISTS "transfer_history" ("id" BIGINT PRIMARY KEY NOT NULL, "owner" VARCHAR(255) NOT NULL, "remote_transfer_id" VARCHAR(255) NULL, "is_server" BOOL NOT NULL, "is_send" BOOL NOT NULL, "account" VARCHAR(255) NOT NULL, "agent" VARCHAR(255) NOT NULL, "protocol" VARCHAR(255) NOT NULL, "source_filename" VARCHAR(255) NOT NULL, "dest_filename" VARCHAR(255) NOT NULL, "rule" VARCHAR(255) NOT NULL, "start" timestamp with time zone NOT NULL, "stop" timestamp with time zone NOT NULL, "status" VARCHAR(50) NOT NULL, "error_code" VARCHAR(50) NOT NULL, "error_details" VARCHAR(255) NOT NULL, "step" VARCHAR(50) NOT NULL, "progression" BIGINT NOT NULL, "task_number" BIGINT NOT NULL);
CREATE UNIQUE INDEX "UQE_transfer_history_histRemID" ON "transfer_history" ("remote_transfer_id","account","agent");
CREATE TABLE IF NOT EXISTS "local_accounts" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "local_agent_id" BIGINT NOT NULL, "login" VARCHAR(255) NOT NULL, "password" BYTEA NULL);
CREATE UNIQUE INDEX "UQE_local_accounts_loc_ac" ON "local_accounts" ("local_agent_id","login");
CREATE TABLE IF NOT EXISTS "local_agents" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "owner" VARCHAR(255) NOT NULL, "name" VARCHAR(255) NOT NULL, "protocol" VARCHAR(255) NOT NULL, "root" VARCHAR(255) NOT NULL, "in_dir" VARCHAR(255) NOT NULL, "out_dir" VARCHAR(255) NOT NULL, "work_dir" VARCHAR(255) NOT NULL, "proto_config" BYTEA NOT NULL, "address" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_local_agents_loc_ag" ON "local_agents" ("owner","name");
CREATE TABLE IF NOT EXISTS "remote_accounts" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "remote_agent_id" BIGINT NOT NULL, "login" VARCHAR(255) NOT NULL, "password" BYTEA NULL);
CREATE UNIQUE INDEX "UQE_remote_accounts_rem_ac" ON "remote_accounts" ("remote_agent_id","login");
CREATE TABLE IF NOT EXISTS "remote_agents" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "name" VARCHAR(255) NOT NULL, "protocol" VARCHAR(255) NOT NULL, "proto_config" BYTEA NOT NULL, "address" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_remote_agents_name" ON "remote_agents" ("name");
CREATE TABLE IF NOT EXISTS "rules" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "name" VARCHAR(255) NOT NULL, "comment" VARCHAR(255) NOT NULL, "send" BOOL NOT NULL, "path" VARCHAR(255) NOT NULL, "in_path" VARCHAR(255) NOT NULL, "out_path" VARCHAR(255) NOT NULL, "work_path" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_rules_dir" ON "rules" ("name","send");
CREATE UNIQUE INDEX "UQE_rules_path" ON "rules" ("send","path");
CREATE TABLE IF NOT EXISTS "rule_access" ("rule_id" BIGINT NOT NULL, "object_id" BIGINT NOT NULL, "object_type" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_rule_access_perm" ON "rule_access" ("rule_id","object_id","object_type");
CREATE TABLE IF NOT EXISTS "tasks" ("rule_id" BIGINT NOT NULL, "chain" VARCHAR(255) NOT NULL, "rank" INTEGER NOT NULL, "type" VARCHAR(255) NOT NULL, "args" BYTEA NOT NULL);
CREATE TABLE IF NOT EXISTS "transfers" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "remote_transfer_id" VARCHAR(255) NULL, "rule_id" BIGINT NOT NULL, "is_server" BOOL NOT NULL, "agent_id" BIGINT NOT NULL, "account_id" BIGINT NOT NULL, "true_filepath" VARCHAR(255) NOT NULL, "source_file" VARCHAR(255) NOT NULL, "dest_file" VARCHAR(255) NOT NULL, "start" timestamp with time zone NOT NULL, "step" VARCHAR(50) NOT NULL, "status" VARCHAR(255) NOT NULL, "owner" VARCHAR(255) NOT NULL, "progression" BIGINT NOT NULL, "task_number" BIGINT NOT NULL, "error_code" VARCHAR(50) NOT NULL, "error_details" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_transfers_transRemID" ON "transfers" ("remote_transfer_id","account_id");
CREATE TABLE IF NOT EXISTS "transfer_info" ("transfer_id" BIGINT NOT NULL, "name" VARCHAR(255) NOT NULL, "value" VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX "UQE_transfer_info_infoName" ON "transfer_info" ("transfer_id","name");
CREATE TABLE IF NOT EXISTS "users" ("id" BIGSERIAL PRIMARY KEY  NOT NULL, "owner" VARCHAR(255) NOT NULL, "username" VARCHAR(255) NOT NULL, "password" BYTEA NOT NULL, "permissions" BYTEA NOT NULL);
CREATE UNIQUE INDEX "UQE_users_name" ON "users" ("owner","username");
INSERT INTO "users" ("owner","username","password","permissions") VALUES ('test_gateway', 'admin', '\x243261243034244E7052683552343875795038504E336A73524C4C5775773734776A54514E394972715A70596A354E384A594D2E7268597877727665', '\xDFFFFFFF');
`

const mysqlCreationScript = `
CREATE TABLE IF NOT EXISTS version (current TEXT NOT NULL);
INSERT INTO version (current) VALUES ('0.0.0');
CREATE TABLE IF NOT EXISTS certificates (id BIGINT(20) PRIMARY KEY AUTO_INCREMENT NOT NULL, owner_type VARCHAR(255) NOT NULL, owner_id BIGINT(20) NOT NULL, name VARCHAR(255) NOT NULL, private_key BLOB NULL, public_key BLOB NULL, cert BLOB NULL);
CREATE UNIQUE INDEX UQE_certificates_cert ON certificates (owner_type,owner_id,name);
CREATE TABLE IF NOT EXISTS transfer_history (id BIGINT(20) PRIMARY KEY NOT NULL, owner VARCHAR(255) NOT NULL, remote_transfer_id VARCHAR(255) NULL, is_server TINYINT(1) NOT NULL, is_send TINYINT(1) NOT NULL, account VARCHAR(255) NOT NULL, agent VARCHAR(255) NOT NULL, protocol VARCHAR(255) NOT NULL, source_filename VARCHAR(255) NOT NULL, dest_filename VARCHAR(255) NOT NULL, rule VARCHAR(255) NOT NULL, start CHAR(64) NOT NULL, stop CHAR(64) NOT NULL, status VARCHAR(50) NOT NULL, error_code VARCHAR(50) NOT NULL, error_details VARCHAR(255) NOT NULL, step VARCHAR(50) NOT NULL, progression BIGINT(20) NOT NULL, task_number BIGINT(20) NOT NULL);
CREATE UNIQUE INDEX UQE_transfer_history_histRemID ON transfer_history (remote_transfer_id,account,agent);
CREATE TABLE IF NOT EXISTS local_accounts (id BIGINT(20) PRIMARY KEY AUTO_INCREMENT NOT NULL, local_agent_id BIGINT(20) NOT NULL, login VARCHAR(255) NOT NULL, password BLOB NULL);
CREATE UNIQUE INDEX UQE_local_accounts_loc_ac ON local_accounts (local_agent_id,login);
CREATE TABLE IF NOT EXISTS local_agents (id BIGINT(20) PRIMARY KEY AUTO_INCREMENT NOT NULL, owner VARCHAR(255) NOT NULL, name VARCHAR(255) NOT NULL, protocol VARCHAR(255) NOT NULL, root VARCHAR(255) NOT NULL, in_dir VARCHAR(255) NOT NULL, out_dir VARCHAR(255) NOT NULL, work_dir VARCHAR(255) NOT NULL, proto_config BLOB NOT NULL, address VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_local_agents_loc_ag ON local_agents (owner,name);
CREATE TABLE IF NOT EXISTS remote_accounts (id BIGINT(20) PRIMARY KEY AUTO_INCREMENT NOT NULL, remote_agent_id BIGINT(20) NOT NULL, login VARCHAR(255) NOT NULL, password BLOB NULL);
CREATE UNIQUE INDEX UQE_remote_accounts_rem_ac ON remote_accounts (remote_agent_id,login);
CREATE TABLE IF NOT EXISTS remote_agents (id BIGINT(20) PRIMARY KEY AUTO_INCREMENT NOT NULL, name VARCHAR(255) NOT NULL, protocol VARCHAR(255) NOT NULL, proto_config BLOB NOT NULL, address VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_remote_agents_name ON remote_agents (name);
CREATE TABLE IF NOT EXISTS rules (id BIGINT(20) PRIMARY KEY AUTO_INCREMENT NOT NULL, name VARCHAR(255) NOT NULL, comment VARCHAR(255) NOT NULL, send TINYINT(1) NOT NULL, path VARCHAR(255) NOT NULL, in_path VARCHAR(255) NOT NULL, out_path VARCHAR(255) NOT NULL, work_path VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_rules_dir ON rules (name,send);
CREATE UNIQUE INDEX UQE_rules_path ON rules (send,path);
CREATE TABLE IF NOT EXISTS rule_access (rule_id BIGINT(20) NOT NULL, object_id BIGINT(20) NOT NULL, object_type VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_rule_access_perm ON rule_access (rule_id,object_id,object_type);
CREATE TABLE IF NOT EXISTS tasks (rule_id BIGINT(20) NOT NULL, chain VARCHAR(255) NOT NULL, rank INT NOT NULL, type VARCHAR(255) NOT NULL, args BLOB NOT NULL);
CREATE TABLE IF NOT EXISTS transfers (id BIGINT(20) PRIMARY KEY AUTO_INCREMENT NOT NULL, remote_transfer_id VARCHAR(255) NULL, rule_id BIGINT(20) NOT NULL, is_server TINYINT(1) NOT NULL, agent_id BIGINT(20) NOT NULL, account_id BIGINT(20) NOT NULL, true_filepath VARCHAR(255) NOT NULL, source_file VARCHAR(255) NOT NULL, dest_file VARCHAR(255) NOT NULL, start CHAR(64) NOT NULL, step VARCHAR(50) NOT NULL, status VARCHAR(255) NOT NULL, owner VARCHAR(255) NOT NULL, progression BIGINT(20) NOT NULL, task_number BIGINT(20) NOT NULL, error_code VARCHAR(50) NOT NULL, error_details VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_transfers_transRemID ON transfers (remote_transfer_id,account_id);
CREATE TABLE IF NOT EXISTS transfer_info (transfer_id BIGINT(20) NOT NULL, name VARCHAR(255) NOT NULL, value VARCHAR(255) NOT NULL);
CREATE UNIQUE INDEX UQE_transfer_info_infoName ON transfer_info (transfer_id,name);
CREATE TABLE IF NOT EXISTS users (id BIGINT(20) PRIMARY KEY AUTO_INCREMENT NOT NULL, owner VARCHAR(255) NOT NULL, username VARCHAR(255) NOT NULL, password BLOB NOT NULL, permissions BINARY(4) NOT NULL);
CREATE UNIQUE INDEX UQE_users_name ON users (owner,username);
INSERT INTO users (owner,username,password,permissions) VALUES ('test_gateway', 'admin', X'243261243034244E7052683552343875795038504E336A73524C4C5775773734776A54514E394972715A70596A354E384A594D2E7268597877727665', X'DFFFFFFF');
`
