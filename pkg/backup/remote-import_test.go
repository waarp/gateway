package backup

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestImportRemoteAgents(t *testing.T) {

	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a database with some remote agent", func() {
			agent := &model.RemoteAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			Convey("Given a list of new agents", func() {
				agent1 := RemoteAgent{
					Name:          "foo",
					Protocol:      "sftp",
					Configuration: []byte(`{}`),
					Address:       "localhost:2022",
					Accounts: []RemoteAccount{
						{
							Login:    "test",
							Password: "pwd",
						}, {
							Login:    "test2",
							Password: "pwd",
						},
					},
				}
				agents := []RemoteAgent{agent1}

				Convey("When calling the importRemotes method", func() {
					err := importRemoteAgents(discard, db, agents)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})
					Convey("Then the database should contains the remote agents", func() {
						var dbAgent model.RemoteAgent
						So(db.Get(&dbAgent, "name=?", agent1.Name).Run(), ShouldBeNil)

						Convey("Then the data should correspond to the "+
							"one imported", func() {
							So(dbAgent.Name, ShouldEqual, agent1.Name)
							So(dbAgent.Protocol, ShouldEqual, agent1.Protocol)
							So(dbAgent.ProtoConfig, ShouldResemble,
								agent1.Configuration)

							var accounts model.RemoteAccounts
							So(db.Select(&accounts).Where("remote_agent_id=?",
								dbAgent.ID).Run(), ShouldBeNil)

							So(len(accounts), ShouldEqual, 2)
						})
					})
				})
			})
		})

		Convey("Given a list of fully updated agents", func() {
			agent1 := RemoteAgent{
				Name:          "test",
				Protocol:      "sftp",
				Configuration: []byte(`{}`),
				Address:       "localhost:6666",
				Accounts: []RemoteAccount{
					{
						Login:    "test",
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
				err := importRemoteAgents(discard, db, agents)

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})
				Convey("Then the database should contains the remote agents", func() {
					var dbAgent model.RemoteAgent
					So(db.Get(&dbAgent, "name=?", agent1.Name).Run(), ShouldBeNil)

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
						So(db.Select(&cryptos).Where("owner_type=? AND owner_id=?",
							model.TableRemAgents, dbAgent.ID).Run(), ShouldBeNil)

						So(len(accounts), ShouldEqual, 1)
					})
				})
			})
		})
	})
}

func TestImportRemoteAccounts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a database with some a remote agent and some remote accounts", func() {
			agent := &model.RemoteAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
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
					Login:    "test",
					Password: "pwd",
				}
				account2 := RemoteAccount{
					Login:    "test2",
					Password: "pwd",
				}
				accounts := []RemoteAccount{
					account1, account2,
				}

				Convey("When calling the importRemoteAccounts method", func() {
					err := importRemoteAccounts(discard, db, accounts, agent.ID)

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
								if accounts[i].Login == account1.Login {
									Convey("Then account1 is found", func() {
										So(accounts[i].Login, ShouldResemble, account1.Login)
										So(accounts[i].Password, ShouldEqual, account1.Password)
									})
								} else if accounts[i].Login == account2.Login {
									Convey("Then account2 is found", func() {
										So(accounts[i].Login, ShouldResemble, account2.Login)
										So(accounts[i].Password, ShouldEqual, account2.Password)
									})
								} else if accounts[i].Login == dbAccount.Login {
									Convey("Then dbAccount is found", func() {
										So(accounts[i], ShouldResemble, *dbAccount)
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

			Convey("Given a list of fully updated agents", func() {
				account1 := RemoteAccount{
					Login:    "foo",
					Password: "notbar",
					Certs: []Certificate{
						{
							Name:        "cert",
							PrivateKey:  testhelpers.ClientKey,
							Certificate: testhelpers.ClientCert,
						},
					},
				}
				accounts := []RemoteAccount{account1}

				Convey("When calling the importRemoteAccounts method", func() {
					err := importRemoteAccounts(discard, db, accounts, agent.ID)

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
										So(db.Select(&cryptos).Where(
											"owner_type=? AND owner_id=?",
											model.TableRemAccounts, dbAccount.ID).
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
				account1 := RemoteAccount{
					Login: "foo",
					Certs: []Certificate{
						{
							Name:        "cert",
							PrivateKey:  testhelpers.ClientKey,
							Certificate: testhelpers.ClientCert,
						},
					},
				}
				accounts := []RemoteAccount{account1}

				Convey("When calling the importRemoteAccounts method", func() {
					err := importRemoteAccounts(discard, db, accounts, agent.ID)

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
										So(db.Select(&cryptos).Where(
											"owner_type=? AND owner_id=?",
											model.TableRemAccounts, dbAccount.ID).
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
