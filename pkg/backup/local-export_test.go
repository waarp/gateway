package backup

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExportLocalAgents(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()
		owner := database.Owner

		Convey("Given the database contains locals agents with accounts", func() {
			agent1 := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(agent1), ShouldBeNil)

			// Change owner for this insert
			database.Owner = "tata"
			So(db.Create(&model.LocalAgent{
				Name:        "foo",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{"address":"localhost","port":2022}`),
			}), ShouldBeNil)
			// Revert database owner
			database.Owner = owner

			account1a := &model.LocalAccount{
				LocalAgentID: agent1.ID,
				Login:        "test",
				Password:     []byte("pwd"),
			}
			So(db.Create(account1a), ShouldBeNil)

			cert := &model.Cert{
				Name:        "test_cert",
				OwnerType:   "local_agents",
				OwnerID:     agent1.ID,
				Certificate: []byte("cert"),
				PublicKey:   []byte("public"),
				PrivateKey:  []byte("private"),
			}
			So(db.Create(cert), ShouldBeNil)

			agent2 := &model.LocalAgent{
				Name:        "test2",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(agent2), ShouldBeNil)

			account2a := &model.LocalAccount{
				LocalAgentID: agent2.ID,
				Login:        "test",
				Password:     []byte("pwd"),
			}
			So(db.Create(account2a), ShouldBeNil)

			account2b := &model.LocalAccount{
				LocalAgentID: agent2.ID,
				Login:        "foo",
				Password:     []byte("pwd"),
			}
			So(db.Create(account2b), ShouldBeNil)

			Convey("Given a new Transaction", func() {
				ses, err := db.BeginTransaction()
				So(err, ShouldBeNil)

				defer ses.Rollback()

				Convey("When calling the exportLocal function", func() {
					res, err := exportLocals(discard, ses)

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
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the dabase contains a local agent with accounts", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(agent), ShouldBeNil)

			account1 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "test",
				Password:     []byte("pwd"),
			}
			So(db.Create(account1), ShouldBeNil)

			account2 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "foo",
				Password:     []byte("bar"),
			}
			So(db.Create(account2), ShouldBeNil)

			cert := &model.Cert{
				Name:        "test_cert",
				OwnerType:   "local_accounts",
				OwnerID:     account2.ID,
				Certificate: []byte("cert"),
				PublicKey:   []byte("public"),
				PrivateKey:  []byte("private"),
			}
			So(db.Create(cert), ShouldBeNil)

			Convey("Given a new Transaction", func() {
				ses, err := db.BeginTransaction()
				So(err, ShouldBeNil)

				defer ses.Rollback()

				Convey("When calling the exportLocalAccounts function", func() {
					res, err := exportLocalAccounts(discard, ses, agent.ID)

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
										So(res[i].Password, ShouldResemble,
											string(account1.Password))
									})

									Convey("Then it should have no certificate", func() {
										So(len(res[i].Certs), ShouldEqual, 0)
									})
								})
							} else if res[i].Login == account2.Login {

								Convey("When login2 is found", func() {

									Convey("Then it should be equal to the data in DB", func() {
										So(res[i].Password, ShouldResemble,
											string(account2.Password))
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
