package backup

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	. "github.com/smartystreets/goconvey/convey"
)

func TestImportRemoteAgents(t *testing.T) {

	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a database with some remote agent", func() {
			agent := &model.RemoteAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Create(agent), ShouldBeNil)

			Convey("Given a list of new agents", func() {
				agent1 := remoteAgent{
					Name:          "foo",
					Protocol:      "sftp",
					Configuration: []byte(`{}`),
					Address:       "localhost:2022",
					Accounts: []remoteAccount{
						{
							Login:    "test",
							Password: "pwd",
						}, {
							Login:    "test2",
							Password: "pwd",
						},
					},
				}
				agents := []remoteAgent{agent1}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importRemotes method", func() {
						err := importRemoteAgents(discard, ses, agents)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the remote agents", func() {
							dbAgent := &model.RemoteAgent{
								Name: agent1.Name,
							}
							So(ses.Get(dbAgent), ShouldBeNil)

							Convey("Then the data should correspond to the "+
								"one imported", func() {
								So(dbAgent.Name, ShouldEqual, agent1.Name)
								So(dbAgent.Protocol, ShouldEqual, agent1.Protocol)
								So(dbAgent.ProtoConfig, ShouldResemble,
									agent1.Configuration)

								accounts := []model.RemoteAccount{}
								So(ses.Select(&accounts, &database.Filters{
									Conditions: builder.Eq{"remote_agent_id": dbAgent.ID},
								}), ShouldBeNil)

								So(len(accounts), ShouldEqual, 2)
							})
						})
					})
				})
			})

			Convey("Given a list of fully updated agents", func() {
				agent1 := remoteAgent{
					Name:          "test",
					Protocol:      "sftp",
					Configuration: []byte(`{}`),
					Address:       "localhost:90",
					Accounts: []remoteAccount{
						{
							Login:    "test",
							Password: "pwd",
						},
					},
					Certs: []certificate{
						{
							Name:        "cert",
							PublicKey:   "public",
							PrivateKey:  "private",
							Certificate: "key",
						},
					},
				}
				agents := []remoteAgent{agent1}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importRemotes method", func() {
						err := importRemoteAgents(discard, ses, agents)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the remote agents", func() {
							dbAgent := &model.RemoteAgent{
								Name: agent1.Name,
							}
							So(ses.Get(dbAgent), ShouldBeNil)

							Convey("Then the data should correspond to the "+
								"one imported", func() {
								So(dbAgent.Name, ShouldEqual, agent1.Name)
								So(dbAgent.Protocol, ShouldEqual, agent1.Protocol)
								So(dbAgent.ProtoConfig, ShouldResemble,
									agent1.Configuration)

								accounts := []model.RemoteAccount{}
								So(ses.Select(&accounts, &database.Filters{
									Conditions: builder.Eq{"remote_agent_id": dbAgent.ID},
								}), ShouldBeNil)

								So(len(accounts), ShouldEqual, 1)

								certs := []model.Cert{}
								So(ses.Select(&certs, &database.Filters{
									Conditions: builder.Eq{"owner_id": dbAgent.ID,
										"owner_type": "remote_agents"},
								}), ShouldBeNil)

								So(len(accounts), ShouldEqual, 1)
							})
						})
					})
				})
			})
		})
	})
}

func TestImportRemoteAccounts(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a database with some a remote agent and some remote accounts", func() {
			agent := &model.RemoteAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Create(agent), ShouldBeNil)

			dbAccount := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "foo",
				Password:      []byte("bar"),
			}
			So(db.Create(dbAccount), ShouldBeNil)

			Convey("Given a list of new accounts", func() {
				account1 := remoteAccount{
					Login:    "test",
					Password: "pwd",
				}
				account2 := remoteAccount{
					Login:    "test2",
					Password: "pwd",
				}
				accounts := []remoteAccount{
					account1, account2,
				}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importRemoteAccounts method", func() {
						err := importRemoteAccounts(discard, ses, accounts, agent.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the "+
							"remote accounts", func() {
							accounts := []model.RemoteAccount{}
							So(ses.Select(&accounts, &database.Filters{
								Conditions: builder.Eq{"remote_agent_id": agent.ID},
							}), ShouldBeNil)

							So(len(accounts), ShouldEqual, 3)

							Convey("Then the data should correspond to "+
								"the one imported", func() {
								for i := 0; i < len(accounts); i++ {
									if accounts[i].Login == account1.Login {

										Convey("Then account1 is found", func() {
											b, err := model.DecryptPassword(accounts[i].Password)
											So(err, ShouldBeNil)
											So(string(b), ShouldResemble, account1.Password)
										})
									} else if accounts[i].Login == account2.Login {

										Convey("Then account2 is found", func() {
											b, err := model.DecryptPassword(accounts[i].Password)
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
			})

			Convey("Given a list of fully updated agents", func() {
				account1 := remoteAccount{
					Login:    "foo",
					Password: "notbar",
					Certs: []certificate{
						{
							Name:        "cert",
							PublicKey:   "public",
							PrivateKey:  "private",
							Certificate: "key",
						},
					},
				}
				accounts := []remoteAccount{account1}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importRemoteAccounts method", func() {
						err := importRemoteAccounts(discard, ses, accounts, agent.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the "+
							"remote accounts", func() {
							accounts := []model.RemoteAccount{}
							So(ses.Select(&accounts, &database.Filters{
								Conditions: builder.Eq{"remote_agent_id": agent.ID},
							}), ShouldBeNil)

							So(len(accounts), ShouldEqual, 1)

							Convey("Then the data should correspond to "+
								"the one imported", func() {
								for i := 0; i < len(accounts); i++ {
									if accounts[i].Login == dbAccount.Login {

										Convey("When dbAccount is found", func() {
											So(accounts[i].Password, ShouldNotResemble,
												dbAccount.Password)
											certs := []model.Cert{}
											So(ses.Select(&certs, &database.Filters{
												Conditions: builder.Eq{"owner_id": dbAccount.ID,
													"owner_type": "remote_accounts"},
											}), ShouldBeNil)

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

			Convey("Given a list of partially updated agents", func() {
				account1 := remoteAccount{
					Login: "foo",
					Certs: []certificate{
						{
							Name:        "cert",
							PublicKey:   "public",
							PrivateKey:  "private",
							Certificate: "key",
						},
					},
				}
				accounts := []remoteAccount{account1}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importRemoteAccounts method", func() {
						err := importRemoteAccounts(discard, ses, accounts, agent.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the "+
							"remote accounts", func() {
							accounts := []model.RemoteAccount{}
							So(ses.Select(&accounts, &database.Filters{
								Conditions: builder.Eq{"remote_agent_id": agent.ID},
							}), ShouldBeNil)

							So(len(accounts), ShouldEqual, 1)

							Convey("Then the data should correspond to "+
								"the one imported", func() {
								for i := 0; i < len(accounts); i++ {
									if accounts[i].Login == dbAccount.Login {

										Convey("When dbAccount is found", func() {
											So(accounts[i].Password, ShouldResemble,
												dbAccount.Password)
											certs := []model.Cert{}
											So(ses.Select(&certs, &database.Filters{
												Conditions: builder.Eq{"owner_id": dbAccount.ID,
													"owner_type": "remote_accounts"},
											}), ShouldBeNil)

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
	})
}
