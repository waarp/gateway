package backup

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
	. "github.com/smartystreets/goconvey/convey"
)

func TestImportCerts(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a database with some certificates", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(agent), ShouldBeNil)

			agent2 := &model.LocalAgent{
				Name:        "test2",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
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

			Convey("Given a list of new certificates to import", func() {
				insert := certificate{
					Name:        "new",
					PublicKey:   "abc",
					PrivateKey:  "zyx",
					Certificate: "cert",
				}
				certificates := []certificate{insert}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importCerts with the new Certificates on the existing agent", func() {
						err := importCerts(ses, certificates, "local_agents", agent.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the agent should have 1 certificates", func() {
							dbCerts := []model.Cert{}
							So(ses.Select(&dbCerts, &database.Filters{
								Conditions: builder.Eq{
									"owner_type": "local_agents",
									"owner_id":   agent.ID,
								},
							}), ShouldBeNil)
							So(len(dbCerts), ShouldEqual, 1)

							Convey("Then the certificate should correspond to the one imported", func() {
								So(dbCerts[0].Name, ShouldResemble, insert.Name)
								So(dbCerts[0].PublicKey, ShouldResemble, []byte(insert.PublicKey))
								So(dbCerts[0].PrivateKey, ShouldResemble, []byte(insert.PrivateKey))
								So(dbCerts[0].Certificate, ShouldResemble, []byte(insert.Certificate))
							})
						})
					})

				})
			})

			Convey("Given a updated certificate to import", func() {

				insert := certificate{
					Name:        "foo",
					PublicKey:   "abc",
					PrivateKey:  "zyx",
					Certificate: "cert",
				}
				certificates := []certificate{insert}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importCerts with the new Certificates on the existing agent", func() {
						err := importCerts(ses, certificates, "local_agents", agent2.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then the agent should have 1 certificates", func() {
							dbCerts := []model.Cert{}
							So(ses.Select(&dbCerts, &database.Filters{
								Conditions: builder.Eq{
									"owner_type": "local_agents",
									"owner_id":   agent2.ID,
								},
							}), ShouldBeNil)
							So(len(dbCerts), ShouldEqual, 1)

							Convey("Then the certificate should correspond to the one imported", func() {
								So(dbCerts[0].Name, ShouldResemble, insert.Name)
								So(dbCerts[0].PublicKey, ShouldResemble, []byte(insert.PublicKey))
								So(dbCerts[0].PrivateKey, ShouldResemble, []byte(insert.PrivateKey))
								So(dbCerts[0].Certificate, ShouldResemble, []byte(insert.Certificate))
							})
						})
					})

				})
			})

			Convey("Given a partially updated certificate to import", func() {

				insert := certificate{
					Name:       "foo",
					PublicKey:  "abc",
					PrivateKey: "zyx",
				}
				certificates := []certificate{insert}

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling the importCerts with the new Certificates on the existing agent", func() {
						err := importCerts(ses, certificates, "local_agents", agent2.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)

							Convey("Then the agent should have 1 certificates", func() {
								dbCerts := []model.Cert{}
								So(ses.Select(&dbCerts, &database.Filters{
									Conditions: builder.Eq{
										"owner_type": "local_agents",
										"owner_id":   agent2.ID,
									},
								}), ShouldBeNil)
								So(len(dbCerts), ShouldEqual, 1)

								Convey("Then the certificate should correspond to the one imported", func() {
									So(dbCerts[0].Name, ShouldResemble, insert.Name)
									So(dbCerts[0].PublicKey, ShouldResemble, []byte(insert.PublicKey))
									So(dbCerts[0].PrivateKey, ShouldResemble, []byte(insert.PrivateKey))
									So(dbCerts[0].Certificate, ShouldResemble, cert2.Certificate)
								})
							})
						})
					})
				})
			})
		})
	})
}
