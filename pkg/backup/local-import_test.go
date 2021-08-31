package backup

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

func TestImportLocalAgents(t *testing.T) {

	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a database with some local agent", func() {
			agent := &model.LocalAgent{
				Name:        "server",
				Protocol:    config.TestProtocol,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			Convey("Given a list of new agents", func() {
				agent1 := LocalAgent{
					Name:          "foo",
					Protocol:      config.TestProtocol,
					Configuration: json.RawMessage(`{}`),
					Address:       "localhost:2022",
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
				agents := []LocalAgent{agent1}

				Convey("Given an empty database", func() {

					Convey("When calling the importLocals method", func() {
						err := importLocalAgents(discard, db, agents)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the local agents", func() {
							var dbAgent model.LocalAgent
							So(db.Get(&dbAgent, "name=?", agent1.Name).Run(), ShouldBeNil)

							Convey("Then the data shuld correspond to the "+
								"one imported", func() {
								So(dbAgent.Name, ShouldEqual, agent1.Name)
								So(dbAgent.Protocol, ShouldEqual, agent1.Protocol)
								So(dbAgent.ProtoConfig, ShouldResemble,
									agent1.Configuration)

								var accounts model.LocalAccounts
								So(db.Select(&accounts).Where("local_agent_id=?",
									dbAgent.ID).Run(), ShouldBeNil)

								So(len(accounts), ShouldEqual, 2)
							})
						})
					})
				})
			})

			Convey("Given a list of fully updated agents", func() {
				agent1 := LocalAgent{
					Name:          "server",
					Protocol:      config.TestProtocol,
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
					err := importLocalAgents(discard, db, agents)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})
					Convey("Then the database should contains the local agents", func() {
						var dbAgent model.LocalAgent
						So(db.Get(&dbAgent, "name=?", agent1.Name).Run(), ShouldBeNil)

						Convey("Then the data shuld correspond to the "+
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
							So(db.Select(&cryptos).Where("owner_type=? AND owner_id=?",
								model.TableLocAgents, dbAgent.ID).Run(), ShouldBeNil)

							So(len(accounts), ShouldEqual, 1)
						})
					})
				})
			})
		})
	})
}

func TestImportLocalAccounts(t *testing.T) {

	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a database with some a local agent and some local accounts", func() {
			agent := &model.LocalAgent{
				Name:        "server",
				Protocol:    config.TestProtocol,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
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
					err := importLocalAccounts(discard, db, accounts, agent)

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
								if accounts[i].Login == account1.Login {

									Convey("Then account1 is found", func() {
										So(bcrypt.CompareHashAndPassword(
											accounts[i].PasswordHash, []byte("pwd")),
											ShouldBeNil)
									})
								} else if accounts[i].Login == account2.Login {

									Convey("Then account2 is found", func() {
										So(bcrypt.CompareHashAndPassword(
											accounts[i].PasswordHash, []byte("pwd")),
											ShouldBeNil)
									})
								} else if accounts[i].Login == dbAccount.Login {

									Convey("Then dbAccount is found", func() {
										So(accounts[i].PasswordHash, ShouldResemble,
											dbAccount.PasswordHash)
									})
								} else {
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
					err := importLocalAccounts(discard, db, accounts, agent)

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
										So(db.Select(&cryptos).Where(
											"owner_type=? AND owner_id=?",
											model.TableLocAccounts, dbAccount.ID).
											Run(), ShouldBeNil)

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
					err := importLocalAccounts(discard, db, accounts, agent)

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
										So(db.Select(&cryptos).Where(
											"owner_type=? AND owner_id=?",
											dbAccount.ID, model.TableLocAccounts).
											Run(), ShouldBeNil)

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
