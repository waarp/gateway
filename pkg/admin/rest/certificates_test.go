package rest

import (
	"bytes"
	"encoding/json"
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

const certURI = "http://localhost:8080" + APIPath + CertificatesPath + "/"

func TestGetCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_get_test", logConf)

	Convey("Given the certificate get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getCertificate(logger, db)
		w := httptest.NewRecorder()

		_, err := db.Query("SELECT * FROM certificates")
		So(err, ShouldBeNil)

		Convey("Given a database with 1 certificate", func() {
			parent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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

						exp, err := json.Marshal(fromCert(expected))

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
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
			}
			p2 := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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
					OwnerType:   p1.TableName(),
					OwnerID:     p1.ID,
					Name:        "local_agent_cert",
					PrivateKey:  []byte("private key"),
					PublicKey:   []byte("public key"),
					Certificate: []byte("certificate"),
				}
				c2 := &model.Cert{
					OwnerType:   p2.TableName(),
					OwnerID:     p2.ID,
					Name:        "remote_agent_cert",
					PrivateKey:  []byte("private key"),
					PublicKey:   []byte("public key"),
					Certificate: []byte("certificate"),
				}
				c3 := &model.Cert{
					OwnerType:   p3.TableName(),
					OwnerID:     p3.ID,
					Name:        "local_account_cert",
					PrivateKey:  []byte("private key"),
					PublicKey:   []byte("public key"),
					Certificate: []byte("certificate"),
				}
				c4 := &model.Cert{
					OwnerType:   p4.TableName(),
					OwnerID:     p4.ID,
					Name:        "remote_account_cert",
					PrivateKey:  []byte("private key"),
					PublicKey:   []byte("public key"),
					Certificate: []byte("certificate"),
				}
				So(db.Create(c1), ShouldBeNil)
				So(db.Create(c2), ShouldBeNil)
				So(db.Create(c3), ShouldBeNil)
				So(db.Create(c4), ShouldBeNil)

				localAgentCert := *fromCert(c1)
				remoteAgentCert := *fromCert(c2)
				localAccountCert := *fromCert(c3)
				remoteAccountCert := *fromCert(c4)

				Convey("Given a request with with no parameters", func() {
					r, err := http.NewRequest(http.MethodGet, "", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{localAccountCert,
							localAgentCert, remoteAccountCert, remoteAgentCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a limit parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?limit=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{localAccountCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a offset parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?offset=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{localAgentCert,
							remoteAccountCert, remoteAgentCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a sort & order parameters", func() {
					r, err := http.NewRequest(http.MethodGet, "?sort=name-", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{remoteAgentCert,
							remoteAccountCert, localAgentCert, localAccountCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a local account parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?local_account=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{localAccountCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a server parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?server=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{localAgentCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a remote account parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?remote_account=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{remoteAccountCert}
						check(w, expected)
					})
				})

				Convey("Given a request with a partner parameter", func() {
					r, err := http.NewRequest(http.MethodGet, "?partner=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{remoteAgentCert}
						check(w, expected)
					})
				})
				Convey("Given a request with a partner & remote accounts parameters", func() {
					r, err := http.NewRequest(http.MethodGet, "?partner=1&remote_account=1", nil)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						expected["certificates"] = []OutCert{remoteAccountCert,
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
			parent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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
					OwnerType:   parent.TableName(),
					OwnerID:     parent.ID,
					Name:        "new_cert",
					PrivateKey:  []byte("new_private key"),
					PublicKey:   []byte("new_public key"),
					Certificate: []byte("new_certificate"),
				}

				Convey("Given that the new account is valid for insertion", func() {
					body, err := json.Marshal(newCert)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, certURI, bytes.NewReader(body))

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
							exist, err := db.Exists(newCert.toModel())

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
	logger := log.NewLogger("rest_cert_delete_test", logConf)

	Convey("Given the certificate deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 certificate", func() {
			parent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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

						exist, err := db.Exists(existing)
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
			parent := &model.RemoteAgent{
				Name:        "remote_agent",
				Protocol:    "sftp",
				ProtoConfig: []byte(`{"address":"localhost","port":2022,"root":"toto"}`),
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

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the certificate with", func() {

				Convey("Given a new name", func() {
					update := &InCert{
						Name:        "new name",
						PrivateKey:  []byte("new private key"),
						PublicKey:   []byte("new public key"),
						Certificate: []byte("new certificate"),
					}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, certURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"certificate": id})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain "+
							"the URI of the updated certificate", func() {

							location := w.Header().Get("Location")
							So(location, ShouldEqual, certURI+id)
						})

						Convey("Then the response body should be empty", func() {
							So(w.Body.String(), ShouldBeEmpty)
						})

						Convey("Then the certificate should have been updated", func() {
							result := &model.Cert{ID: old.ID}
							err := db.Get(result)
							So(err, ShouldBeNil)

							So(result.Name, ShouldEqual, update.Name)
							So(result.OwnerID, ShouldEqual, old.OwnerID)
							So(result.OwnerType, ShouldEqual, old.OwnerType)
							So(result.PrivateKey, ShouldResemble, update.PrivateKey)
							So(result.PublicKey, ShouldResemble, update.PublicKey)
							So(result.Certificate, ShouldResemble, update.Certificate)
						})
					})
				})

				Convey("Given an invalid certificate ID", func() {
					update := &InCert{
						Name:        "new name",
						PrivateKey:  []byte("new private key"),
						PublicKey:   []byte("new public key"),
						Certificate: []byte("new certificate"),
					}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						r, err := http.NewRequest(http.MethodPatch, certURI+id,
							bytes.NewReader(body))
						So(err, ShouldBeNil)
						r = mux.SetURLVars(r, map[string]string{"certificate": "1000"})

						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'NotFound'", func() {
							So(w.Code, ShouldEqual, http.StatusNotFound)
						})

						Convey("Then the response body should state that "+
							"the certificate was not found", func() {
							So(w.Body.String(), ShouldEqual, "Record not found\n")
						})

						Convey("Then the old certificate should still exist", func() {
							exist, err := db.Exists(old)

							So(err, ShouldBeNil)
							So(exist, ShouldBeTrue)
						})
					})
				})
			})
		})
	})
}
