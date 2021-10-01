package migrations

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/tk/migration"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

// func tempFilename() string {
// 	f, err := ioutil.TempFile(os.TempDir(), "test_migration_database_*.db")
// 	So(err, ShouldBeNil)
// 	So(f.Close(), ShouldBeNil)
// 	So(os.Remove(f.Name()), ShouldBeNil)
//
// 	return f.Name()
// }

//nolint:lll // this won't change a lot, readability might not be that important
const sqliteCreationScript = `
CREATE TABLE IF NOT EXISTS 'version' ('current' TEXT NOT NULL);
INSERT INTO 'version' ('current') VALUES ('0.0.0');
CREATE TABLE IF NOT EXISTS 'certificates' ('id' INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 'owner_type' TEXT NOT NULL, 'owner_id' INTEGER NOT NULL, 'name' TEXT NOT NULL, 'private_key' BLOB NULL, 'public_key' BLOB NULL, 'cert' BLOB NULL);
CREATE UNIQUE INDEX 'UQE_certificates_cert' ON 'certificates' ('owner_type','owner_id','name');
CREATE TABLE IF NOT EXISTS 'transfer_history' ('id' INTEGER PRIMARY KEY NOT NULL, 'owner' TEXT NOT NULL, 'remote_transfer_id' TEXT NULL, 'is_server' INTEGER NOT NULL, 'is_send' INTEGER NOT NULL, 'account' TEXT NOT NULL, 'agent' TEXT NOT NULL, 'protocol' TEXT NOT NULL, 'source_filename' TEXT NOT NULL, 'dest_filename' TEXT NOT NULL, 'rule' TEXT NOT NULL, 'start' TEXT NOT NULL, 'stop' TEXT, 'status' TEXT NOT NULL, 'error_code' TEXT NOT NULL, 'error_details' TEXT NOT NULL, 'step' TEXT NOT NULL, 'progression' INTEGER NOT NULL, 'task_number' INTEGER NOT NULL);
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
INSERT INTO 'users' ('owner','username','password','permissions') VALUES ('test_gateway', 'admin', X'243261243034244E7052683552343875795038504E336A73524C4C5775773734776A54514E394972715A70596A354E384A594D2E7268597877727665', X'DFFFFFFF');`

func getSQLiteEngine(c C) *migration.Engine {
	db := testhelpers.GetTestSqliteDB(c)

	_, err := db.Exec(sqliteCreationScript)
	So(err, ShouldBeNil)

	eng, err := migration.NewEngine(db, migration.SQLite, nil)
	So(err, ShouldBeNil)

	return eng
}

func TestSQLiteCreationScript(t *testing.T) {
	Convey("Given a SQLite database", t, func(c C) {
		db := testhelpers.GetTestSqliteDB(c)

		Convey("Given the script to initialize version 0.0.0 of the database", func() {
			Convey("When executing the script", func() {
				_, err := db.Exec(sqliteCreationScript)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})
	})
}
