package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAddTransfer(t *testing.T) {
	logger := log.NewLogger("rest_transfer_add_test")

	Convey("Testing the REST transfer launcher", t, func() {
		db := database.GetTestDatabase()
		handler := addTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner, 1 certificate & 1 account", func() {
			partner := model.RemoteAgent{
				Name:        "sftp_test",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			So(db.Create(&partner), ShouldBeNil)

			cert := model.Cert{
				OwnerType:   (&partner).TableName(),
				OwnerID:     partner.ID,
				Name:        "sftp_cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(&cert), ShouldBeNil)

			account := model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      []byte("titi"),
			}
			So(db.Create(&account), ShouldBeNil)

			push := model.Rule{Name: "test_push", IsGet: false}
			So(db.Create(&push), ShouldBeNil)

			trans := model.Transfer{
				RuleID:      push.ID,
				RemoteID:    partner.ID,
				AccountID:   account.ID,
				Source:      "src/test/path",
				Destination: "dst/test/path",
			}

			Convey("Given a valid new transfer", func() {

				Convey("When calling the handler", func() {
					content, err := json.Marshal(&trans)
					So(err, ShouldBeNil)
					body := bytes.NewReader(content)
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 201", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeBlank)
					})
				})
			})

			Convey("Given a new transfer with an invalid ruleID", func() {
				trans.RuleID = 1000

				Convey("When calling the handler", func() {
					content, err := json.Marshal(&trans)
					So(err, ShouldBeNil)
					body := bytes.NewReader(content)
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say the partner is invalid", func() {
						So(w.Body.String(), ShouldEqual, fmt.Sprintf("The rule "+
							"%v does not exist\n", trans.RuleID))
					})
				})
			})

			Convey("Given a new transfer with an invalid partnerID", func() {
				trans.RemoteID = 1000

				Convey("When calling the handler", func() {
					content, err := json.Marshal(&trans)
					So(err, ShouldBeNil)
					body := bytes.NewReader(content)
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say the partner is invalid", func() {
						So(w.Body.String(), ShouldEqual, fmt.Sprintf("The partner "+
							"%v does not exist\n", trans.RemoteID))
					})
				})
			})

			Convey("Given a new transfer with an invalid account", func() {
				trans.AccountID = 1000

				Convey("When calling the handler", func() {
					content, err := json.Marshal(&trans)
					So(err, ShouldBeNil)
					body := bytes.NewReader(content)
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say the partner is invalid", func() {
						So(w.Body.String(), ShouldEqual, fmt.Sprintf("The agent "+
							"%v does not have an account %v\n", trans.RemoteID,
							trans.AccountID))
					})
				})
			})

			Convey("Given the partner does not have a certificate", func() {
				So(db.Delete(&cert), ShouldBeNil)

				Convey("When calling the handler", func() {
					content, err := json.Marshal(&trans)
					So(err, ShouldBeNil)
					body := bytes.NewReader(content)
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say no certificates were found", func() {
						So(w.Body.String(), ShouldEqual, fmt.Sprintf("No "+
							"certificate found for agent %v\n", trans.RemoteID))
					})
				})
			})
		})
	})
}
