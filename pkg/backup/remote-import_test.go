package backup

import (
	"encoding/json"
	"testing"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
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
				Address:       "localhost:90",
				Accounts: []RemoteAccount{
					{
						Login:    "test",
						Password: "pwd",
					},
				},
				Certs: []Certificate{
					{
						Name:        "cert",
						PublicKey:   "public",
						PrivateKey:  "private",
						Certificate: "key",
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

						var certs model.Certificates
						So(db.Select(&certs).Where("owner_type='remote_agents' "+
							"AND owner_id=?", dbAgent.ID).Run(), ShouldBeNil)

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
				Password:      []byte("bar"),
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
										b, err := utils.DecryptPassword(database.GCM, accounts[i].Password)
										So(err, ShouldBeNil)
										So(string(b), ShouldResemble, account1.Password)
									})
								} else if accounts[i].Login == account2.Login {

									Convey("Then account2 is found", func() {
										b, err := utils.DecryptPassword(database.GCM, accounts[i].Password)
										So(err, ShouldBeNil)
										So(string(b), ShouldResemble, account2.Password)
									})
								} else if accounts[i].Login == dbAccount.Login {

									Convey("Then dbAccount is found", func() {
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
							PublicKey:   "public",
							PrivateKey:  "private",
							Certificate: "key",
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
										var certs model.Certificates
										So(db.Select(&certs).Where("owner_type='remote_accounts'"+
											" AND owner_id=?", dbAccount.ID).Run(), ShouldBeNil)

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
							PublicKey:   "public",
							PrivateKey:  "private",
							Certificate: "key",
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
										var certs model.Certificates
										So(db.Select(&certs).Where("owner_type='remote_accounts' AND "+
											"owner_id=?", dbAccount.ID).Run(), ShouldBeNil)

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
