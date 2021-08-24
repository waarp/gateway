package backup

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExportLocalAgents(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")
		owner := conf.GlobalConfig.GatewayName

		Convey("Given the database contains locals agents with accounts", func() {
			agent1 := &model.LocalAgent{
				Name:        "test",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:6666",
			}
			So(db.Insert(agent1).Run(), ShouldBeNil)

			// Change owner for this insert
			conf.GlobalConfig.GatewayName = "tata"
			So(db.Insert(&model.LocalAgent{
				Name:        "foo",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}).Run(), ShouldBeNil)
			// Revert database owner
			conf.GlobalConfig.GatewayName = owner

			account1a := &model.LocalAccount{
				LocalAgentID: agent1.ID,
				Login:        "test",
				PasswordHash: hash("pwd"),
			}
			So(db.Insert(account1a).Run(), ShouldBeNil)

			cert := &model.Crypto{
				Name:        "test_cert",
				OwnerType:   model.TableLocAgents,
				OwnerID:     agent1.ID,
				Certificate: testhelpers.LocalhostCert,
				PrivateKey:  testhelpers.LocalhostKey,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			agent2 := &model.LocalAgent{
				Name:        "test2",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:6666",
			}
			So(db.Insert(agent2).Run(), ShouldBeNil)

			account2a := &model.LocalAccount{
				LocalAgentID: agent2.ID,
				Login:        "test",
				PasswordHash: hash("pwd"),
			}
			So(db.Insert(account2a).Run(), ShouldBeNil)

			account2b := &model.LocalAccount{
				LocalAgentID: agent2.ID,
				Login:        "foo",
				PasswordHash: hash("pwd"),
			}
			So(db.Insert(account2b).Run(), ShouldBeNil)

			Convey("Given an empty database", func() {

				Convey("When calling the exportLocal function", func() {
					res, err := exportLocals(discard, db)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should return 2 local agents", func() {
						So(len(res), ShouldEqual, 2)
					})

					Convey("When searching for local agents", func() {
						for i := 0; i < len(res); i++ {
							if res[i].Name == agent1.Name {

								Convey("When agent1 is found", func() {

									Convey("Then it should be equal to the data in DB", func() {
										So(res[i].Protocol, ShouldEqual, agent1.Protocol)
										So(res[i].Root, ShouldEqual, agent1.Root)
										So(res[i].InDir, ShouldEqual, agent1.InDir)
										So(res[i].OutDir, ShouldEqual, agent1.OutDir)
										So(res[i].WorkDir, ShouldEqual, agent1.WorkDir)
										So(res[i].Address, ShouldEqual, agent1.Address)
										So(res[i].Configuration, ShouldResemble,
											agent1.ProtoConfig)

										Convey("Then it should have 1 local Account", func() {
											So(len(res[i].Accounts), ShouldEqual, 1)
										})

										Convey("Then it should have 1 certificate", func() {
											So(len(res[i].Certs), ShouldEqual, 1)
										})
									})
								})
							} else if res[i].Name == agent2.Name {

								Convey("When agent2 is found", func() {

									Convey("Then it should be equal to the data in DB", func() {
										So(res[i].Protocol, ShouldEqual, agent2.Protocol)
										So(res[i].Root, ShouldEqual, agent2.Root)
										So(res[i].InDir, ShouldEqual, agent2.InDir)
										So(res[i].OutDir, ShouldEqual, agent2.OutDir)
										So(res[i].WorkDir, ShouldEqual, agent2.WorkDir)
										So(res[i].Address, ShouldEqual, agent2.Address)
										So(res[i].Configuration, ShouldResemble,
											agent2.ProtoConfig)

										Convey("Then it should have 2 local Account", func() {
											So(len(res[i].Accounts), ShouldEqual, 2)
										})

										Convey("Then it should have no certificate", func() {
											So(len(res[i].Certs), ShouldEqual, 0)
										})
									})
								})
							} else {

								Convey("Then they should be no other records", func() {
									So(1, ShouldBeNil)
								})
							}
						}
					})
				})
			})
		})
	})
}

func TestExportLocalAccounts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the dabase contains a local agent with accounts", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account1 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "test",
				PasswordHash: hash("pwd"),
			}
			So(db.Insert(account1).Run(), ShouldBeNil)

			account2 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "foo",
				PasswordHash: hash("bar"),
			}
			So(db.Insert(account2).Run(), ShouldBeNil)

			cert := &model.Crypto{
				Name:        "test_cert",
				OwnerType:   model.TableLocAccounts,
				OwnerID:     account2.ID,
				Certificate: testhelpers.ClientCert,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			Convey("Given an empty database", func() {

				Convey("When calling the exportLocalAccounts function", func() {
					res, err := exportLocalAccounts(discard, db, agent.ID)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should return 2 local accounts", func() {
						So(len(res), ShouldEqual, 2)
					})

					Convey("When searching for local accounts", func() {
						for i := 0; i < len(res); i++ {
							if res[i].Login == account1.Login {

								Convey("When login1 is found", func() {

									Convey("Then it should be equal to the data in DB", func() {
										So(res[i].PasswordHash, ShouldResemble,
											string(account1.PasswordHash))
									})

									Convey("Then it should have no certificate", func() {
										So(len(res[i].Certs), ShouldEqual, 0)
									})
								})
							} else if res[i].Login == account2.Login {

								Convey("When login2 is found", func() {

									Convey("Then it should be equal to the data in DB", func() {
										So(res[i].PasswordHash, ShouldResemble,
											string(account2.PasswordHash))
									})

									Convey("Then it should have 1 certificate", func() {
										So(len(res[i].Certs), ShouldEqual, 1)
									})
								})
							} else {

								Convey("Then they should be no other records", func() {
									So(1, ShouldBeNil)
								})
							}
						}
					})
				})
			})

		})
	})
}
