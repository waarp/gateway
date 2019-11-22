package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

const certURI = APIPath + CertificatesPath + "/"

func TestGetCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_get_test", logConf)

	Convey("Given the certificate get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getCertificate(logger, db)
		w := httptest.NewRecorder()

		_, err := db.Query("SELECT * FROM certificates")
		So(err, ShouldBeNil)

		Convey("Given a database with 1 certificate", func() {
			parent := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			expected := model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "expected",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate"),
			}
			err = db.Create(&expected)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid certificate ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"certificate": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'OK'", func() {
						So(w.Code, ShouldEqual, http.StatusOK)
					})

					Convey("Then the 'Content-Type' header should contain 'application/json'", func() {
						contentType := w.Header().Get("Content-Type")

						So(contentType, ShouldEqual, "application/json")
					})

					Convey("Then the body should contain the requested certificate "+
						"in JSON format", func() {

						exp, err := json.Marshal(expected)

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing certificate ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"certificate": "1000"})

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

func TestListCerts(t *testing.T) {
	logger := log.NewLogger("rest_cert_list_test", logConf)

	check := func(w *httptest.ResponseRecorder, expected map[string][]model.Cert) {
		Convey("Then it should reply 'OK'", func() {
			So(w.Code, ShouldEqual, http.StatusOK)
		})

		Convey("Then the 'Content-Type' header should contain "+
			"'application/json'", func() {
			contentType := w.Header().Get("Content-Type")

			So(contentType, ShouldEqual, "application/json")
		})

		Convey("Then the response body should contain an array "+
			"of the requested certificates in JSON format", func() {

			exp, err := json.Marshal(expected)

			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldEqual, string(exp)+"\n")
		})
	}

	Convey("Given the certificate listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listCertificates(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]model.Cert{}

		Convey("Given a database with 4 parents", func() {
			localAgentParent := model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&localAgentParent)
			So(err, ShouldBeNil)

			remoteAgentParent := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err = db.Create(&remoteAgentParent)
			So(err, ShouldBeNil)

			localAccountParent := model.LocalAccount{
				Login:        "local_account",
				LocalAgentID: localAgentParent.ID,
				Password:     []byte("local_account"),
			}
			err = db.Create(&localAccountParent)
			So(err, ShouldBeNil)

			remoteAccountParent := model.RemoteAccount{
				Login:         "remote_account",
				RemoteAgentID: remoteAgentParent.ID,
				Password:      []byte("remote_account"),
			}
			err = db.Create(&remoteAccountParent)
			So(err, ShouldBeNil)

			Convey("Given a database with 4 certificates", func() {
				localAgentCert := model.Cert{
					OwnerType:   localAgentParent.TableName(),
					OwnerID:     localAgentParent.ID,
					Name:        "local_agent_cert",
					PrivateKey:  []byte("private_key"),
					PublicKey:   []byte("public_key"),
					Certificate: []byte("certificate"),
				}
				err = db.Create(&localAgentCert)
				So(err, ShouldBeNil)

				remoteAgentCert := model.Cert{
					OwnerType:   remoteAgentParent.TableName(),
					OwnerID:     remoteAgentParent.ID,
					Name:        "remote_agent_cert",
					PrivateKey:  []byte("private_key"),
					PublicKey:   []byte("public_key"),
					Certificate: []byte("certificate"),
				}
				err = db.Create(&remoteAgentCert)
				So(err, ShouldBeNil)

				localAccountCert := model.Cert{
					OwnerType:   localAccountParent.TableName(),
					OwnerID:     localAccountParent.ID,
					Name:        "local_account_cert",
					PrivateKey:  []byte("private_key"),
					PublicKey:   []byte("public_key"),
					Certificate: []byte("certificate"),
				}
				err = db.Create(&localAccountCert)
				So(err, ShouldBeNil)

				remoteAccountCert := model.Cert{
					OwnerType:   remoteAccountParent.TableName(),
					OwnerID:     remoteAccountParent.ID,
					Name:        "remote_account_cert",
					PrivateKey:  []byte("private_key"),
					PublicKey:   []byte("public_key"),
					Certificate: []byte("certificate"),
				}
				err = db.Create(&remoteAccountCert)
				So(err, ShouldBeNil)

				Convey("Given a request with with no parameters", func() {
					r, err := http.NewRequest(http.MethodGet, "", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []model.Cert{localAccountCert,
							localAgentCert, remoteAccountCert, remoteAgentCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a limit parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []model.Cert{localAccountCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a offset parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []model.Cert{localAgentCert,
							remoteAccountCert, remoteAgentCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a sort & order parameters", func() {
					r, err := http.NewRequest(http.MethodGet, "?sortby=name&order=desc", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []model.Cert{remoteAgentCert,
							remoteAccountCert, localAgentCert, localAccountCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a local account parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?local_accounts=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []model.Cert{localAccountCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a local agent parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?local_agents=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []model.Cert{localAgentCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a remote account parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?remote_accounts=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []model.Cert{remoteAccountCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a remote agent parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?remote_agents=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []model.Cert{remoteAgentCert}
						check(w, expected)
					})
				})
				Convey("Given a request with a remote agent & remote accounts parameters", func() {
					r, err := http.NewRequest(http.MethodGet, "?remote_agents=1&remote_accounts=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []model.Cert{remoteAccountCert,
							remoteAgentCert}
						check(w, expected)
					})
				})
			})
		})
	})
}

func TestCreateCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_create_logger", logConf)

	Convey("Given the certificate creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 certificate", func() {
			parent := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "existing",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate"),
			}
			err = db.Create(&existing)
			So(err, ShouldBeNil)

			Convey("Given a new certificate to insert in the database", func() {
				newCert := model.Cert{
					OwnerType:   parent.TableName(),
					OwnerID:     parent.ID,
					Name:        "new_cert",
					PrivateKey:  []byte("new_private_key"),
					PublicKey:   []byte("new_public_key"),
					Certificate: []byte("new_certificate"),
				}

				Convey("Given that the new account is valid for insertion", func() {
					body, err := json.Marshal(newCert)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new account", func() {

							location := w.Header().Get("Location")
							So(location, ShouldStartWith, certURI)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new certificate should be inserted "+
							"in the database", func() {
							exist, err := db.Exists(&newCert)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the existing certificate should still be "+
							"present as well", func() {
							exist, err := db.Exists(&existing)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})

				Convey("Given that the new certificate has an ID", func() {
					newCert.ID = existing.ID

					body, err := json.Marshal(newCert)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain "+
							"a message stating that the ID already exist", func() {

							So(w.Body.String(), ShouldEqual, "The certificate's "+
								"ID cannot be entered manually\n")
						})

						Convey("Then the new certificate should NOT be "+
							"inserted in the database", func() {
							exist, err := db.Exists(&newCert)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new certificate's name already exist", func() {
					newCert.Name = existing.Name

					body, err := json.Marshal(newCert)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the name already exist", func() {

							So(w.Body.String(), ShouldEqual, "A certificate "+
								"with the same name '"+existing.Name+"' already exist\n")
						})

						Convey("Then the new certificate should NOT be "+
							"inserted in the database", func() {
							exist, err := db.Exists(&newCert)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})

				Convey("Given that the new certificate's accountID type "+
					"is not a valid one", func() {
					newCert.OwnerID = 1000

					body, err := json.Marshal(newCert)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, "", bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the accountID is not valid", func() {

							So(w.Body.String(), ShouldEqual, "No "+newCert.OwnerType+
								" found with ID '1000'\n")
						})

						Convey("Then the new certificate should NOT be "+
							"inserted in the database", func() {
							exist, err := db.Exists(&newCert)

							So(err, ShouldBeNil)
							So(exist, ShouldBeFalse)
						})
					})
				})
			})
		})
	})
}

func TestDeleteCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_delete_test", logConf)

	Convey("Given the certificate deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 certificate", func() {
			parent := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "existing",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate"),
			}
			err = db.Create(&existing)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid certificate ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"certificate": id})

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					Convey("Then it should reply 'No Content'", func() {
						So(w.Code, ShouldEqual, http.StatusNoContent)
					})

					Convey("Then the body should be empty", func() {
						So(w.Body.String(), ShouldBeEmpty)
					})

					Convey("Then the certificate should no longer be present "+
						"in the database", func() {

						exist, err := db.Exists(&existing)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})

			Convey("Given a request with a non-existing certificate ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"certificate": "1000"})

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

func TestUpdateCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_update_logger", logConf)

	Convey("Given the certificate updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 certificates", func() {
			parent := model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "old",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate"),
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)

			other := model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "other",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate"),
			}
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the certificate with", func() {

				Convey("Given a new name", func() {
					update := struct{ Name string }{Name: "update"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					expected := model.Cert{
						OwnerType:   old.OwnerType,
						OwnerID:     old.OwnerID,
						Name:        update.Name,
						PrivateKey:  old.PrivateKey,
						PublicKey:   old.PublicKey,
						Certificate: old.Certificate,
					}

					checkValidUpdate(db, w, http.MethodPatch, certURI, id,
						"certificate", body, handler, &old, &expected)
				})

				Convey("Given an already existing name", func() {
					update := struct{ Name string }{Name: other.Name}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "A certificate with the same name '" + update.Name +
						"' already exist\n"
					checkInvalidUpdate(db, handler, w, body, certURI, id,
						"certificate", &old, msg)
				})
			})
		})
	})
}

func TestReplaceCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_replace_logger", logConf)

	Convey("Given the certificate replacing handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 certificates", func() {
			parent := model.RemoteAgent{
				Name:        "remote_account",
				Protocol:    "sftp",
				ProtoConfig: []byte("{}"),
			}
			err := db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "old",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate"),
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)

			other := model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "other",
				PrivateKey:  []byte("private_key"),
				PublicKey:   []byte("public_key"),
				Certificate: []byte("certificate"),
			}
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := fmt.Sprint(old.ID)

			replace := model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "replace",
				PrivateKey:  []byte("new_private_key"),
				PublicKey:   []byte("new_public_key"),
				Certificate: []byte("new_certificate"),
			}

			Convey("Given a valid new certificate", func() {
				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				expected := model.Cert{
					ID:          old.ID,
					OwnerType:   replace.OwnerType,
					OwnerID:     replace.OwnerID,
					Name:        replace.Name,
					PrivateKey:  replace.PrivateKey,
					PublicKey:   replace.PublicKey,
					Certificate: replace.Certificate,
				}

				Convey("Given a valid certificate ID parameter", func() {
					checkValidUpdate(db, w, http.MethodPut, certURI,
						id, "certificate", body, handler, &old, &expected)
				})

				Convey("Given a non-existing certificate ID parameter", func() {

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPut, "", bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"certificate": "1000"})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Not Found' error", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})
					})
				})
			})
		})
	})
}
