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

func TestImportCerts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a database with some Cryptos", func() {
			agent := &model.LocalAgent{
				Name:     "server",
				Protocol: testProtocol,
				Address:  types.Addr("localhost", 6666),
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			agent2 := &model.LocalAgent{
				Name:     "agent2",
				Protocol: testProtocol,
				Address:  types.Addr("localhost", 7777),
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

			Convey("Given a list of new Cryptos to import", func() {
				insert := Certificate{
					Name:        "new",
					PrivateKey:  testhelpers.LocalhostKey,
					Certificate: testhelpers.LocalhostCert,
				}
				Certificates := []Certificate{insert}

				Convey("When calling the importCerts with the new "+
					"Cryptos on the existing agent", func() {
					err := importCerts(discard(), db, Certificates, agent)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the agent should have 1 Cryptos", func() {
						var dbCerts model.Credentials
						So(db.Select(&dbCerts).Where("local_agent_id=?",
							agent.ID).Run(), ShouldBeNil)
						So(len(dbCerts), ShouldEqual, 1)

						Convey("Then the Certificate should correspond "+
							"to the one imported", func() {
							So(dbCerts[0].Name, ShouldResemble, insert.Name)
							So(dbCerts[0].Value2, ShouldResemble, insert.PrivateKey)
							So(dbCerts[0].Value, ShouldResemble, insert.Certificate)
						})
					})
				})
			})

			Convey("Given a updated Certificate to import", func() {
				insert := Certificate{
					Name:        "foo",
					PrivateKey:  testhelpers.LocalhostKey,
					Certificate: testhelpers.LocalhostCert,
				}
				Certificates := []Certificate{insert}

				Convey("When calling the importCerts with the new "+
					"Cryptos on the existing agent", func() {
					err := importCerts(discard(), db, Certificates, agent2)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the agent should have 1 Cryptos", func() {
						var dbCerts model.Credentials
						So(db.Select(&dbCerts).Where("local_agent_id=? AND type=?",
							agent2.ID, auth.TLSCertificate).Run(), ShouldBeNil)
						So(len(dbCerts), ShouldEqual, 1)

						Convey("Then the Certificate should correspond "+
							"to the one imported", func() {
							So(dbCerts[0].Name, ShouldResemble, insert.Name)
							So(dbCerts[0].Value2, ShouldResemble, insert.PrivateKey)
							So(dbCerts[0].Value, ShouldResemble, insert.Certificate)
						})
					})
				})
			})
		})
	})
}
