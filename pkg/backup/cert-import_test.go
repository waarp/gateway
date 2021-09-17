package backup

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	. "code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestImportCerts(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a database with some Cryptos", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:6666",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			agent2 := &model.LocalAgent{
				Name:        "test2",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:6666",
			}
			So(db.Insert(agent2).Run(), ShouldBeNil)

			cert2 := &model.Crypto{
				Name:        "foo",
				OwnerType:   model.TableLocAgents,
				OwnerID:     agent2.ID,
				PrivateKey:  testhelpers.LocalhostKey,
				Certificate: testhelpers.LocalhostCert,
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
					err := importCerts(discard, db, Certificates,
						model.TableLocAgents, agent.ID)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the agent should have 1 Cryptos", func() {
						var dbCerts model.Cryptos
						So(db.Select(&dbCerts).Where("owner_type=? AND owner_id=?",
							model.TableLocAgents, agent.ID).Run(), ShouldBeNil)
						So(len(dbCerts), ShouldEqual, 1)

						Convey("Then the Certificate should correspond "+
							"to the one imported", func() {
							So(dbCerts[0].Name, ShouldResemble, insert.Name)
							So(dbCerts[0].SSHPublicKey, ShouldResemble, insert.PublicKey)
							So(string(dbCerts[0].PrivateKey), ShouldResemble, insert.PrivateKey)
							So(dbCerts[0].Certificate, ShouldResemble, insert.Certificate)
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
					err := importCerts(discard, db, Certificates,
						model.TableLocAgents, agent2.ID)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the agent should have 1 Cryptos", func() {
						var dbCerts model.Cryptos
						So(db.Select(&dbCerts).Where("owner_type=? AND owner_id=?",
							model.TableLocAgents, agent2.ID).Run(), ShouldBeNil)
						So(len(dbCerts), ShouldEqual, 1)

						Convey("Then the Certificate should correspond "+
							"to the one imported", func() {
							So(dbCerts[0].Name, ShouldResemble, insert.Name)
							So(dbCerts[0].SSHPublicKey, ShouldResemble, insert.PublicKey)
							So(string(dbCerts[0].PrivateKey), ShouldResemble, insert.PrivateKey)
							So(dbCerts[0].Certificate, ShouldResemble, insert.Certificate)
						})
					})
				})
			})
		})
	})
}
