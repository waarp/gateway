package rest

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

const transferURI = "http://localhost:8080" + APIPath + TransfersPath + "/"

func TestAddTransfer(t *testing.T) {
	logger := log.NewLogger("rest_transfer_add_test")

	Convey("Testing the transfer add handler", t, func() {
		db := database.GetTestDatabase()
		handler := createTransfer(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 partner, 1 certificate & 1 account", func() {
			partner := &model.RemoteAgent{
				Name:        "sftp_test",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(partner), ShouldBeNil)

			cert := &model.Cert{
				OwnerType:   partner.TableName(),
				OwnerID:     partner.ID,
				Name:        "sftp_cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(cert), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "login",
				Password:      []byte("password"),
			}
			So(db.Create(account), ShouldBeNil)

			push := model.Rule{Name: "test_push", IsSend: false, Path: "path"}
			So(db.Create(&push), ShouldBeNil)

			trans := &InTransfer{
				Rule:       push.Name,
				Partner:    partner.Name,
				Account:    account.Login,
				IsSend:     true,
				SourcePath: "src",
				DestPath:   "dst",
			}

			Convey("Given a valid new transfer", func() {

				Convey("When calling the handler", func() {
					content, err := json.Marshal(trans)
					So(err, ShouldBeNil)
					body := bytes.NewReader(content)
					r, err := http.NewRequest(http.MethodPost, transferURI, body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 201", func() {
						So(w.Code, ShouldEqual, http.StatusCreated)
					})

					Convey("Then the response body should be empty", func() {
						So(w.Body.String(), ShouldBeBlank)
					})

					Convey("Then the 'Location' header should contain the URI "+
						"of the new transfer", func() {

						location := w.Header().Get("Location")
						So(location, ShouldStartWith, transferURI)
					})

					Convey("Then the new transfer should be inserted in "+
						"the database", func() {
						t, err := trans.ToModel(db)
						So(err, ShouldBeNil)
						So(db.Get(t), ShouldBeNil)
					})
				})
			})

			Convey("Given a new transfer with an invalid rule name", func() {
				trans.Rule = "toto"

				Convey("When calling the handler", func() {
					content, err := json.Marshal(&trans)
					So(err, ShouldBeNil)
					body := bytes.NewReader(content)
					r, err := http.NewRequest(http.MethodPost, transferURI, body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say the partner is invalid", func() {
						So(w.Body.String(), ShouldEqual, "no rule 'toto' found\n")
					})
				})
			})

			Convey("Given a new transfer with an invalid partner name", func() {
				trans.Partner = "toto"

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
						So(w.Body.String(), ShouldEqual, "no partner 'toto' found\n")
					})
				})
			})

			Convey("Given a new transfer with an invalid account name", func() {
				trans.Account = "toto"

				Convey("When calling the handler", func() {
					content, err := json.Marshal(trans)
					So(err, ShouldBeNil)
					body := bytes.NewReader(content)
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say the partner is invalid", func() {
						So(w.Body.String(), ShouldEqual, "no account 'toto' found "+
							"for partner "+partner.Name+"\n")
					})
				})
			})

			Convey("Given the partner does not have a certificate", func() {
				So(db.Delete(cert), ShouldBeNil)

				Convey("When calling the handler", func() {
					content, err := json.Marshal(trans)
					So(err, ShouldBeNil)
					body := bytes.NewReader(content)
					r, err := http.NewRequest(http.MethodPost, "", body)
					So(err, ShouldBeNil)

					handler.ServeHTTP(w, r)

					Convey("Then it should return a code 400", func() {
						So(w.Code, ShouldEqual, http.StatusBadRequest)
					})

					Convey("Then the response body should say a host key is missing", func() {
						So(w.Body.String(), ShouldEqual, "the partner is missing "+
							"an SFTP host key\n")
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
			partner := &model.RemoteAgent{
				Name:        "sftp_test",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022}`),
			}
			So(db.Create(partner), ShouldBeNil)

			cert := &model.Cert{
				OwnerType:   partner.TableName(),
				OwnerID:     partner.ID,
				Name:        "sftp_cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(cert), ShouldBeNil)

			account := &model.RemoteAccount{
				RemoteAgentID: partner.ID,
				Login:         "toto",
				Password:      []byte("titi"),
			}
			So(db.Create(account), ShouldBeNil)

			push := &model.Rule{Name: "test_push", IsSend: false, Path: "path"}
			So(db.Create(push), ShouldBeNil)

			trans := &model.Transfer{
				RuleID:     push.ID,
				AgentID:    partner.ID,
				AccountID:  account.ID,
				SourceFile: "src",
				DestFile:   "dst",
			}
			So(db.Create(trans), ShouldBeNil)

			Convey("Given a request with the valid transfer ID parameter", func() {
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
						jsonObj, err := FromTransfer(db, trans)
						So(err, ShouldBeNil)
						exp, err := json.Marshal(jsonObj)
						So(err, ShouldBeNil)

						So(w.Body.String(), ShouldResemble, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with an invalid transfer ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, transferURI+"1000", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"transfer": "1000"})

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

		expected := map[string][]OutTransfer{}

		Convey("Given a database with 2 transfer", func() {
			p1 := &model.RemoteAgent{
				Name:        "sftp1",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(p1), ShouldBeNil)

			p2 := &model.RemoteAgent{
				Name:        "sftp2",
				Protocol:    "test2",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(p2), ShouldBeNil)

			c1 := &model.Cert{
				OwnerType:   p1.TableName(),
				OwnerID:     p1.ID,
				Name:        "sftp_cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(c1), ShouldBeNil)

			c2 := &model.Cert{
				OwnerType:   p2.TableName(),
				OwnerID:     p2.ID,
				Name:        "sftp_cert",
				PrivateKey:  nil,
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(c2), ShouldBeNil)

			a1 := &model.RemoteAccount{
				RemoteAgentID: p1.ID,
				Login:         "toto",
				Password:      []byte("titi"),
			}
			So(db.Create(a1), ShouldBeNil)

			a2 := &model.RemoteAccount{
				RemoteAgentID: p2.ID,
				Login:         "toto",
				Password:      []byte("titi"),
			}
			So(db.Create(a2), ShouldBeNil)

			r1 := &model.Rule{Name: "rule1", IsSend: false, Path: "path1"}
			So(db.Create(r1), ShouldBeNil)

			r2 := &model.Rule{Name: "rule2", IsSend: false, Path: "path2"}
			So(db.Create(r2), ShouldBeNil)

			t1 := &model.Transfer{
				RuleID:     r1.ID,
				AgentID:    p1.ID,
				AccountID:  a1.ID,
				SourceFile: "src1",
				DestFile:   "dst2",
				Progress:   1,
				TaskNumber: 2,
			}
			So(db.Create(t1), ShouldBeNil)

			t2 := &model.Transfer{
				RuleID:     r2.ID,
				AgentID:    p2.ID,
				AccountID:  a2.ID,
				SourceFile: "src2",
				DestFile:   "dst2",
			}
			So(db.Create(t2), ShouldBeNil)

			t3 := &model.Transfer{
				RuleID:     r2.ID,
				AgentID:    p1.ID,
				AccountID:  a1.ID,
				SourceFile: "src3",
				DestFile:   "dst3",
				Start:      t2.Start.Add(2 * time.Hour),
			}
			So(db.Create(t3), ShouldBeNil)

			trans1, err := FromTransfer(db, t1)
			So(err, ShouldBeNil)
			trans2, err := FromTransfer(db, t2)
			So(err, ShouldBeNil)
			trans3, err := FromTransfer(db, t3)
			So(err, ShouldBeNil)

			Convey("Given a request with no parameters", func() {
				req, err := http.NewRequest(http.MethodGet, "", nil)
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
						expected["transfers"] = []OutTransfer{*trans1, *trans2, *trans3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a valid 'rule' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?rule="+r2.Name, nil)
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
						expected["transfers"] = []OutTransfer{*trans2, *trans3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a valid 'status' parameter", func() {
				req, err := http.NewRequest(http.MethodGet, "?status=PLANNED", nil)
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
						expected["transfers"] = []OutTransfer{*trans1, *trans2, *trans3}
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
						expected["transfers"] = []OutTransfer{*trans3}
						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})
		})
	})
}
