package backup

import (
	"testing"

	r66lib "code.waarp.fr/lib/r66"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	. "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestImportLocalAgents(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a database with some local agent", func() {
			agent := &model.LocalAgent{
				Name: "server", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}

			// add another LocalAgent with the same name but different owner
			agent2 := &model.LocalAgent{
				Name: agent.Name, Protocol: testProtocol,
				Address: types.Addr("localhost", 9999),
			}
			owner := conf.GlobalConfig.GatewayName
			conf.GlobalConfig.GatewayName = "toto"

			So(db.Insert(agent2).Run(), ShouldBeNil)

			conf.GlobalConfig.GatewayName = owner

			So(db.Insert(agent).Run(), ShouldBeNil)

			other := &model.LocalAgent{
				Name: "other", Protocol: testProtocol,
				Address: types.Addr("localhost", 8888),
			}
			So(db.Insert(other).Run(), ShouldBeNil)

			Convey("Given a list of new agents", func() {
				newServer := LocalAgent{
					Name:          "foo",
					Protocol:      testProtocol,
					Configuration: map[string]any{},
					Address:       "localhost:1111",
					Accounts: []LocalAccount{
						{
							Login:    "acc1",
							Password: "pwd",
						}, {
							Login:    "acc2",
							Password: "pwd",
						},
					},
				}
				newServers := []LocalAgent{newServer}

				Convey("When calling the importLocals method", func() {
					err := importLocalAgents(discard(), db, newServers, false)
					So(err, ShouldBeNil)

					var dbAgents model.LocalAgents
					So(db.Select(&dbAgents).Where("owner=?", conf.GlobalConfig.GatewayName).
						OrderBy("id", true).Run(),
						ShouldBeNil)
					So(dbAgents, ShouldHaveLength, 3)

					Convey("Then the new agent should have been imported", func() {
						dbAgent := dbAgents[2]

						So(dbAgent.Name, ShouldEqual, newServer.Name)
						So(dbAgent.Protocol, ShouldEqual, newServer.Protocol)
						So(dbAgent.ProtoConfig, ShouldResemble,
							newServer.Configuration)

						Convey("Then the local accounts should have been imported", func() {
							var accounts model.LocalAccounts
							So(db.Select(&accounts).Where("local_agent_id=?",
								dbAgent.ID).Run(), ShouldBeNil)
							So(accounts, ShouldHaveLength, 2)

							So(accounts[0].Login, ShouldEqual, newServer.Accounts[0].Login)
							So(accounts[1].Login, ShouldEqual, newServer.Accounts[1].Login)
						})
					})

					Convey("Then the other local agents should be unchanged", func() {
						So(dbAgents[0], ShouldResemble, agent)
						So(dbAgents[1], ShouldResemble, other)
					})
				})

				Convey("When calling the importLocals method with reset ON", func() {
					err := importLocalAgents(discard(), db, newServers, true)
					So(err, ShouldBeNil)

					var dbAgents model.LocalAgents
					So(db.Select(&dbAgents).Where("owner=?", conf.GlobalConfig.GatewayName).
						OrderBy("id", true).Run(),
						ShouldBeNil)
					So(dbAgents, ShouldHaveLength, 1)

					Convey("Then only the imported agent should be left", func() {
						dbAgent := dbAgents[0]

						So(dbAgent.Name, ShouldEqual, newServer.Name)
						So(dbAgent.Protocol, ShouldEqual, newServer.Protocol)
						So(dbAgent.ProtoConfig, ShouldResemble,
							newServer.Configuration)
					})
				})
			})

			Convey("Given a list of fully updated agents", func() {
				agent1 := LocalAgent{
					Name:          agent.Name,
					Protocol:      testProtocol,
					Configuration: map[string]any{},
					Address:       "localhost:6666",
					Accounts: []LocalAccount{
						{
							Login:    "toto",
							Password: "pwd",
						},
					},
					Credentials: []Credential{
						{
							Type:   auth.TLSCertificate,
							Value:  testhelpers.LocalhostCert,
							Value2: testhelpers.LocalhostKey,
						},
					},
				}
				agents := []LocalAgent{agent1}

				Convey("When calling the importLocals method", func() {
					var accs model.LocalAccounts
					So(db.Select(&accs).Run(), ShouldBeNil)

					err := importLocalAgents(discard(), db, agents, false)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the database should contain the local agents", func() {
						var dbAgents model.LocalAgents
						So(db.Select(&dbAgents).OrderBy("id", true).Run(), ShouldBeNil)
						So(dbAgents, ShouldHaveLength, 3)

						dbAgent := dbAgents[1]

						Convey("Then the data should correspond to the "+
							"one imported", func() {
							So(dbAgent.Name, ShouldEqual, agent1.Name)
							So(dbAgent.Protocol, ShouldEqual, agent1.Protocol)
							So(dbAgent.ProtoConfig, ShouldResemble,
								agent1.Configuration)

							var accounts model.LocalAccounts
							So(db.Select(&accounts).Where("local_agent_id=?",
								dbAgent.ID).Run(), ShouldBeNil)

							So(accounts, ShouldHaveLength, 1)

							var cryptos model.Credentials
							So(db.Select(&cryptos).Where("local_agent_id=?",
								dbAgent.ID).Run(), ShouldBeNil)

							So(accounts, ShouldHaveLength, 1)
						})

						Convey("Then the other agents should be unchanged", func() {
							So(dbAgents[2], ShouldResemble, other)
						})
					})
				})
			})
		})
	})
}

func TestImportLocalAccounts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a database with some a local agent and some local accounts", func() {
			agent := &model.LocalAgent{
				Name: "server", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			dbAccount := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "foo",
			}
			So(db.Insert(dbAccount).Run(), ShouldBeNil)

			accPswd := &model.Credential{
				LocalAccountID: utils.NewNullInt64(dbAccount.ID),
				Type:           auth.Password,
				Value:          "bar",
			}
			So(db.Insert(accPswd).Run(), ShouldBeNil)

			Convey("Given a list of new accounts", func() {
				account1 := LocalAccount{Login: "toto", Password: "pwd"}
				account2 := LocalAccount{Login: "tata", Password: "pwd"}
				accounts := []LocalAccount{account1, account2}
				So(preprocessLocalAccounts(accounts), ShouldBeNil)

				Convey("When calling the importLocalAccounts method", func() {
					err := importLocalAccounts(discard(), db, accounts, agent)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})
					Convey("Then the database should contains the local "+
						"accounts", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Where("local_agent_id=?",
							agent.ID).OrderBy("id", true).Run(), ShouldBeNil)

						So(accounts, ShouldHaveLength, 3)

						Convey("Then the exiting account should be unchanged", func() {
							So(accounts[0], ShouldResemble, dbAccount)
						})

						Convey("Then the 1st account should have been imported", func() {
							var pswd model.Credential
							So(db.Get(&pswd, "local_account_id=? AND type=?",
								accounts[1].ID, auth.Password).Run(), ShouldBeNil)

							So(accounts[1].Login, ShouldEqual, account1.Login)
							So(bcrypt.CompareHashAndPassword([]byte(pswd.Value),
								[]byte(account1.Password)), ShouldBeNil)
						})

						Convey("Then the 2nd account should have been imported", func() {
							var pswd model.Credential
							So(db.Get(&pswd, "local_account_id=? AND type=?",
								accounts[2].ID, auth.Password).Run(), ShouldBeNil)

							So(accounts[2].Login, ShouldEqual, account2.Login)
							So(bcrypt.CompareHashAndPassword([]byte(pswd.Value),
								[]byte(account2.Password)), ShouldBeNil)
						})
					})
				})
			})

			Convey("Given a list of fully updated agents", func() {
				account1 := LocalAccount{
					Login:    dbAccount.Login,
					Password: "notbar",
					Credentials: []Credential{
						{
							Type:  auth.TLSTrustedCertificate,
							Value: testhelpers.ClientFooCert,
						},
					},
				}
				accounts := []LocalAccount{account1}
				So(preprocessLocalAccounts(accounts), ShouldBeNil)

				Convey("When calling the importLocalAccounts method", func() {
					err := importLocalAccounts(discard(), db, accounts, agent)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})
					Convey("Then the database should contains the "+
						"local accounts", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Where("local_agent_id=?",
							agent.ID).Run(), ShouldBeNil)

						So(accounts, ShouldHaveLength, 1)

						Convey("Then the data should correspond to the "+
							"one imported", func() {
							So(accounts[0].Login, ShouldEqual, dbAccount.Login)

							var pswd model.Credential
							So(db.Get(&pswd, "local_account_id=? AND type=?",
								accounts[0].ID, auth.Password).Run(), ShouldBeNil)
							So(bcrypt.CompareHashAndPassword([]byte(pswd.Value),
								[]byte(account1.Password)), ShouldBeNil)

							var cert model.Credential
							So(db.Get(&cert, "local_account_id=? AND type=?",
								accounts[0].ID, auth.TLSTrustedCertificate).Run(), ShouldBeNil)
							So(cert.Value, ShouldEqual, account1.Credentials[0].Value)
						})
					})
				})
			})

			Convey("Given a list of partially updated agents", func() {
				account1 := LocalAccount{
					Login: "foo",
					Credentials: []Credential{
						{
							Type:  auth.TLSTrustedCertificate,
							Value: testhelpers.ClientFooCert,
						},
					},
				}
				accounts := []LocalAccount{account1}

				Convey("When calling the importLocalAccounts method", func() {
					err := importLocalAccounts(discard(), db, accounts, agent)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})
					Convey("Then the database should contains the "+
						"local accounts", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Where("local_agent_id=?",
							agent.ID).Run(), ShouldBeNil)

						So(accounts, ShouldHaveLength, 1)

						Convey("Then the data should correspond to the one imported", func() {
							So(accounts[0].Login, ShouldEqual, dbAccount.Login)

							var cert model.Credential
							So(db.Get(&cert, "local_account_id=? AND type=?",
								accounts[0].ID, auth.TLSTrustedCertificate).Run(), ShouldBeNil)
							So(cert.Value, ShouldEqual, account1.Credentials[0].Value)
						})
					})
				})
			})
		})
	})
}

func TestR66PasswordImport(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	server := &model.LocalAgent{
		Name:     "r66_server",
		Address:  types.Addr("localhost", 0),
		Protocol: r66TLS,
	}
	require.NoError(t, db.Insert(server).Run())

	const pswd = "bar"
	hashed, err := utils.HashPassword(bcrypt.MinCost, string(r66lib.CryptPass([]byte(pswd))))
	require.NoError(t, err)

	accounts := []LocalAccount{{
		Login:        "foo",
		Password:     pswd,
		PasswordHash: hashed,
	}}
	require.NoError(t, preprocessLocalAccounts(accounts))

	require.NoError(t, importLocalAccounts(logger, db, accounts, server))

	var cred model.Credential
	require.NoError(t, db.Get(&cred, "type=?", auth.Password).Run())

	assert.Equal(t, hashed, cred.Value)
}
