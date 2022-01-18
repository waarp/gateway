package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestExportRemoteAgents(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains remotes agents with accounts", func() {
			agent1 := &model.RemoteAgent{
				Name: "agent1", Protocol: testProtocol,
				Address: types.Addr("localhost", 6666),
			}
			So(db.Insert(agent1).Run(), ShouldBeNil)

			account1a := &model.RemoteAccount{
				RemoteAgentID: agent1.ID,
				Login:         "acc1a",
			}
			So(db.Insert(account1a).Run(), ShouldBeNil)

			cert := &model.Credential{
				Name:          "test_cert",
				RemoteAgentID: utils.NewNullInt64(agent1.ID),
				Type:          auth.TLSTrustedCertificate,
				Value:         testhelpers.LocalhostCert,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			agent2 := &model.RemoteAgent{
				Name: "agent2", Protocol: testProtocol,
				Address: types.Addr("localhost", 2023),
			}
			So(db.Insert(agent2).Run(), ShouldBeNil)

			account2a := &model.RemoteAccount{
				RemoteAgentID: agent2.ID,
				Login:         "acc2a",
			}
			So(db.Insert(account2a).Run(), ShouldBeNil)

			account2b := &model.RemoteAccount{
				RemoteAgentID: agent2.ID,
				Login:         "foo",
			}
			So(db.Insert(account2b).Run(), ShouldBeNil)

			Convey("When calling the exportRemote function", func() {
				res, err := exportRemotes(discard(), db)
				So(err, ShouldBeNil)
				So(res, ShouldHaveLength, 2)

				Convey("Then it should have exported the first agent", func() {
					So(res[0].Protocol, ShouldEqual, agent1.Protocol)
					So(res[0].Address, ShouldEqual, agent1.Address.String())
					So(res[0].Configuration, ShouldResemble,
						agent1.ProtoConfig)

					So(res[0].Credentials, ShouldHaveLength, 1)
					So(res[0].Credentials[0].Name, ShouldEqual, cert.Name)
					So(res[0].Credentials[0].Type, ShouldEqual, cert.Type)
					So(res[0].Credentials[0].Value, ShouldEqual, cert.Value)

					So(res[0].Certs, ShouldHaveLength, 1)
					So(res[0].Certs[0].Name, ShouldEqual, cert.Name)
					So(res[0].Certs[0].Certificate, ShouldEqual, cert.Value)

					So(res[0].Accounts, ShouldHaveLength, 1)
				})

				Convey("Then it should have exported the second agent", func() {
					So(res[1].Protocol, ShouldEqual, agent2.Protocol)
					So(res[1].Address, ShouldEqual, agent2.Address.String())
					So(res[1].Configuration, ShouldResemble,
						agent2.ProtoConfig)

					So(res[1].Credentials, ShouldHaveLength, 0)
					So(res[1].Certs, ShouldHaveLength, 0)
					So(res[1].Accounts, ShouldHaveLength, 2)
				})
			})
		})
	})
}

func TestExportRemoteAccounts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains a remote agent with accounts", func() {
			pwd1 := "pwd"
			agent := &model.RemoteAgent{
				Name: "partner", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account1 := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "acc1",
			}
			So(db.Insert(account1).Run(), ShouldBeNil)

			pswd := &model.Credential{
				RemoteAccountID: utils.NewNullInt64(account1.ID),
				Type:            auth.Password,
				Value:           pwd1,
			}
			So(db.Insert(pswd).Run(), ShouldBeNil)

			account2 := &model.RemoteAccount{
				RemoteAgentID: agent.ID,
				Login:         "foo",
			}
			So(db.Insert(account2).Run(), ShouldBeNil)

			cert := &model.Credential{
				Name:            "test_cert",
				RemoteAccountID: utils.NewNullInt64(account2.ID),
				Type:            auth.TLSCertificate,
				Value:           testhelpers.ClientFooCert,
				Value2:          testhelpers.ClientFooKey,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			Convey("When calling the exportRemoteAccounts function", func() {
				res, err := exportRemoteAccounts(discard(), db, agent.ID)
				So(err, ShouldBeNil)
				So(res, ShouldHaveLength, 2)

				Convey("Then it should have exported the first account", func() {
					So(res[0].Login, ShouldEqual, account1.Login)
					So(res[0].Password, ShouldEqual, pwd1)
					So(res[0].Certs, ShouldHaveLength, 0)

					So(res[0].Credentials, ShouldHaveLength, 1)
					So(res[0].Credentials[0].Name, ShouldEqual, pswd.Name)
					So(res[0].Credentials[0].Type, ShouldEqual, pswd.Type)
					So(res[0].Credentials[0].Value, ShouldEqual, pswd.Value)
				})

				Convey("Then it should have exported the second account", func() {
					So(res[1].Login, ShouldEqual, account2.Login)
					So(res[1].Password, ShouldBeEmpty)

					So(res[1].Certs, ShouldHaveLength, 1)
					So(res[1].Certs[0].Name, ShouldEqual, cert.Name)
					So(res[1].Certs[0].Certificate, ShouldEqual, cert.Value)
					So(res[1].Certs[0].PrivateKey, ShouldEqual, cert.Value2)

					So(res[1].Credentials, ShouldHaveLength, 1)
					So(res[1].Credentials[0].Name, ShouldEqual, cert.Name)
					So(res[1].Credentials[0].Type, ShouldEqual, cert.Type)
					So(res[1].Credentials[0].Value, ShouldEqual, cert.Value)
					So(res[1].Credentials[0].Value2, ShouldEqual, cert.Value2)
				})
			})
		})
	})
}
