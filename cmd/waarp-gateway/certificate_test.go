package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testAccountName      = "test_account"
	incorrectAccountName = "incorrect"
)

func TestCertCreate(t *testing.T) {
	testCert := &model.CertChain{
		Name:        "test_cert_create",
		PrivateKey:  []byte("private_key"),
		PublicKey:   []byte("public_key"),
		PrivateCert: []byte("private_cert"),
		PublicCert:  []byte("public_cert"),
	}
	path := admin.RestURI + admin.PartnersURI + "/" + testPartnerName + admin.AccountsURI +
		"/" + testAccountName + admin.CertsURI
	createHandler := testHandler(http.MethodPost, path, testCert, nil, nil,
		http.StatusCreated)

	Convey("Testing the certificate creation function", t, func() {
		server := httptest.NewServer(createHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		c := certificateCreateCommand{certificateCommand: &certificateCommand{
			Partner: testPartnerName,
			Account: testAccountName,
		}}

		Reset(server.Close)

		Convey("Given correct flags", func() {
			args := []string{"-n", testCert.Name,
				"--private_key=" + string(testCert.PrivateKey),
				"--public_key=" + string(testCert.PublicKey),
				"--private_cert=" + string(testCert.PrivateCert),
				"--public_cert=" + string(testCert.PublicCert),
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given incorrect flags", func() {
			args := []string{"-n", "incorrect",
				"--private_key=incorrect",
				"--public_key=incorrect",
				"--private_cert=incorrect",
				"--public_cert=incorrect",
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusBadRequest))
			})
		})

		Convey("Given an incorrect partner name", func() {
			c.Partner = incorrectPartnerName
			args := []string{"-n", testCert.Name,
				"--private_key=" + string(testCert.PrivateKey),
				"--public_key=" + string(testCert.PublicKey),
				"--private_cert=" + string(testCert.PrivateCert),
				"--public_cert=" + string(testCert.PublicCert),
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})

		Convey("Given an incorrect account name", func() {
			c.Account = incorrectAccountName
			args := []string{"-n", testCert.Name,
				"--private_key=" + string(testCert.PrivateKey),
				"--public_key=" + string(testCert.PublicKey),
				"--private_cert=" + string(testCert.PrivateCert),
				"--public_cert=" + string(testCert.PublicCert),
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})
	})
}

func TestCertGet(t *testing.T) {
	testCert := &model.CertChain{
		Name:        "test_cert_get",
		PrivateKey:  []byte("private_key"),
		PublicKey:   []byte("public_key"),
		PrivateCert: []byte("private_cert"),
		PublicCert:  []byte("public_cert"),
	}

	Convey("Testing the account get function", t, func() {
		path := admin.RestURI + admin.PartnersURI + "/" + testPartnerName + admin.AccountsURI +
			"/" + testAccountName + admin.CertsURI + "/" + testCert.Name
		getHandler := testHandler(http.MethodGet, path, nil, testCert, nil,
			http.StatusOK)

		server := httptest.NewServer(getHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		c := certificateGetCommand{certificateCommand: &certificateCommand{
			Partner: testPartnerName,
			Account: testAccountName,
		}}

		Reset(server.Close)

		Convey("Given a correct name", func() {
			args := []string{testCert.Name}
			err := c.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect name", func() {
			args := []string{"unknown"}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})

		Convey("Given a no name", func() {
			args := []string{}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, "missing certificate name")
			})
		})

		Convey("Given an incorrect partner name", func() {
			c.Partner = incorrectPartnerName
			args := []string{testCert.Name}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})

		Convey("Given an incorrect account name", func() {
			c.Account = incorrectAccountName
			args := []string{testCert.Name}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})
	})
}

func TestCertSelect(t *testing.T) {
	testCert1 := &model.CertChain{
		Name:        "test_cert_create1",
		PrivateKey:  []byte("private_key"),
		PublicKey:   []byte("public_key"),
		PrivateCert: []byte("private_cert"),
		PublicCert:  []byte("public_cert"),
	}
	testCert2 := &model.CertChain{
		Name:        "test_cert_create2",
		PrivateKey:  []byte("private_key"),
		PublicKey:   []byte("public_key"),
		PrivateCert: []byte("private_cert"),
		PublicCert:  []byte("public_cert"),
	}
	testCert3 := &model.CertChain{
		Name:        "test_cert_create3",
		PrivateKey:  []byte("private_key"),
		PublicKey:   []byte("public_key"),
		PrivateCert: []byte("private_cert"),
		PublicCert:  []byte("public_cert"),
	}
	testCert4 := &model.CertChain{
		Name:        "test_cert_create4",
		PrivateKey:  []byte("private_key"),
		PublicKey:   []byte("public_key"),
		PrivateCert: []byte("private_cert"),
		PublicCert:  []byte("public_cert"),
	}
	path := admin.RestURI + admin.PartnersURI + "/" + testPartnerName + admin.AccountsURI +
		"/" + testAccountName + admin.CertsURI

	Convey("Testing the account listing function", t, func() {
		c := certificateListCommand{certificateCommand: &certificateCommand{
			Partner: testPartnerName,
			Account: testAccountName,
		}}

		Convey("Given no flags", func() {
			arrayResult := []*model.CertChain{testCert1, testCert2, testCert3, testCert4}
			expectedResults := map[string][]*model.CertChain{"certificates": arrayResult}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				nil, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a limit flag", func() {
			arrayResult := []*model.CertChain{testCert1, testCert2}
			expectedResults := map[string][]*model.CertChain{"certificates": arrayResult}
			expectedParams := url.Values{"limit": []string{"2"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-l", "2"}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			So(c.Limit, ShouldEqual, 2)
			err = c.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an offset flag", func() {
			arrayResult := []*model.CertChain{testCert3, testCert4}
			expectedResults := map[string][]*model.CertChain{"accounts": arrayResult}
			expectedParams := url.Values{"offset": []string{"2"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-o", "2"}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a sort flag", func() {
			arrayResult := []*model.CertChain{testCert1, testCert2, testCert3, testCert4}
			expectedResults := map[string][]*model.CertChain{"certificates": arrayResult}
			expectedParams := url.Values{"sortby": []string{"name"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-s", "name"}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an order flag", func() {
			arrayResult := []*model.CertChain{testCert4, testCert3, testCert2, testCert1}
			expectedResults := map[string][]*model.CertChain{"certificates": arrayResult}
			expectedParams := url.Values{"order": []string{"desc"}}
			selectHandler := testHandler(http.MethodGet, path, nil, expectedResults,
				expectedParams, http.StatusOK)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			args := []string{"-d"}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect partner name", func() {
			selectHandler := testHandler(http.MethodGet, path, nil,
				http.StatusText(http.StatusNotFound), nil, http.StatusNotFound)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			c.Partner = incorrectPartnerName
			args := []string{}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})

		Convey("Given an incorrect account name", func() {
			selectHandler := testHandler(http.MethodGet, path, nil,
				http.StatusText(http.StatusNotFound), nil, http.StatusNotFound)

			server := httptest.NewServer(selectHandler)
			auth = ConnectionOptions{
				Address:  server.URL,
				Username: "test",
			}

			c.Account = incorrectAccountName
			args := []string{}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})
	})
}

func TestCertDelete(t *testing.T) {
	testCertificateName := "test_certificate"

	Convey("Testing the account deletion function", t, func() {
		path := admin.RestURI + admin.PartnersURI + "/" + testPartnerName + admin.AccountsURI +
			"/" + testAccountName + admin.CertsURI + "/" + testCertificateName
		getHandler := testHandler(http.MethodDelete, path, nil, nil, nil,
			http.StatusNoContent)

		server := httptest.NewServer(getHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		c := certificateDeleteCommand{certificateCommand: &certificateCommand{
			Partner: testPartnerName,
			Account: testAccountName,
		}}

		Reset(server.Close)

		Convey("Given a correct name", func() {
			args := []string{testCertificateName}
			err := c.Execute(args)

			Reset(server.Close)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an incorrect name", func() {
			args := []string{"unknown"}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})

		Convey("Given a no name", func() {
			args := []string{}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, "missing certificate name")
			})
		})

		Convey("Given an incorrect partner name", func() {
			c.Partner = incorrectPartnerName
			args := []string{testCertificateName}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})

		Convey("Given an incorrect account name", func() {
			c.Account = incorrectAccountName
			args := []string{testCertificateName}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})
	})
}

func TestCertUpdate(t *testing.T) {
	testCertificateName := "test_cert_before"
	testCertAfter := &model.CertChain{
		Name:        "test_cert_after",
		PrivateKey:  []byte("private_key"),
		PublicKey:   []byte("public_key"),
		PrivateCert: []byte("private_cert"),
		PublicCert:  []byte("public_cert"),
	}
	path := admin.RestURI + admin.PartnersURI + "/" + testPartnerName + admin.AccountsURI +
		"/" + testAccountName + admin.CertsURI + "/" + testCertificateName
	createHandler := testHandler(http.MethodPatch, path, testCertAfter, nil, nil,
		http.StatusCreated)

	Convey("Testing the account update function", t, func() {
		server := httptest.NewServer(createHandler)
		auth = ConnectionOptions{
			Address:  server.URL,
			Username: "test",
		}
		c := certificateUpdateCommand{certificateCommand: &certificateCommand{
			Partner: testPartnerName,
			Account: testAccountName,
		}}

		Reset(server.Close)

		Convey("Given correct flags", func() {
			args := []string{"-n", testCertAfter.Name,
				"--private_key=" + string(testCertAfter.PrivateKey),
				"--public_key=" + string(testCertAfter.PublicKey),
				"--private_cert=" + string(testCertAfter.PrivateCert),
				"--public_cert=" + string(testCertAfter.PublicCert),
				testCertificateName,
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given incorrect flags", func() {
			args := []string{"-n", "incorrect",
				"--private_key=incorrect",
				"--public_key=incorrect",
				"--private_cert=incorrect",
				"--public_cert=incorrect",
				testCertificateName,
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusBadRequest))
			})
		})

		Convey("Given no certificate name", func() {
			args := []string{"-n", testCertAfter.Name,
				"--private_key=" + string(testCertAfter.PrivateKey),
				"--public_key=" + string(testCertAfter.PublicKey),
				"--private_cert=" + string(testCertAfter.PrivateCert),
				"--public_cert=" + string(testCertAfter.PublicCert),
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, "missing certificate name")
			})
		})

		Convey("Given an incorrect partner name", func() {
			c.Partner = incorrectPartnerName
			args := []string{"-n", testCertAfter.Name,
				"--private_key=" + string(testCertAfter.PrivateKey),
				"--public_key=" + string(testCertAfter.PublicKey),
				"--private_cert=" + string(testCertAfter.PrivateCert),
				"--public_cert=" + string(testCertAfter.PublicCert),
				testCertificateName,
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)

				So(err.Error(), ShouldResemble, http.StatusText(http.StatusNotFound))
			})
		})
	})
}

func TestDisplayCertificate(t *testing.T) {

	Convey("Given a certificate", t, func() {
		testCert := &model.CertChain{
			Name:        "test_cert",
			PrivateKey:  []byte("private_key"),
			PublicKey:   []byte("public_key"),
			PrivateCert: []byte("private_cert"),
			PublicCert:  []byte("public_cert"),
		}

		Convey("When calling the 'displayCertificate' function", func() {
			out, err := ioutil.TempFile(".", "waarp_gateway")
			So(err, ShouldBeNil)
			err = displayCertificate(out, testCert)
			So(err, ShouldBeNil)
			err = out.Close()
			So(err, ShouldBeNil)

			Reset(func() {
				_ = os.Remove(out.Name())
			})

			Convey("Then it should display the certificate correctly", func() {
				in, err := os.Open(out.Name())
				So(err, ShouldBeNil)
				result, err := ioutil.ReadAll(in)
				So(err, ShouldBeNil)

				expected :=
					"Certificate '" + testCert.Name + "':\n" +
						"├─Private Key: " + string(testCert.PrivateKey) + "\n" +
						"├─Public Key: " + string(testCert.PublicKey) + "\n" +
						"├─Private Cert: " + string(testCert.PrivateCert) + "\n" +
						"└─Public Cert: " + string(testCert.PublicCert) + "\n"

				So(string(result), ShouldEqual, expected)
			})
		})
	})
}
