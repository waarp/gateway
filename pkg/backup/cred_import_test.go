package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestImportAuth(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a database with some Credentials", func() {
			agent := &model.LocalAgent{
				Name: "server", Protocol: testProtocol,
				Address: types.Addr("localhost", 6666),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			agent2 := &model.LocalAgent{
				Name: "agent2", Protocol: testProtocol,
				Address: types.Addr("localhost", 7777),
			}
			So(db.Insert(agent2).Run(), ShouldBeNil)

			cert2 := &model.Credential{
				Name:         "foo",
				LocalAgentID: utils.NewNullInt64(agent2.ID),
				Type:         auth.TLSCertificate,
				Value2:       testhelpers.OtherLocalhostKey,
				Value:        testhelpers.OtherLocalhostCert,
			}
			So(db.Insert(cert2).Run(), ShouldBeNil)

			Convey("Given a list of new Credentials to import", func() {
				cert := file.Credential{
					Name:   "cert",
					Type:   auth.TLSCertificate,
					Value:  testhelpers.LocalhostCert,
					Value2: testhelpers.LocalhostKey,
				}
				pswd := file.Credential{
					Name:  "pswd",
					Type:  auth.Password,
					Value: "foobar",
				}
				credentials := []file.Credential{cert, pswd}

				Convey("When calling the importCerts with the new "+
					"Credentials on the existing agent", func() {
					err := credentialsImport(discard(), db, credentials, agent, agent.Protocol)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the agent should have 1 Credentials", func() {
						var dbCerts model.Credentials
						So(db.Select(&dbCerts).Where("local_agent_id=?",
							agent.ID).Run(), ShouldBeNil)
						So(len(dbCerts), ShouldEqual, 2)

						Convey("Then the Certificate should correspond "+
							"to the one imported", func() {
							So(dbCerts[0].Name, ShouldResemble, cert.Name)
							So(dbCerts[0].Value, ShouldResemble, cert.Value)
							So(dbCerts[0].Value2, ShouldResemble, cert.Value2)

							So(dbCerts[1].Name, ShouldResemble, pswd.Name)
							So(dbCerts[1].Value, ShouldEqual, pswd.Value)
						})
					})
				})
			})

			Convey("Given a updated Certificate to import", func() {
				cert := file.Credential{
					Name:   "foo",
					Type:   auth.TLSCertificate,
					Value:  testhelpers.LocalhostCert,
					Value2: testhelpers.LocalhostKey,
				}
				credentials := []file.Credential{cert}

				Convey("When calling the credentialsImport with the new "+
					"Credentials on the existing agent", func() {
					err := credentialsImport(discard(), db, credentials, agent2, agent2.Protocol)
					So(err, ShouldBeNil)

					Convey("Then the agent should have 1 Credentials", func() {
						var dbCerts model.Credentials
						So(db.Select(&dbCerts).Where("type=?", auth.TLSCertificate).
							Where(agent2.GetCredCond()).Run(), ShouldBeNil)
						So(dbCerts, ShouldHaveLength, 1)

						Convey("Then the Certificate should correspond "+
							"to the one imported", func() {
							So(dbCerts[0].Name, ShouldResemble, cert.Name)
							So(dbCerts[0].Value, ShouldResemble, cert.Value)
							So(dbCerts[0].Value2, ShouldResemble, cert.Value2)
						})
					})
				})
			})
		})
	})
}
