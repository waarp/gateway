package backup

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExportRemoteAgents(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains remotes agents with accounts", func() {
			agent1 := &model.RemoteAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(agent1).Run(), ShouldBeNil)

			account1a := &model.RemoteAccount{
				RemoteAgentID: agent1.ID,
				Login:         "test",
				Password:      []byte("pwd"),
			}
			So(db.Insert(account1a).Run(), ShouldBeNil)

			cert := &model.Cert{
				Name:        "test_cert",
				OwnerType:   "remote_agents",
				OwnerID:     agent1.ID,
				Certificate: []byte("cert"),
				PublicKey:   []byte("public"),
				PrivateKey:  []byte("private"),
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			agent2 := &model.RemoteAgent{
				Name:        "test2",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2023",
			}
			So(db.Insert(agent2).Run(), ShouldBeNil)

			account2a := &model.RemoteAccount{
				RemoteAgentID: agent2.ID,
				Login:         "test",
				Password:      []byte("pwd"),
			}
			So(db.Insert(account2a).Run(), ShouldBeNil)

			account2b := &model.RemoteAccount{
				RemoteAgentID: agent2.ID,
				Login:         "foo",
				Password:      []byte("pwd"),
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
						if res[i].Name == agent1.Name {

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
						} else if res[i].Name == agent2.Name {

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
}

func TestExportRemoteAccounts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains a remote agent with accounts", func() {
			pwd1 := "pwd"
			pwd2 := "bar"
			agent := &model.RemoteAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account1 := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "test",
				Password:      []byte(pwd1),
			}
			So(db.Insert(account1).Run(), ShouldBeNil)

			account2 := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "foo",
				Password:      []byte(pwd2),
			}
			So(db.Insert(account2).Run(), ShouldBeNil)

			cert := &model.Cert{
				Name:        "test_cert",
				OwnerType:   "remote_accounts",
				OwnerID:     account2.ID,
				Certificate: []byte("cert"),
				PublicKey:   []byte("public"),
				PrivateKey:  []byte("private"),
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
						if res[i].Login == account1.Login {

							Convey("When login1 is found", func() {

								Convey("Then it should be equal to the data in DB", func() {
									So(res[i].Password, ShouldResemble, pwd1)
								})

								Convey("Then it should have no certificate", func() {
									So(len(res[i].Certs), ShouldEqual, 0)
								})
							})
						} else if res[i].Login == account2.Login {

							Convey("When login2 is found", func() {

								Convey("Then it should be equal to the data in DB", func() {
									So(res[i].Password, ShouldResemble, pwd2)
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
}
