package admin

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

const certPath = RestURI + CertsURI + "/"

func TestGetCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_get_test")

	Convey("Given the certificate get handler", t, func() {
		db := database.GetTestDatabase()
		handler := getCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 certificate", func() {
			grandparent := model.Partner{
				ID:          1,
				Name:        "grandparent",
				Address:     "address",
				Port:        1,
				InterfaceID: 1,
			}
			err := db.Create(&grandparent)
			So(err, ShouldBeNil)

			parent := model.Account{
				ID:        1,
				Username:  "parent",
				Password:  []byte("password"),
				PartnerID: grandparent.ID,
			}
			err = db.Create(&parent)
			So(err, ShouldBeNil)

			expected := model.CertChain{
				ID:         1,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "expected",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			err = db.Create(&expected)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(expected.ID, 10)

			Convey("Given a request with the valid certificate ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, certPath+id, nil)
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

						res := model.CertChain{}
						err := json.Unmarshal(w.Body.Bytes(), &res)

						So(err, ShouldBeNil)
						So(res, ShouldResemble, expected)
					})
				})
			})

			Convey("Given a request with a non-existing certificate ID parameter", func() {
				r, err := http.NewRequest(http.MethodGet, certPath+"1000", nil)
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
	logger := log.NewLogger("rest_cert_list_test")

	check := func(w *httptest.ResponseRecorder, expected map[string][]model.CertChain) {
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

			response := map[string][]model.CertChain{}
			err := json.Unmarshal(w.Body.Bytes(), &response)

			So(err, ShouldBeNil)
			So(response, ShouldResemble, expected)
		})
	}

	Convey("Given the certificate listing handler", t, func() {
		db := database.GetTestDatabase()
		handler := listCertificates(logger, db)
		w := httptest.NewRecorder()
		expected := map[string][]model.CertChain{}

		Convey("Given a database with 4 certificates", func() {
			grandParent := model.Partner{
				ID:          1,
				Name:        "grandparent",
				Address:     "address",
				Port:        1,
				InterfaceID: 1,
			}
			err := db.Create(&grandParent)
			So(err, ShouldBeNil)

			parent := model.Account{
				ID:        1,
				Username:  "parent",
				Password:  []byte("password"),
				PartnerID: grandParent.ID,
			}
			err = db.Create(&parent)
			So(err, ShouldBeNil)

			cert1 := model.CertChain{
				ID:         1,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "cert1",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			cert2 := model.CertChain{
				ID:         2,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "cert2",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			cert3 := model.CertChain{
				ID:         3,
				OwnerType:  "ACCOUNT",
				OwnerID:    1000,
				Name:       "cert3",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			cert4 := model.CertChain{
				ID:         4,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "cert4",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			err = db.Create(&cert1)
			So(err, ShouldBeNil)
			err = db.Create(&cert2)
			So(err, ShouldBeNil)
			err = db.Create(&cert3)
			So(err, ShouldBeNil)
			err = db.Create(&cert4)
			So(err, ShouldBeNil)

			Convey("Given a request with with no parameters", func() {
				r, err := http.NewRequest(http.MethodGet, certPath, nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["certificates"] = []model.CertChain{cert1, cert2, cert3, cert4}
					check(w, expected)
				})
			})

			Convey("Given a request with a limit parameter", func() {
				r, err := http.NewRequest(http.MethodGet, accountsPath+"?limit=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["certificates"] = []model.CertChain{cert1}
					check(w, expected)
				})
			})

			Convey("Given a request with a offset parameter", func() {
				r, err := http.NewRequest(http.MethodGet, certPath+"?offset=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["certificates"] = []model.CertChain{cert2, cert3, cert4}
					check(w, expected)
				})
			})

			Convey("Given a request with a sort & order parameters", func() {
				r, err := http.NewRequest(http.MethodGet, certPath+
					"?sortby=name&order=desc", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["certificates"] = []model.CertChain{cert4, cert3,
						cert2, cert1}
					check(w, expected)
				})
			})

			Convey("Given a request with an account parameter", func() {
				r, err := http.NewRequest(http.MethodGet, certPath+"?account=1", nil)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					handler.ServeHTTP(w, r)

					expected["certificates"] = []model.CertChain{cert1, cert2, cert4}
					check(w, expected)
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
			grandParent := model.Partner{
				ID:          1,
				Name:        "grandparent",
				Address:     "address",
				Port:        1,
				InterfaceID: 1,
			}
			err := db.Create(&grandParent)
			So(err, ShouldBeNil)

			parent := model.Account{
				ID:        1,
				Username:  "parent",
				Password:  []byte("password"),
				PartnerID: grandParent.ID,
			}
			err = db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.CertChain{
				ID:         1,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "existing",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			err = db.Create(&existing)
			So(err, ShouldBeNil)

			Convey("Given a new certificate to insert in the database", func() {
				newCert := model.CertChain{
					ID:         2,
					OwnerType:  "ACCOUNT",
					OwnerID:    parent.ID,
					Name:       "new_cert",
					PrivateKey: []byte("private_key"),
					PublicKey:  []byte("public_key"),
					Cert:       []byte("certificate"),
				}

				Convey("Given that the new account is valid for insertion", func() {
					body, err := json.Marshal(newCert)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, certPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply 'Created'", func() {
							So(w.Code, ShouldEqual, http.StatusCreated)
						})

						Convey("Then the 'Location' header should contain the URI "+
							"of the new account", func() {

							location := w.Header().Get("Location")
							expected := certPath + strconv.FormatUint(newCert.ID, 10)
							So(location, ShouldEqual, expected)
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

				Convey("Given that the new certificate's ID already exist", func() {
					newCert.ID = existing.ID

					body, err := json.Marshal(newCert)
					So(err, ShouldBeNil)
					r, err := http.NewRequest(http.MethodPost, accountsPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain "+
							"a message stating that the ID already exist", func() {

							So(w.Body.String(), ShouldEqual, "A certificate "+
								"with the same ID already exist\n")
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
					r, err := http.NewRequest(http.MethodPost, certPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the naùe already exist", func() {

							So(w.Body.String(), ShouldEqual, "A certificate "+
								"with the same name already exist for this account\n")
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
					r, err := http.NewRequest(http.MethodPost, certPath, bytes.NewReader(body))

					So(err, ShouldBeNil)

					Convey("When sending the request to the handler", func() {
						handler.ServeHTTP(w, r)

						Convey("Then it should reply with a 'Bad Request' error", func() {
							So(w.Code, ShouldEqual, http.StatusBadRequest)
						})

						Convey("Then the response body should contain a message stating "+
							"that the accountID is not valid", func() {

							So(w.Body.String(), ShouldEqual,
								"No account found with ID '1000'\n")
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
	logger := log.NewLogger("rest_cert_delete_test")

	Convey("Given the certificate deletion handler", t, func() {
		db := database.GetTestDatabase()
		handler := deleteCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 1 certificate", func() {
			grandParent := model.Partner{
				ID:          1,
				Name:        "grandparent",
				Address:     "address",
				Port:        1,
				InterfaceID: 1,
			}
			err := db.Create(&grandParent)
			So(err, ShouldBeNil)

			parent := model.Account{
				ID:        1,
				Username:  "parent",
				Password:  []byte("password"),
				PartnerID: grandParent.ID,
			}
			err = db.Create(&parent)
			So(err, ShouldBeNil)

			existing := model.CertChain{
				ID:         1,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "existing",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			err = db.Create(&existing)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(existing.ID, 10)

			Convey("Given a request with the valid certificate ID parameter", func() {
				r, err := http.NewRequest(http.MethodDelete, certPath+id, nil)
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
				r, err := http.NewRequest(http.MethodDelete, certPath+"1000", nil)
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
	logger := log.NewLogger("rest_cert_update_logger")

	Convey("Given the certificate updating handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 certificates", func() {
			grandParent := model.Partner{
				ID:          1,
				Name:        "grandparent",
				Address:     "address",
				Port:        1,
				InterfaceID: 1,
			}
			err := db.Create(&grandParent)
			So(err, ShouldBeNil)

			parent := model.Account{
				ID:        1,
				Username:  "parent",
				Password:  []byte("password"),
				PartnerID: grandParent.ID,
			}
			err = db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.CertChain{
				ID:         1,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "old",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			other := model.CertChain{
				ID:         2,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "other",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given new values to update the certificate with", func() {

				Convey("Given a new name", func() {
					update := struct{ Name string }{Name: "update"}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					expected := model.CertChain{
						ID:         old.ID,
						OwnerType:  old.OwnerType,
						OwnerID:    old.OwnerID,
						Name:       update.Name,
						PrivateKey: old.PrivateKey,
						PublicKey:  old.PublicKey,
						Cert:       old.Cert,
					}

					checkValidUpdate(db, w, http.MethodPatch, certPath, id,
						"certificate", body, handler, &old, &expected)
				})

				Convey("Given an already existing name", func() {
					update := struct{ Name string }{Name: other.Name}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "A certificate with the same name already exist for this account\n"
					checkInvalidUpdate(db, handler, w, body, certPath, id,
						"certificate", &old, msg)
				})

				Convey("Given an invalid new account ID", func() {
					update := struct{ OwnerID uint64 }{OwnerID: 1000}
					body, err := json.Marshal(update)
					So(err, ShouldBeNil)

					msg := "No account found with ID '1000'\n"
					checkInvalidUpdate(db, handler, w, body, certPath, id,
						"certificate", &old, msg)
				})
			})
		})
	})
}

func TestReplaceCert(t *testing.T) {
	logger := log.NewLogger("rest_cert_replace_logger")

	Convey("Given the certificate replacing handler", t, func() {
		db := database.GetTestDatabase()
		handler := updateCertificate(logger, db)
		w := httptest.NewRecorder()

		Convey("Given a database with 2 certificates", func() {
			grandParent := model.Partner{
				ID:          1,
				Name:        "grandparent",
				Address:     "address",
				Port:        1,
				InterfaceID: 1,
			}
			err := db.Create(&grandParent)
			So(err, ShouldBeNil)

			parent := model.Account{
				ID:        1,
				Username:  "parent",
				Password:  []byte("password"),
				PartnerID: grandParent.ID,
			}
			err = db.Create(&parent)
			So(err, ShouldBeNil)

			old := model.CertChain{
				ID:         1,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "old",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			other := model.CertChain{
				ID:         2,
				OwnerType:  "ACCOUNT",
				OwnerID:    parent.ID,
				Name:       "other",
				PrivateKey: []byte("private_key"),
				PublicKey:  []byte("public_key"),
				Cert:       []byte("certificate"),
			}
			err = db.Create(&old)
			So(err, ShouldBeNil)
			err = db.Create(&other)
			So(err, ShouldBeNil)

			id := strconv.FormatUint(old.ID, 10)

			Convey("Given a valid new certificate", func() {
				replace := struct {
					OwnerType                   string
					OwnerID                     uint64
					Name                        string
					PrivateKey, PublicKey, Cert []byte
				}{
					OwnerType:  "ACCOUNT",
					OwnerID:    parent.ID,
					Name:       "replace",
					PrivateKey: []byte("new_private_key"),
					PublicKey:  []byte("new_public_key"),
					Cert:       []byte("new_certificate"),
				}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				expected := model.CertChain{
					ID:      old.ID,
					OwnerID: replace.OwnerID,
					Name:    replace.Name,
				}

				checkValidUpdate(db, w, http.MethodPut, certPath,
					id, "certificate", body, handler, &old, &expected)
			})

			Convey("Given a non-existing certificate ID parameter", func() {
				replace := struct{}{}

				body, err := json.Marshal(replace)
				So(err, ShouldBeNil)

				Convey("When sending the request to the handler", func() {
					r, err := http.NewRequest(http.MethodPut, certPath+"1000",
						bytes.NewReader(body))
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
}
