package backup

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExportCertificates(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given the database contains 1 local agent with a certificate", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(agent), ShouldBeNil)

			cert := &model.Cert{
				Name:        "test_cert",
				OwnerType:   "local_agents",
				OwnerID:     agent.ID,
				Certificate: []byte("cert"),
				PublicKey:   []byte("public"),
				PrivateKey:  []byte("private"),
			}
			So(db.Create(cert), ShouldBeNil)

			Convey("Given a new Transaction", func() {
				ses, err := db.BeginTransaction()
				So(err, ShouldBeNil)

				defer ses.Rollback()

				Convey("When calling exportCertificates with the correct argument", func() {
					res, err := exportCertificates(discard, ses, "local_agents", agent.ID)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should return 1 certificate", func() {
						So(len(res), ShouldEqual, 1)

						Convey("Then the certificate retrieved should be the same "+
							"than in the database", func() {
							c := res[0]
							So(c.Name, ShouldEqual, cert.Name)
							So([]byte(c.Certificate), ShouldResemble, cert.Certificate)
							So([]byte(c.PublicKey), ShouldResemble, cert.PublicKey)
							So([]byte(c.PrivateKey), ShouldResemble, cert.PrivateKey)
						})
					})
				})

				Convey("When calling exportCertificates with incorrect argument", func() {
					res, err := exportCertificates(discard, ses, "local_agents", agent.ID+1)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should return 0 certificate", func() {
						So(len(res), ShouldEqual, 0)
					})
				})
			})

			Convey("Given the database contains 1 local account with 2 certificates", func() {
				account := &model.LocalAccount{
					LocalAgentID: agent.ID,
					Login:        "test",
					Password:     []byte("pwd"),
				}
				So(db.Create(account), ShouldBeNil)

				cert1 := &model.Cert{
					Name:        "cert1",
					OwnerType:   "local_accounts",
					OwnerID:     account.ID,
					Certificate: []byte("cert"),
					PublicKey:   []byte("public"),
					PrivateKey:  []byte("private"),
				}
				So(db.Create(cert1), ShouldBeNil)

				cert2 := &model.Cert{
					Name:        "cert2",
					OwnerType:   "local_accounts",
					OwnerID:     account.ID,
					Certificate: []byte("cert"),
					PublicKey:   []byte("public"),
					PrivateKey:  []byte("private"),
				}
				So(db.Create(cert2), ShouldBeNil)

				Convey("Given a new Transaction", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					defer ses.Rollback()

					Convey("When calling exportCertificates with the correct argument", func() {
						res, err := exportCertificates(discard, ses, "local_accounts", account.ID)

						Convey("Then it should return no error", func() {
							So(err, ShouldBeNil)
						})

						Convey("Then it should return 2 certificates", func() {
							So(len(res), ShouldEqual, 2)
						})

					})
				})

			})
		})
	})
}
