package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestExportRemoteAgents(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains remotes agents with accounts", func() {
			agent1 := &model.RemoteAgent{
				Name:     "agent1",
				Protocol: testProtocol,
				Address:  "localhost:6666",
			}
			So(db.Insert(agent1).Run(), ShouldBeNil)

			account1a := &model.RemoteAccount{
				RemoteAgentID: agent1.ID,
				Login:         "acc1a",
				Password:      "pwd",
			}
			So(db.Insert(account1a).Run(), ShouldBeNil)

			cert := &model.Crypto{
				Name:        "test_cert",
				OwnerType:   model.TableRemAgents,
				OwnerID:     agent1.ID,
				Certificate: testhelpers.LocalhostCert,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			agent2 := &model.RemoteAgent{
				Name:     "agent2",
				Protocol: testProtocol,
				Address:  "localhost:2023",
			}
			So(db.Insert(agent2).Run(), ShouldBeNil)

			account2a := &model.RemoteAccount{
				RemoteAgentID: agent2.ID,
				Login:         "acc2a",
				Password:      "pwd",
			}
			So(db.Insert(account2a).Run(), ShouldBeNil)

			account2b := &model.RemoteAccount{
				RemoteAgentID: agent2.ID,
				Login:         "foo",
				Password:      "pwd",
			}
			So(db.Insert(account2b).Run(), ShouldBeNil)

			Convey("When calling the exportRemote function", func() {
				res, err := exportRemotes(discard, db)

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return 2 remote agents", func() {
					So(len(res), ShouldEqual, 2)
				})

				Convey("When searching for remote agents", func() {
					for i := 0; i < len(res); i++ {
						switch {
						case res[i].Name == agent1.Name:
							Convey("When agent1 is found", func() {
								Convey("Then it should be equal to the data in DB", func() {
									So(res[i].Protocol, ShouldEqual, agent1.Protocol)
									So(res[i].Address, ShouldEqual, agent1.Address)
									So(res[i].Configuration, ShouldResemble,
										agent1.ProtoConfig)

									Convey("Then it should have 1 remote Account", func() {
										So(len(res[i].Accounts), ShouldEqual, 1)
									})

									Convey("Then it should have 1 certificate", func() {
										So(len(res[i].Certs), ShouldEqual, 1)
									})
								})
							})

						case res[i].Name == agent2.Name:
							Convey("When agent2 is found", func() {
								Convey("Then it should be equal to the data in DB", func() {
									So(res[i].Protocol, ShouldEqual, agent2.Protocol)
									So(res[i].Address, ShouldEqual, agent2.Address)
									So(res[i].Configuration, ShouldResemble,
										agent2.ProtoConfig)
									Convey("Then it should have 2 remote Account", func() {
										So(len(res[i].Accounts), ShouldEqual, 2)
									})

									Convey("Then it should have no certificate", func() {
										So(len(res[i].Certs), ShouldEqual, 0)
									})
								})
							})

						default:
							Convey("Then they should be no other records", func() {
								So(1, ShouldBeNil)
							})
						}
					}
				})
			})
		})
	})
}

func TestExportRemoteAccounts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains a remote agent with accounts", func() {
			pwd1 := "pwd"
			pwd2 := "bar"
			agent := &model.RemoteAgent{
				Name:     "partner",
				Protocol: testProtocol,
				Address:  "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account1 := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "acc1",
				Password:      types.CypherText(pwd1),
			}
			So(db.Insert(account1).Run(), ShouldBeNil)

			account2 := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "foo",
				Password:      types.CypherText(pwd2),
			}
			So(db.Insert(account2).Run(), ShouldBeNil)

			cert := &model.Crypto{
				Name:        "test_cert",
				OwnerType:   model.TableRemAccounts,
				OwnerID:     account2.ID,
				Certificate: testhelpers.ClientFooCert,
				PrivateKey:  testhelpers.ClientFooKey,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			Convey("When calling the exportRemoteAccounts function", func() {
				res, err := exportRemoteAccounts(discard, db, agent.ID)

				Convey("Then it should return no error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then it should return 2 remote accounts", func() {
					So(len(res), ShouldEqual, 2)
				})

				Convey("When searching for remote accounts", func() {
					for i := 0; i < len(res); i++ {
						switch {
						case res[i].Login == account1.Login:
							Convey("When login1 is found", func() {
								Convey("Then it should be equal to the data in DB", func() {
									So(res[i].Password, ShouldResemble, pwd1)
								})

								Convey("Then it should have no certificate", func() {
									So(len(res[i].Certs), ShouldEqual, 0)
								})
							})
						case res[i].Login == account2.Login:
							Convey("When login2 is found", func() {
								Convey("Then it should be equal to the data in DB", func() {
									So(res[i].Password, ShouldResemble, pwd2)
								})

								Convey("Then it should have 1 certificate", func() {
									So(len(res[i].Certs), ShouldEqual, 1)
								})
							})
						default:
							Convey("Then they should be no other records", func() {
								So(1, ShouldBeNil)
							})
						}
					}
				})
			})
		})
	})
}
