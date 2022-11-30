package backup

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	. "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestImportLocalAgents(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a database with some local agent", func() {
			agent := &model.LocalAgent{
				Name:     "server",
				Protocol: testProtocol,
				Address:  "localhost:2022",
			}

			// add another LocalAgent with the same name but different owner
			agent2 := &model.LocalAgent{
				Name:     agent.Name,
				Protocol: testProtocol,
				Address:  "localhost:9999",
			}
			owner := conf.GlobalConfig.GatewayName
			conf.GlobalConfig.GatewayName = "toto"
			So(db.Insert(agent2).Run(), ShouldBeNil)
			conf.GlobalConfig.GatewayName = owner

			So(db.Insert(agent).Run(), ShouldBeNil)

			other := &model.LocalAgent{
				Name:     "other",
				Protocol: testProtocol,
				Address:  "localhost:8888",
			}
			So(db.Insert(other).Run(), ShouldBeNil)

			Convey("Given a list of new agents", func() {
				newServer := LocalAgent{
					Name:          "foo",
					Protocol:      testProtocol,
					Configuration: json.RawMessage(`{}`),
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
					Configuration: json.RawMessage(`{}`),
					Address:       "localhost:6666",
					Accounts: []LocalAccount{
						{
							Login:    "toto",
							Password: "pwd",
						},
					},
					Certs: []Certificate{
						{
							Name:        "cert",
							PrivateKey:  testhelpers.LocalhostKey,
							Certificate: testhelpers.LocalhostCert,
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

							So(len(accounts), ShouldEqual, 1)

							var cryptos model.Cryptos
							So(db.Select(&cryptos).Where("local_agent_id=?",
								dbAgent.ID).Run(), ShouldBeNil)

							So(len(accounts), ShouldEqual, 1)
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
				Name:     "server",
				Protocol: testProtocol,
				Address:  "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			dbAccount := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "foo",
				PasswordHash: hash("bar"),
			}
			So(db.Insert(dbAccount).Run(), ShouldBeNil)

			Convey("Given a list of new accounts", func() {
				account1 := LocalAccount{
					Login:    "toto",
					Password: "pwd",
				}
				account2 := LocalAccount{
					Login:    "tata",
					Password: "pwd",
				}
				accounts := []LocalAccount{
					account1, account2,
				}

				Convey("When calling the importLocalAccounts method", func() {
					err := importLocalAccounts(discard(), db, accounts, agent)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})
					Convey("Then the database should contains the local "+
						"accounts", func() {
						var accounts model.LocalAccounts
						So(db.Select(&accounts).Where("local_agent_id=?",
							agent.ID).Run(), ShouldBeNil)

						So(len(accounts), ShouldEqual, 3)

						Convey("Then the data should correspond to the "+
							"one imported", func() {
							for i := 0; i < len(accounts); i++ {
								switch {
								case accounts[i].Login == account1.Login:
									Convey("Then account1 is found", func() {
										So(bcrypt.CompareHashAndPassword(
											[]byte(accounts[i].PasswordHash),
											[]byte("pwd")), ShouldBeNil)
									})
								case accounts[i].Login == account2.Login:
									Convey("Then account2 is found", func() {
										So(bcrypt.CompareHashAndPassword(
											[]byte(accounts[i].PasswordHash),
											[]byte("pwd")), ShouldBeNil)
									})
								case accounts[i].Login == dbAccount.Login:
									Convey("Then dbAccount is found", func() {
										So(accounts[i].PasswordHash, ShouldResemble,
											dbAccount.PasswordHash)
									})
								default:
									Convey("Then they should be no other "+
										"records", func() {
										So(1, ShouldBeNil)
									})
								}
							}
						})
					})
				})
			})

			Convey("Given a list of fully updated agents", func() {
				account1 := LocalAccount{
					Login:    "foo",
					Password: "notbar",
					Certs: []Certificate{
						{
							Name:        "cert",
							Certificate: testhelpers.ClientFooCert,
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

						So(len(accounts), ShouldEqual, 1)

						Convey("Then the data should correspond to the "+
							"one imported", func() {
							for i := 0; i < len(accounts); i++ {
								if accounts[i].Login == dbAccount.Login {
									Convey("When dbAccount is found", func() {
										So(accounts[i].PasswordHash, ShouldNotResemble,
											dbAccount.PasswordHash)
										var cryptos model.Cryptos
										So(db.Select(&cryptos).Where("local_account_id=?",
											dbAccount.ID).Run(), ShouldBeNil)

										So(len(accounts), ShouldEqual, 1)
									})
								} else {
									Convey("Then they should be no "+
										"other records", func() {
										So(1, ShouldBeNil)
									})
								}
							}
						})
					})
				})
			})

			Convey("Given a list of partially updated agents", func() {
				account1 := LocalAccount{
					Login: "foo",
					Certs: []Certificate{
						{
							Name:        "cert",
							Certificate: testhelpers.ClientFooCert,
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

						So(len(accounts), ShouldEqual, 1)

						Convey("Then the data should correspond to the "+
							"one imported", func() {
							for i := 0; i < len(accounts); i++ {
								if accounts[i].Login == dbAccount.Login {
									Convey("When dbAccount is found", func() {
										So(accounts[i].PasswordHash, ShouldResemble,
											dbAccount.PasswordHash)
										var cryptos model.Cryptos
										So(db.Select(&cryptos).Where("local_account_id=?",
											dbAccount.ID).Run(), ShouldBeNil)

										So(len(accounts), ShouldEqual, 1)
									})
								} else {
									Convey("Then they should be no "+
										"other records", func() {
										So(1, ShouldBeNil)
									})
								}
							}
						})
					})
				})
			})
		})
	})
}
