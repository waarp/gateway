package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	. "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestImportRemoteAgents(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a database with some remote agent", func() {
			agent := &model.RemoteAgent{
				Name:     "partner",
				Protocol: testProtocol,
				Address:  "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			other := &model.RemoteAgent{
				Name:     "other",
				Protocol: testProtocol,
				Address:  "localhost:8888",
			}
			So(db.Insert(other).Run(), ShouldBeNil)

			Convey("Given a list of new agents", func() {
				newPartner := RemoteAgent{
					Name:          "foo",
					Protocol:      testProtocol,
					Configuration: map[string]any{},
					Address:       "localhost:2022",
					Accounts: []RemoteAccount{
						{
							Login:    "acc1",
							Password: "pwd",
						}, {
							Login:    "acc2",
							Password: "pwd",
						},
					},
				}
				newPartners := []RemoteAgent{newPartner}

				Convey("When calling the importRemotes method", func() {
					err := importRemoteAgents(discard(), db, newPartners, false)
					So(err, ShouldBeNil)

					var dbAgents model.RemoteAgents
					So(db.Select(&dbAgents).OrderBy("id", true).Run(), ShouldBeNil)
					So(dbAgents, ShouldHaveLength, 3)

					Convey("Then the new agent should have been imported", func() {
						dbAgent := dbAgents[2]

						So(dbAgent.Name, ShouldEqual, newPartner.Name)
						So(dbAgent.Protocol, ShouldEqual, newPartner.Protocol)
						So(dbAgent.ProtoConfig, ShouldResemble,
							newPartner.Configuration)

						Convey("Then the new accounts should have been imported", func() {
							var accounts model.RemoteAccounts
							So(db.Select(&accounts).Where("remote_agent_id=?",
								dbAgent.ID).Run(), ShouldBeNil)

							So(len(accounts), ShouldEqual, 2)

							So(accounts[0].Login, ShouldEqual, newPartner.Accounts[0].Login)
							So(accounts[1].Login, ShouldEqual, newPartner.Accounts[1].Login)
						})
					})

					Convey("Then the other agents should be unchanged", func() {
						So(dbAgents[0], ShouldResemble, agent)
						So(dbAgents[1], ShouldResemble, other)
					})
				})

				Convey("When calling the importRemotes method with reset ON", func() {
					err := importRemoteAgents(discard(), db, newPartners, true)
					So(err, ShouldBeNil)

					var dbAgents model.RemoteAgents
					So(db.Select(&dbAgents).OrderBy("id", true).Run(), ShouldBeNil)
					So(dbAgents, ShouldHaveLength, 1)

					Convey("Then only the imported agent should be left", func() {
						dbAgent := dbAgents[0]

						So(dbAgent.Name, ShouldEqual, newPartner.Name)
						So(dbAgent.Protocol, ShouldEqual, newPartner.Protocol)
						So(dbAgent.ProtoConfig, ShouldResemble,
							newPartner.Configuration)
					})
				})
			})
		})

		Convey("Given a list of fully updated agents", func() {
			agent1 := RemoteAgent{
				Name:          "agent1",
				Protocol:      testProtocol,
				Configuration: map[string]any{},
				Address:       "localhost:6666",
				Accounts: []RemoteAccount{
					{
						Login:    "acc1",
						Password: "pwd",
					},
				},
				Certs: []Certificate{
					{
						Name:        "cert",
						Certificate: testhelpers.LocalhostCert,
					},
				},
			}
			agents := []RemoteAgent{agent1}

			Convey("When calling the importRemotes method", func() {
				err := importRemoteAgents(discard(), db, agents, false)

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})
				Convey("Then the database should contains the remote agents", func() {
					var dbAgents model.RemoteAgents
					So(db.Select(&dbAgents).Run(), ShouldBeNil)
					So(dbAgents, ShouldHaveLength, 1)

					dbAgent := dbAgents[0]

					Convey("Then the data should correspond to the "+
						"one imported", func() {
						So(dbAgent.Name, ShouldEqual, agent1.Name)
						So(dbAgent.Protocol, ShouldEqual, agent1.Protocol)
						So(dbAgent.ProtoConfig, ShouldResemble,
							agent1.Configuration)

						var accounts model.RemoteAccounts
						So(db.Select(&accounts).Where("remote_agent_id=?",
							dbAgent.ID).Run(), ShouldBeNil)

						So(len(accounts), ShouldEqual, 1)

						var cryptos model.Cryptos
						So(db.Select(&cryptos).Where("remote_agent_id=?",
							dbAgent.ID).Run(), ShouldBeNil)

						So(len(accounts), ShouldEqual, 1)
					})
				})
			})
		})
	})
}

func TestImportRemoteAccounts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a database with some a remote agent and some remote accounts", func() {
			agent := &model.RemoteAgent{
				Name:     "partner",
				Protocol: testProtocol,
				Address:  "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			dbAccount := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "foo",
				Password:      "bar",
			}
			So(db.Insert(dbAccount).Run(), ShouldBeNil)

			Convey("Given a list of new accounts", func() {
				account1 := RemoteAccount{
					Login:    "acc1",
					Password: "pwd",
				}
				account2 := RemoteAccount{
					Login:    "acc2",
					Password: "pwd",
				}
				accounts := []RemoteAccount{
					account1, account2,
				}

				Convey("When calling the importRemoteAccounts method", func() {
					err := importRemoteAccounts(discard(), db, accounts, agent)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the database should contains the "+
						"remote accounts", func() {
						var accounts model.RemoteAccounts
						So(db.Select(&accounts).Where("remote_agent_id=?",
							agent.ID).Run(), ShouldBeNil)

						So(len(accounts), ShouldEqual, 3)

						Convey("Then the data should correspond to "+
							"the one imported", func() {
							for i := 0; i < len(accounts); i++ {
								switch {
								case accounts[i].Login == account1.Login:
									Convey("Then account1 is found", func() {
										So(accounts[i].Login, ShouldResemble, account1.Login)
										So(string(accounts[i].Password), ShouldEqual, account1.Password)
									})
								case accounts[i].Login == account2.Login:
									Convey("Then account2 is found", func() {
										So(accounts[i].Login, ShouldResemble, account2.Login)
										So(string(accounts[i].Password), ShouldEqual, account2.Password)
									})
								case accounts[i].Login == dbAccount.Login:
									Convey("Then dbAccount is found", func() {
										So(accounts[i], ShouldResemble, dbAccount)
									})
								default:
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

			Convey("Given a list of fully updated agents", func() {
				account1 := RemoteAccount{
					Login:    "foo",
					Password: "notbar",
					Certs: []Certificate{
						{
							Name:        "cert",
							PrivateKey:  testhelpers.ClientFooKey,
							Certificate: testhelpers.ClientFooCert,
						},
					},
				}
				accounts := []RemoteAccount{account1}

				Convey("When calling the importRemoteAccounts method", func() {
					err := importRemoteAccounts(discard(), db, accounts, agent)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})
					Convey("Then the database should contains the "+
						"remote accounts", func() {
						var accounts model.RemoteAccounts
						So(db.Select(&accounts).Where("remote_agent_id=?",
							agent.ID).Run(), ShouldBeNil)

						So(len(accounts), ShouldEqual, 1)

						Convey("Then the data should correspond to "+
							"the one imported", func() {
							for i := 0; i < len(accounts); i++ {
								if accounts[i].Login == dbAccount.Login {
									Convey("When dbAccount is found", func() {
										So(accounts[i].Password, ShouldNotResemble,
											dbAccount.Password)
										var cryptos model.Cryptos
										So(db.Select(&cryptos).Where("remote_account_id=?",
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
				account1 := RemoteAccount{
					Login: "foo",
					Certs: []Certificate{
						{
							Name:        "cert",
							PrivateKey:  testhelpers.ClientFooKey,
							Certificate: testhelpers.ClientFooCert,
						},
					},
				}
				accounts := []RemoteAccount{account1}

				Convey("When calling the importRemoteAccounts method", func() {
					err := importRemoteAccounts(discard(), db, accounts, agent)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})
					Convey("Then the database should contains the "+
						"remote accounts", func() {
						var accounts model.RemoteAccounts
						So(db.Select(&accounts).Where("remote_agent_id=?",
							agent.ID).Run(), ShouldBeNil)

						So(len(accounts), ShouldEqual, 1)

						Convey("Then the data should correspond to "+
							"the one imported", func() {
							for i := 0; i < len(accounts); i++ {
								if accounts[i].Login == dbAccount.Login {
									Convey("When dbAccount is found", func() {
										So(accounts[i].Password, ShouldResemble,
											dbAccount.Password)
										var cryptos model.Cryptos
										So(db.Select(&cryptos).Where("remote_account_id=?",
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
