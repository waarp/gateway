package backup

import (
	"encoding/json"
	"testing"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

func TestImportLocalAgents(t *testing.T) {

	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a database with some local agent", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Create(agent), ShouldBeNil)

			Convey("Given a list of new agents", func() {
				agent1 := LocalAgent{
					Name:          "foo",
					Protocol:      "sftp",
					Configuration: json.RawMessage(`{}`),
					Address:       "localhost:2022",
					Accounts: []LocalAccount{
						{
							Login:    "test",
							Password: "pwd",
						}, {
							Login:    "test2",
							Password: "pwd",
						},
					},
				}
				agents := []LocalAgent{agent1}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importLocals method", func() {
						err := importLocalAgents(discard, ses, agents)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the local agents", func() {
							dbAgent := &model.LocalAgent{
								Name: agent1.Name,
							}
							So(ses.Get(dbAgent), ShouldBeNil)

							Convey("Then the data shuld correspond to the "+
								"one imported", func() {
								So(dbAgent.Name, ShouldEqual, agent1.Name)
								So(dbAgent.Protocol, ShouldEqual, agent1.Protocol)
								So(dbAgent.ProtoConfig, ShouldResemble,
									agent1.Configuration)

								var accounts []model.LocalAccount
								So(ses.Select(&accounts, &database.Filters{
									Conditions: builder.Eq{"local_agent_id": dbAgent.ID},
								}), ShouldBeNil)

								So(len(accounts), ShouldEqual, 2)
							})
						})
					})
				})
			})

			Convey("Given a list of fully updated agents", func() {
				agent1 := LocalAgent{
					Name:          "test",
					Protocol:      "sftp",
					Configuration: json.RawMessage(`{}`),
					Address:       "localhost:90",
					Accounts: []LocalAccount{
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
				agents := []LocalAgent{agent1}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importLocals method", func() {
						err := importLocalAgents(discard, ses, agents)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the local agents", func() {
							dbAgent := &model.LocalAgent{
								Name: agent1.Name,
							}
							So(ses.Get(dbAgent), ShouldBeNil)

							Convey("Then the data shuld correspond to the "+
								"one imported", func() {
								So(dbAgent.Name, ShouldEqual, agent1.Name)
								So(dbAgent.Protocol, ShouldEqual, agent1.Protocol)
								So(dbAgent.ProtoConfig, ShouldResemble,
									agent1.Configuration)

								var accounts []model.LocalAccount
								So(ses.Select(&accounts, &database.Filters{
									Conditions: builder.Eq{"local_agent_id": dbAgent.ID},
								}), ShouldBeNil)

								So(len(accounts), ShouldEqual, 1)

								var certs []model.Cert
								So(ses.Select(&certs, &database.Filters{
									Conditions: builder.Eq{"owner_id": dbAgent.ID,
										"owner_type": "local_agents"},
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

func TestImportLocalAccounts(t *testing.T) {

	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a database with some a local agent and some local accounts", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Create(agent), ShouldBeNil)

			dbAccount := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "foo",
				Password:     []byte("bar"),
			}
			So(db.Create(dbAccount), ShouldBeNil)

			Convey("Given a list of new accounts", func() {
				account1 := LocalAccount{
					Login:    "test",
					Password: "pwd",
				}
				account2 := LocalAccount{
					Login:    "test2",
					Password: "pwd",
				}
				accounts := []LocalAccount{
					account1, account2,
				}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importLocalAccounts method", func() {
						err := importLocalAccounts(discard, ses, accounts, agent.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the local "+
							"accounts", func() {
							var accounts []model.LocalAccount
							So(ses.Select(&accounts, &database.Filters{
								Conditions: builder.Eq{"local_agent_id": agent.ID},
							}), ShouldBeNil)

							So(len(accounts), ShouldEqual, 3)

							Convey("Then the data should correspond to the "+
								"one imported", func() {
								for i := 0; i < len(accounts); i++ {
									if accounts[i].Login == account1.Login {

										Convey("Then account1 is found", func() {
											So(bcrypt.CompareHashAndPassword(
												accounts[i].Password, []byte("pwd")),
												ShouldBeNil)
										})
									} else if accounts[i].Login == account2.Login {

										Convey("Then account2 is found", func() {
											So(bcrypt.CompareHashAndPassword(
												accounts[i].Password, []byte("pwd")),
												ShouldBeNil)
										})
									} else if accounts[i].Login == dbAccount.Login {

										Convey("Then dbAccount is found", func() {
											So(accounts[i].Password, ShouldResemble,
												dbAccount.Password)
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
			})

			Convey("Given a list of fully updated agents", func() {
				account1 := LocalAccount{
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
				accounts := []LocalAccount{account1}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importLocalAccounts method", func() {
						err := importLocalAccounts(discard, ses, accounts, agent.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the "+
							"local accounts", func() {
							var accounts []model.LocalAccount
							So(ses.Select(&accounts, &database.Filters{
								Conditions: builder.Eq{"local_agent_id": agent.ID},
							}), ShouldBeNil)

							So(len(accounts), ShouldEqual, 1)

							Convey("Then the data should correspond to the "+
								"one imported", func() {
								for i := 0; i < len(accounts); i++ {
									if accounts[i].Login == dbAccount.Login {

										Convey("When dbAccount is found", func() {
											So(accounts[i].Password, ShouldNotResemble,
												dbAccount.Password)
											var certs []model.Cert
											So(ses.Select(&certs, &database.Filters{
												Conditions: builder.Eq{"owner_id": dbAccount.ID,
													"owner_type": "local_accounts"},
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
				account1 := LocalAccount{
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
				accounts := []LocalAccount{account1}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importLocalAccounts method", func() {
						err := importLocalAccounts(discard, ses, accounts, agent.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})
						Convey("Then the database should contains the "+
							"local accounts", func() {
							var accounts []model.LocalAccount
							So(ses.Select(&accounts, &database.Filters{
								Conditions: builder.Eq{"local_agent_id": agent.ID},
							}), ShouldBeNil)

							So(len(accounts), ShouldEqual, 1)

							Convey("Then the data should correspond to the "+
								"one imported", func() {
								for i := 0; i < len(accounts); i++ {
									if accounts[i].Login == dbAccount.Login {

										Convey("When dbAccount is found", func() {
											So(accounts[i].Password, ShouldResemble,
												dbAccount.Password)
											var certs []model.Cert
											So(ses.Select(&certs, &database.Filters{
												Conditions: builder.Eq{"owner_id": dbAccount.ID,
													"owner_type": "local_accounts"},
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
