package backup

import (
	"encoding/json"
	"testing"

	. "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup/file"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	. "github.com/smartystreets/goconvey/convey"
)

func TestImportCerts(t *testing.T) {

	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a database with some Certificates", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2022",
			}
			So(db.Create(agent), ShouldBeNil)

			agent2 := &model.LocalAgent{
				Name:        "test2",
				Protocol:    "sftp",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:2023",
			}
			So(db.Create(agent2), ShouldBeNil)

			cert2 := &model.Cert{
				Name:        "foo",
				OwnerType:   "local_agents",
				OwnerID:     agent2.ID,
				PublicKey:   []byte("abc"),
				PrivateKey:  []byte("zyx"),
				Certificate: []byte("cert"),
			}
			So(db.Create(cert2), ShouldBeNil)

			Convey("Given a list of new Certificates to import", func() {
				insert := Certificate{
					Name:        "new",
					PublicKey:   "abc",
					PrivateKey:  "zyx",
					Certificate: "cert",
				}
				Certificates := []Certificate{insert}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importCerts with the new "+
						"Certificates on the existing agent", func() {
						err := importCerts(discard, ses, Certificates,
							"local_agents", agent.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the agent should have 1 Certificates", func() {
							var dbCerts []model.Cert
							So(ses.Select(&dbCerts, &database.Filters{
								Conditions: builder.Eq{
									"owner_type": "local_agents",
									"owner_id":   agent.ID,
								},
							}), ShouldBeNil)
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

			Convey("Given a updated Certificate to import", func() {

				insert := Certificate{
					Name:        "foo",
					PublicKey:   "abc",
					PrivateKey:  "zyx",
					Certificate: "cert",
				}
				Certificates := []Certificate{insert}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importCerts with the new "+
						"Certificates on the existing agent", func() {
						err := importCerts(discard, ses, Certificates,
							"local_agents", agent2.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the agent should have 1 Certificates", func() {
							var dbCerts []model.Cert
							So(ses.Select(&dbCerts, &database.Filters{
								Conditions: builder.Eq{
									"owner_type": "local_agents",
									"owner_id":   agent2.ID,
								},
							}), ShouldBeNil)
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
	})
}
