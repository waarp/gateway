package rest

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
	"github.com/gorilla/mux"
	. "github.com/smartystreets/goconvey/convey"
)

func remAgentCertURI(partner, cert string) string {
	return fmt.Sprintf("http://localhost:8080/api/partners/%s/certificates/%s", partner, cert)
}

func TestGetCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_get_test")

	Convey("Given the certificate get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getCertificate(logger, db)
		w := httptest.NewRecorder()

		_, err := db.Query("SELECT * FROM certificates")
		So(err, ShouldBeNil)

		Convey("Given a database with 1 certificate", func() {
			parent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(parent), ShouldBeNil)

			expected := &model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "expected",
				PrivateKey:  []byte("private key"),
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(expected), ShouldBeNil)

			Convey("Given a request with a valid certificate name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
					"certificate": expected.Name})

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

						exp, err := json.Marshal(FromCert(expected))

						So(err, ShouldBeNil)
						So(w.Body.String(), ShouldEqual, string(exp)+"\n")
					})
				})
			})

			Convey("Given a request with a non-existing certificate name parameter", func() {
				r, err := http.NewRequest(http.MethodGet, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
					"certificate": "toto"})

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
	logger := log.NewLogger("rest_cert_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]OutCert) {
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
		expected := map[string][]OutCert{}

		Convey("Given a database with 4 parents", func() {
			p1 := &model.LocalAgent{
				Name:        "local_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			p2 := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(p1), ShouldBeNil)
			So(db.Create(p2), ShouldBeNil)

			p3 := &model.LocalAccount{
				Login:        "local_account",
				LocalAgentID: p1.ID,
				Password:     []byte("local_account"),
			}
			p4 := &model.RemoteAccount{
				Login:         "remote_account",
				RemoteAgentID: p2.ID,
				Password:      []byte("remote_account"),
			}
			So(db.Create(p3), ShouldBeNil)
			So(db.Create(p4), ShouldBeNil)

			Convey("Given a database with 4 certificates", func() {
				c1 := &model.Cert{
					OwnerType:   p3.TableName(),
					OwnerID:     p3.ID,
					Name:        "local_account_cert1",
					PrivateKey:  []byte("private key 1"),
					PublicKey:   []byte("public key 1"),
					Certificate: []byte("certificate 1"),
				}
				c2 := &model.Cert{
					OwnerType:   p2.TableName(),
					OwnerID:     p2.ID,
					Name:        "remote_agent_cert",
					PrivateKey:  []byte("private key 2"),
					PublicKey:   []byte("public key 2"),
					Certificate: []byte("certificate 2"),
				}
				c3 := &model.Cert{
					OwnerType:   p3.TableName(),
					OwnerID:     p3.ID,
					Name:        "local_account_cert2",
					PrivateKey:  []byte("private key 3"),
					PublicKey:   []byte("public key 3"),
					Certificate: []byte("certificate 3"),
				}
				c4 := &model.Cert{
					OwnerType:   p4.TableName(),
					OwnerID:     p4.ID,
					Name:        "remote_account_cert",
					PrivateKey:  []byte("private key 4"),
					PublicKey:   []byte("public key 4"),
					Certificate: []byte("certificate 4"),
				}
				So(db.Create(c1), ShouldBeNil)
				So(db.Create(c2), ShouldBeNil)
				So(db.Create(c3), ShouldBeNil)
				So(db.Create(c4), ShouldBeNil)

				cert1 := *FromCert(c1)
				cert2 := *FromCert(c2)
				cert3 := *FromCert(c3)
				cert4 := *FromCert(c4)

				Convey("Given a request with no parameters", func() {
					r, err := http.NewRequest(http.MethodGet, "", nil)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": p1.Name,
						"local_account": p3.Login})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{cert1, cert3}
						check(w, expected)
					})
				})

				Convey("Given a request with a different owner", func() {
					r, err := http.NewRequest(http.MethodGet, "", nil)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": p2.Name,
						"remote_account": p4.Login})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{cert4}
						check(w, expected)
					})
				})

				Convey("Given a request with another different owner", func() {
					r, err := http.NewRequest(http.MethodGet, "", nil)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": p2.Name})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{cert2}
						check(w, expected)
					})
				})

				Convey("Given a request with a limit parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": p1.Name,
						"local_account": p3.Login})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{cert1}
						check(w, expected)
					})
				})

				Convey("Given a request with a offset parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": p1.Name,
						"local_account": p3.Login})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{cert3}
						check(w, expected)
					})
				})

				Convey("Given a request with a sort & order parameters", func() {
					r, err := http.NewRequest(http.MethodGet, "?sort=name-", nil)
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"local_agent": p1.Name,
						"local_account": p3.Login})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{cert3, cert1}
						check(w, expected)
					})
				})
			})
		})
	})
}

func TestCreateCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_create_logger")

	Convey("Given the certificate creation handler", t, func() {
		db := database.GetTestDatabase()
		handler := createCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 certificate", func() {
			parent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(parent), ShouldBeNil)

			existing := &model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "existing",
				PrivateKey:  []byte("private key"),
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a new certificate to insert in the database", func() {
				newCert := &InCert{
					Name:        "new_cert",
					PrivateKey:  []byte("new_private key"),
					PublicKey:   []byte("new_public key"),
					Certificate: []byte("new_certificate"),
				}

				Convey("Given that the new account is valid for insertion", func() {
					body, err := json.Marshal(newCert)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, remAgentCertURI(
						parent.Name, ""), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new account", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, remAgentCertURI(parent.Name,
								newCert.Name))
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the new certificate should be inserted "+
							"in the database", func() {
							exist, err := db.Exists(newCert.toModel(parent.TableName(),
								parent.ID))

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})

						Convey("Then the existing certificate should still be "+
							"present as well", func() {
							exist, err := db.Exists(existing)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})
			})
		})
	})
}

func TestDeleteCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_delete_test")

	Convey("Given the certificate deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 certificate", func() {
			parent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(parent), ShouldBeNil)

			existing := &model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "existing",
				PrivateKey:  []byte("private key"),
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(existing), ShouldBeNil)

			Convey("Given a request with the valid certificate name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
					"certificate": existing.Name})

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

						exist, err := db.Exists(existing)
						So(err, ShouldBeNil)
						So(exist, ShouldBeFalse)
					})
				})
			})

			Convey("Given a request with a non-existing certificate name parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, "", nil)
				So(err, ShouldBeNil)
				r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
					"certificate": "toto"})

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
	logger := log.NewLogger("rest_cert_update_logger")

	Convey("Given the certificate updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 certificates", func() {
			parent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "test",
				ProtoConfig: []byte(`{}`),
			}
			So(db.Create(parent), ShouldBeNil)

			old := &model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "old",
				PrivateKey:  []byte("private key"),
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(old), ShouldBeNil)

			other := &model.Cert{
				OwnerType:   parent.TableName(),
				OwnerID:     parent.ID,
				Name:        "other",
				PrivateKey:  []byte("private key"),
				PublicKey:   []byte("public key"),
				Certificate: []byte("certificate"),
			}
			So(db.Create(other), ShouldBeNil)

			Convey("Given new values to update the certificate with", func() {
				update := &InCert{
					Name:        "new name",
					PrivateKey:  []byte("new private key"),
					PublicKey:   []byte("new public key"),
					Certificate: []byte("new certificate"),
				}
				body, err := json.Marshal(update)
				So(err, ShouldBeNil)

				Convey("Given a valid certificate name parameter", func() {
					r, err := http.NewRequest(http.MethodPatch, remAgentCertURI(
						parent.Name, old.Name), bytes.NewReader(body))
					So(err, ShouldBeNil)
					r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
						"certificate": old.Name})

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated certificate", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, remAgentCertURI(parent.Name,
								update.Name))
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the certificate should have been updated", func() {
							result := &model.Cert{
								ID:          old.ID,
								OwnerID:     old.OwnerID,
								OwnerType:   old.OwnerType,
								Name:        update.Name,
								PrivateKey:  update.PrivateKey,
								PublicKey:   update.PublicKey,
								Certificate: update.Certificate,
							}
							So(db.Get(result), ShouldBeNil)
						})
					})
				})

				Convey("Given an invalid certificate name parameter", func() {

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, remAgentCertURI(
							parent.Name, "toto"), bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"remote_agent": parent.Name,
							"certificate": "toto"})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the certificate was not found", func() {
							So(w.Body.String(), ShouldEqual, "certificate 'toto' not found\n")
						})

						Convey("Then the old certificate should still exist", func() {
							So(db.Get(old), ShouldBeNil)
						})
					})
				})
			})
		})
	})
}
