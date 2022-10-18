package backup

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestExportCertificates(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains 1 local agent with a certificate", func() {
			agent := &model.LocalAgent{
				Name:        "server",
				Protocol:    testProtocol,
				ProtoConfig: json.RawMessage(`{}`),
				Address:     "localhost:6666",
			}
			So(db.Insert(agent).Run(), ShouldBeNil)

			cert := &model.Crypto{
				Name:         "test_cert",
				LocalAgentID: utils.NewNullInt64(agent.ID),
				Certificate:  testhelpers.OtherLocalhostCert,
				PrivateKey:   testhelpers.OtherLocalhostKey,
			}
			So(db.Insert(cert).Run(), ShouldBeNil)

			Convey("Given an new Transaction", func() {
				Convey("When calling exportCertificates with the correct argument", func() {
					res, err := exportCertificates(discard(), db, agent)

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
							So(c.PrivateKey, ShouldResemble, string(cert.PrivateKey))
						})
					})
				})

				Convey("When calling exportCertificates with incorrect argument", func() {
					agent.ID++
					res, err := exportCertificates(discard(), db, agent)

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
					Login:        "foo",
					PasswordHash: hash("sesame"),
				}
				So(db.Insert(account).Run(), ShouldBeNil)

				cert1 := &model.Crypto{
					Name:           "cert1",
					LocalAccountID: utils.NewNullInt64(account.ID),
					Certificate:    testhelpers.ClientFooCert,
				}
				So(db.Insert(cert1).Run(), ShouldBeNil)

				cert2 := &model.Crypto{
					Name:           "cert2",
					LocalAccountID: utils.NewNullInt64(account.ID),
					Certificate:    testhelpers.ClientFooCert2,
				}
				So(db.Insert(cert2).Run(), ShouldBeNil)

				Convey("When calling exportCertificates with the correct argument", func() {
					res, err := exportCertificates(discard(), db, account)

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
