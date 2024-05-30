package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestExportLocalAgents(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)
		owner := conf.GlobalConfig.GatewayName

		Convey("Given the database contains locals agents with accounts", func() {
			agent1 := &model.LocalAgent{
				Name: "agent1", Protocol: testProtocol,
				Address: types.Addr("localhost", 6666),
			}
			So(db.Insert(agent1).Run(), ShouldBeNil)

			// Change owner for this insert
			conf.GlobalConfig.GatewayName = "unknown"

			So(db.Insert(&model.LocalAgent{
				Name: "foo", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}).Run(), ShouldBeNil)
			// Revert database owner
			conf.GlobalConfig.GatewayName = owner

			account1a := &model.LocalAccount{
				LocalAgentID: agent1.ID,
				Login:        "acc1a",
			}
			So(db.Insert(account1a).Run(), ShouldBeNil)

			cert := &model.Credential{
				Name:         "test_cert",
				LocalAgentID: utils.NewNullInt64(agent1.ID),
				Type:         auth.TLSCertificate,
				Value:        testhelpers.LocalhostCert,
				Value2:       testhelpers.LocalhostKey,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			agent2 := &model.LocalAgent{
				Name: "agent2", Protocol: testProtocol,
				Address: types.Addr("localhost", 7777),
			}
			So(db.Insert(agent2).Run(), ShouldBeNil)

			account2a := &model.LocalAccount{
				LocalAgentID: agent2.ID,
				Login:        "acc2a",
			}
			So(db.Insert(account2a).Run(), ShouldBeNil)

			account2b := &model.LocalAccount{
				LocalAgentID: agent2.ID,
				Login:        "foo",
			}
			So(db.Insert(account2b).Run(), ShouldBeNil)

			Convey("Given an empty database", func() {
				Convey("When calling the exportLocal function", func() {
					res, err := exportLocals(discard(), db)
					So(err, ShouldBeNil)
					So(res, ShouldHaveLength, 2)

					Convey("Then it should have exported the first agent", func() {
						So(res[0].Protocol, ShouldEqual, agent1.Protocol)
						So(res[0].RootDir, ShouldEqual, agent1.RootDir)
						So(res[0].ReceiveDir, ShouldEqual, agent1.ReceiveDir)
						So(res[0].SendDir, ShouldEqual, agent1.SendDir)
						So(res[0].TmpReceiveDir, ShouldEqual, agent1.TmpReceiveDir)
						So(res[0].Address, ShouldEqual, agent1.Address.String())
						So(res[0].Configuration, ShouldResemble,
							agent1.ProtoConfig)

						So(res[0].Credentials, ShouldHaveLength, 1)
						So(res[0].Credentials[0].Name, ShouldEqual, cert.Name)
						So(res[0].Credentials[0].Type, ShouldEqual, cert.Type)
						So(res[0].Credentials[0].Value, ShouldEqual, cert.Value)
						So(res[0].Credentials[0].Value2, ShouldEqual, cert.Value2)

						So(res[0].Certs, ShouldHaveLength, 1)
						So(res[0].Certs[0].Name, ShouldEqual, cert.Name)
						So(res[0].Certs[0].Certificate, ShouldEqual, cert.Value)
						So(res[0].Certs[0].PrivateKey, ShouldEqual, cert.Value2)

						So(res[0].Accounts, ShouldHaveLength, 1)
					})

					Convey("Then it should have exported the second agent", func() {
						So(res[1].Protocol, ShouldEqual, agent2.Protocol)
						So(res[1].RootDir, ShouldEqual, agent2.RootDir)
						So(res[1].ReceiveDir, ShouldEqual, agent2.ReceiveDir)
						So(res[1].SendDir, ShouldEqual, agent2.SendDir)
						So(res[1].TmpReceiveDir, ShouldEqual, agent2.TmpReceiveDir)
						So(res[1].Address, ShouldEqual, agent2.Address.String())
						So(res[1].Configuration, ShouldResemble,
							agent2.ProtoConfig)

						So(res[1].Credentials, ShouldHaveLength, 0)
						So(res[1].Accounts, ShouldHaveLength, 2)
						So(res[1].Certs, ShouldHaveLength, 0)
					})
				})
			})
		})
	})
}

func TestExportLocalAccounts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the dabase contains a local agent with accounts", func() {
			agent := &model.LocalAgent{
				Name: "server", Protocol: testProtocol,
				Address: types.Addr("localhost", 2022),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			account1 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "acc1",
			}
			So(db.Insert(account1).Run(), ShouldBeNil)

			pswd := &model.Credential{
				Name:           "test_cert",
				LocalAccountID: utils.NewNullInt64(account1.ID),
				Type:           auth.Password,
				Value:          "foobar",
			}
			So(db.Insert(pswd).Run(), ShouldBeNil)

			account2 := &model.LocalAccount{
				LocalAgentID: agent.ID,
				Login:        "foo",
			}
			So(db.Insert(account2).Run(), ShouldBeNil)

			cert := &model.Credential{
				Name:           "test_cert",
				LocalAccountID: utils.NewNullInt64(account2.ID),
				Type:           auth.TLSTrustedCertificate,
				Value:          testhelpers.ClientFooCert,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			Convey("Given an empty database", func() {
				Convey("When calling the exportLocalAccounts function", func() {
					res, err := exportLocalAccounts(discard(), db, agent.ID)
					So(err, ShouldBeNil)
					So(res, ShouldHaveLength, 2)

					Convey("Then it should return the first account", func() {
						So(res[0].Login, ShouldEqual, account1.Login)
						So(res[0].PasswordHash, ShouldEqual, pswd.Value)

						So(res[0].Credentials, ShouldHaveLength, 1)
						So(res[0].Credentials[0].Name, ShouldEqual, pswd.Name)
						So(res[0].Credentials[0].Type, ShouldEqual, pswd.Type)
						So(res[0].Credentials[0].Value, ShouldEqual, pswd.Value)
					})

					Convey("Then it should return the second account", func() {
						So(res[1].Login, ShouldEqual, account2.Login)

						So(res[1].Credentials, ShouldHaveLength, 1)
						So(res[1].Credentials[0].Name, ShouldEqual, cert.Name)
						So(res[1].Credentials[0].Type, ShouldEqual, cert.Type)
						So(res[1].Credentials[0].Value, ShouldEqual, cert.Value)
						So(res[1].Credentials[0].Value2, ShouldEqual, cert.Value2)

						So(res[1].Certs, ShouldHaveLength, 1)
						So(res[1].Certs[0].Name, ShouldEqual, cert.Name)
						So(res[1].Certs[0].Certificate, ShouldEqual, cert.Value)
						So(res[1].Certs[0].PrivateKey, ShouldEqual, cert.Value2)
					})
				})
			})
		})
	})
}
