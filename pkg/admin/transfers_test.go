package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const transferURI = APIPath + TransfersPath + "/"

func TestAddTransfer(t *testing.T) {
	logger := log.NewLogger("rest_transfer_add_test")

	Convey("Testing the transfer add handler", t, func() {
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
				RuleID:     push.ID,
				RemoteID:   partner.ID,
				AccountID:  account.ID,
				SourcePath: "src/test/path",
				DestPath:   "dst/test/path",
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

func TestGetTransfer(t *testing.T) {
	logger := log.NewLogger("rest_transfer_get_test")

	Convey("Testing the transfer get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 transfer", func() {
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
				RuleID:     push.ID,
				RemoteID:   partner.ID,
				AccountID:  account.ID,
				SourcePath: "src/test/path",
				DestPath:   "dst/test/path",
			}
			So(db.Create(&trans), ShouldBeNil)

			Convey("Given a request with the valid history ID parameter", func() {
				id := strconv.FormatUint(trans.ID, 10)
				req, err := http.NewRequest(http.MethodGet, transferURI+id, nil)
				So(err, ShouldBeNil)
				req = mux.SetURLVars(req, map[string]string{"transfer": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested transfer "+
						"in JSON format", func() {
						exp, err := json.Marshal(&trans)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with an invalid history ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, transferURI+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"history": "1000"})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply with a 'Not Found' error", func() {
						So(w.Code, ShouldEqual, http.StatusNotFound)
					})
				})
			})
		})
	})
}

func TestListTransfer(t *testing.T) {
	logger := log.NewLogger("rest_transfer_list_test")

	Convey("Testing the transfer list handler", t, func() {
		db := database.GetTestDatabase()
		handler := listTransfers(logger, db)
		w := httptest.NewRecorder()

		expected := map[string][]interface{}{}

		Convey("Given a database with 2 transfer", func() {
			p1 := model.RemoteAgent{
				Name:        "sftp_test",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			So(db.Create(&p1), ShouldBeNil)

			p2 := model.RemoteAgent{
				Name:        "sftp2",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			So(db.Create(&p2), ShouldBeNil)

			c1 := model.Cert{
				OwnerType:   (&p1).TableName(),
				OwnerID:     p1.ID,
				Name:        "sftp_cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(&c1), ShouldBeNil)

			c2 := model.Cert{
				OwnerType:   (&p2).TableName(),
				OwnerID:     p2.ID,
				Name:        "sftp_cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(&c2), ShouldBeNil)

			a1 := model.RemoteAccount{
				RemoteAgentID: p1.ID,
				Login:         "toto",
				Password:      []byte("titi"),
			}
			So(db.Create(&a1), ShouldBeNil)

			a2 := model.RemoteAccount{
				RemoteAgentID: p2.ID,
				Login:         "toto",
				Password:      []byte("titi"),
			}
			So(db.Create(&a2), ShouldBeNil)

			r1 := model.Rule{Name: "test_push", IsGet: false}
			So(db.Create(&r1), ShouldBeNil)

			r2 := model.Rule{Name: "rule2", IsGet: false}
			So(db.Create(&r2), ShouldBeNil)

			t1 := model.Transfer{
				RuleID:     r1.ID,
				RemoteID:   p1.ID,
				AccountID:  a1.ID,
				SourcePath: "src/test/path",
				DestPath:   "dst/test/path",
			}
			So(db.Create(&t1), ShouldBeNil)

			t2 := model.Transfer{
				RuleID:     r2.ID,
				RemoteID:   p2.ID,
				AccountID:  a2.ID,
				SourcePath: "src/test/path",
				DestPath:   "dst/test/path",
			}
			So(db.Create(&t2), ShouldBeNil)

			t3 := model.Transfer{
				RuleID:     r2.ID,
				RemoteID:   p1.ID,
				AccountID:  a1.ID,
				SourcePath: "src/test/path",
				DestPath:   "dst/test/path",
				Start:      t2.Start.Add(2 * time.Hour),
			}
			So(db.Create(&t3), ShouldBeNil)

			Convey("Given a request with a valid 'remoteID' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("?remote=%d", p1.ID), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return 2 transfer", func() {
						expected["transfers"] = []interface{}{t1, t3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a valid 'accountID' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("?account=%d", p2.ID), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return 1 transfer", func() {
						expected["transfers"] = []interface{}{t2}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a valid 'ruleID' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("?rule=%d", r2.ID), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return 2 transfer", func() {
						expected["transfers"] = []interface{}{t2, t3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a valid 'status' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("?status=%s", "PLANNED"), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return all transfer", func() {
						expected["transfers"] = []interface{}{t1, t2, t3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a valid 'start' parameter", func() {
				date := t3.Start.Add(-time.Minute)
				req, err := http.NewRequest(http.MethodGet,
					fmt.Sprintf("?start=%s", url.QueryEscape(date.Format(time.RFC3339))), nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, req)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})
					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then it should return all transfer", func() {
						expected["transfers"] = []interface{}{t3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})
		})
	})
}
