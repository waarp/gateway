package backup

import (
	"encoding/json"
	"testing"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestImportCerts(t *testing.T) {

	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given a database with some Certificates", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			agent2 := &model.LocalAgent{
				Name:        "test2",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2023",
			}
			So(db.Insert(agent2).Run(), ShouldBeNil)

			cert2 := &model.Cert{
				Name:        "foo",
				OwnerType:   "local_agents",
				OwnerID:     agent2.ID,
				PublicKey:   []byte("abc"),
				PrivateKey:  []byte("zyx"),
				Certificate: []byte("cert"),
			}
			So(db.Insert(cert2).Run(), ShouldBeNil)

			Convey("Given a list of new Certificates to import", func() {
				insert := Certificate{
					Name:        "new",
					PublicKey:   "abc",
					PrivateKey:  "zyx",
					Certificate: "cert",
				}
				Certificates := []Certificate{insert}

				Convey("When calling the importCerts with the new "+
					"Certificates on the existing agent", func() {
					err := importCerts(discard, db, Certificates,
						"local_agents", agent.ID)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the agent should have 1 Certificates", func() {
						var dbCerts model.Certificates
						So(db.Select(&dbCerts).Where("owner_type='local_agents' AND "+
							"owner_id=?", agent.ID).Run(), ShouldBeNil)
						So(len(dbCerts), ShouldEqual, 1)

						Convey("Then the Certificate should correspond "+
							"to the one imported", func() {
							So(dbCerts[0].Name, ShouldResemble, insert.Name)
							So(dbCerts[0].PublicKey, ShouldResemble, []byte(insert.PublicKey))
							So(dbCerts[0].PrivateKey, ShouldResemble, []byte(insert.PrivateKey))
							So(dbCerts[0].Certificate, ShouldResemble,
								[]byte(insert.Certificate))
						})
					})
				})
			})

			Convey("Given a updated Certificate to import", func() {

				insert := Certificate{
					Name:        "foo",
					PublicKey:   "abc",
					PrivateKey:  "zyx",
					Certificate: "cert",
				}
				Certificates := []Certificate{insert}

				Convey("When calling the importCerts with the new "+
					"Certificates on the existing agent", func() {
					err := importCerts(discard, db, Certificates,
						"local_agents", agent2.ID)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the agent should have 1 Certificates", func() {
						var dbCerts model.Certificates
						So(db.Select(&dbCerts).Where("owner_type='local_agents' AND "+
							"owner_id=?", agent2.ID).Run(), ShouldBeNil)
						So(len(dbCerts), ShouldEqual, 1)

						Convey("Then the Certificate should correspond "+
							"to the one imported", func() {
							So(dbCerts[0].Name, ShouldResemble, insert.Name)
							So(dbCerts[0].PublicKey, ShouldResemble, []byte(insert.PublicKey))
							So(dbCerts[0].PrivateKey, ShouldResemble, []byte(insert.PrivateKey))
							So(dbCerts[0].Certificate, ShouldResemble,
								[]byte(insert.Certificate))
						})
					})

				})
			})
		})
	})
}
