package backup

import (
	"encoding/json"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExportCertificates(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c, "ERROR")

		Convey("Given the database contains 1 local agent with a certificate", func() {
			agent := &model.LocalAgent{
				Name:        "test",
				Protocol:    "test",
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:6666",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			cert := &model.Crypto{
				Name:        "test_cert",
				OwnerType:   "local_agents",
				OwnerID:     agent.ID,
				Certificate: testhelpers.LocalhostCert,
				PrivateKey:  testhelpers.LocalhostKey,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			Convey("Given an new Transaction", func() {

				Convey("When calling exportCertificates with the correct argument", func() {
					res, err := exportCertificates(discard, db, "local_agents", agent.ID)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then it should return 1 certificate", func() {
						So(len(res), ShouldEqual, 1)

						Convey("Then the certificate retrieved should be the same "+
							"than in the database", func() {
							c := res[0]
							So(c.Name, ShouldEqual, cert.Name)
							So(c.Certificate, ShouldResemble, cert.Certificate)
							So(c.PublicKey, ShouldResemble, cert.SSHPublicKey)
							So(c.PrivateKey, ShouldResemble, cert.PrivateKey)
						})
					})
				})

				Convey("When calling exportCertificates with incorrect argument", func() {
					res, err := exportCertificates(discard, db, "local_agents", agent.ID+1)

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
				So(db.Insert(account).Run(), ShouldBeNil)

				cert1 := &model.Crypto{
					Name:        "cert1",
					OwnerType:   "local_accounts",
					OwnerID:     account.ID,
					Certificate: testhelpers.ClientCert,
				}
				So(db.Insert(cert1).Run(), ShouldBeNil)

				cert2 := &model.Crypto{
					Name:        "cert2",
					OwnerType:   "local_accounts",
					OwnerID:     account.ID,
					Certificate: testhelpers.ClientCert,
				}
				So(db.Insert(cert2).Run(), ShouldBeNil)

				Convey("When calling exportCertificates with the correct argument", func() {
					res, err := exportCertificates(discard, db, "local_accounts", account.ID)

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
}
