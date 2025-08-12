package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	. "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestImportRemoteAgents(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a database with some remote agent", func() {
			agent := &model.RemoteAgent{
				Name: "partner", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			other := &model.RemoteAgent{
				Name: "other", Protocol: testProtocol,
				Address: types.Addr("localhost", 8888),
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

					Convey("Then the database should contains the remote agents", func() {
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
					Credentials: []Credential{
						{
							Name:  "cert",
							Type:  auth.TLSTrustedCertificate,
							Value: testhelpers.LocalhostCert,
						},
					},
				}
				agents := []RemoteAgent{agent1}

				Convey("When calling the importRemotes method", func() {
					err := importRemoteAgents(discard(), db, agents, true)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})
					Convey("Then the database should contains the remote agents", func() {
						var dbAgents model.RemoteAgents
						So(db.Select(&dbAgents).Run(), ShouldBeNil)
						So(dbAgents, ShouldHaveLength, 1)

						dbAgent := dbAgents[0]

						Convey("Then it should have updated the agent", func() {
							So(dbAgent.Name, ShouldEqual, agent1.Name)
							So(dbAgent.Protocol, ShouldEqual, agent1.Protocol)
							So(dbAgent.ProtoConfig, ShouldResemble,
								agent1.Configuration)

							var accounts model.RemoteAccounts
							So(db.Select(&accounts).Where("remote_agent_id=?",
								dbAgent.ID).Run(), ShouldBeNil)

							So(len(accounts), ShouldEqual, 1)

							var credentials model.Credentials
							So(db.Select(&credentials).Where("remote_agent_id=?",
								dbAgent.ID).Run(), ShouldBeNil)

							So(len(accounts), ShouldEqual, 1)
						})
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
				Name: "partner", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			dbAccount := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "foo",
			}
			So(db.Insert(dbAccount).Run(), ShouldBeNil)

			pswd := &model.Credential{
				RemoteAccountID: utils.NewNullInt64(dbAccount.ID),
				Type:            auth.Password,
				Value:           "sesame",
			}
			So(db.Insert(pswd).Run(), ShouldBeNil)

			Convey("Given a list of new accounts", func() {
				account1 := RemoteAccount{
					Login:    "bar",
					Password: "pwd1",
					Credentials: []Credential{{
						Name:   "cert",
						Type:   auth.TLSCertificate,
						Value:  testhelpers.ClientBarCert,
						Value2: testhelpers.ClientBarKey,
					}},
				}
				account2 := RemoteAccount{
					Login:    "toto",
					Password: "pwd2",
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
							agent.ID).OrderBy("id", true).Run(), ShouldBeNil)
						So(accounts, ShouldHaveLength, 3)

						Convey("Then it should have imported the first account", func() {
							So(accounts[1].Login, ShouldResemble, account1.Login)

							var pwd model.Credential
							So(db.Get(&pwd, "remote_account_id=? AND type=?",
								accounts[1].ID, auth.Password).Run(), ShouldBeNil)
							So(pwd.Value, ShouldEqual, account1.Password)

							var cert model.Credential
							So(db.Get(&cert, "remote_account_id=? AND type=?",
								accounts[1].ID, auth.TLSCertificate).Run(), ShouldBeNil)
							So(cert.Value, ShouldEqual, account1.Credentials[0].Value)
							So(cert.Value2, ShouldEqual, account1.Credentials[0].Value2)
						})

						Convey("Then it should have imported the second account", func() {
							So(accounts[2].Login, ShouldResemble, account2.Login)

							var pswd model.Credential
							So(db.Get(&pswd, "remote_account_id=?", accounts[2].ID).Run(), ShouldBeNil)
							So(pswd.Type, ShouldEqual, auth.Password)
							So(pswd.Value, ShouldEqual, account2.Password)
						})

						Convey("Then the existing account should be unchanged", func() {
							So(accounts[0], ShouldResemble, dbAccount)
						})
					})
				})
			})

			Convey("Given a list of fully updated agents", func() {
				account1 := RemoteAccount{
					Login:    dbAccount.Login,
					Password: "new password",
					Certificates: []Certificate{
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
						So(accounts, ShouldHaveLength, 1)

						Convey("Then it should have updated the account", func() {
							So(accounts[0].Login, ShouldEqual, account1.Login)

							var pwd model.Credential
							So(db.Get(&pwd, "remote_account_id=? AND type=?",
								accounts[0].ID, auth.Password).Run(), ShouldBeNil)
							So(pwd.Value, ShouldEqual, account1.Password)

							var cert model.Credential
							So(db.Get(&cert, "remote_account_id=? AND type=?",
								accounts[0].ID, auth.TLSCertificate).Run(), ShouldBeNil)
							So(cert.Value, ShouldEqual, account1.Certificates[0].Certificate)
							So(cert.Value2, ShouldEqual, account1.Certificates[0].PrivateKey)
						})
					})
				})
			})

			Convey("Given a list of partially updated agents", func() {
				account1 := RemoteAccount{
					Login: dbAccount.Login,
					Certificates: []Certificate{{
						Name:        "cert",
						PrivateKey:  testhelpers.ClientFooKey,
						Certificate: testhelpers.ClientFooCert,
					}},
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

						Convey("Then ip should have updated the account", func() {
							So(accounts[0].Login, ShouldEqual, account1.Login)

							var pwd model.Credential
							So(db.Get(&pwd, "remote_account_id=? AND type=?",
								accounts[0].ID, auth.Password).Run(), ShouldBeNil)
							So(pwd.Value, ShouldEqual, pswd.Value)

							var cert model.Credential
							So(db.Get(&cert, "remote_account_id=? AND type=?",
								accounts[0].ID, auth.TLSCertificate).Run(), ShouldBeNil)
							So(cert.Value, ShouldEqual, account1.Certificates[0].Certificate)
							So(cert.Value2, ShouldEqual, account1.Certificates[0].PrivateKey)
						})
					})
				})
			})
		})
	})
}
