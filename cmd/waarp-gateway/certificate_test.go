package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
)

var (
	testCertPartner model.Partner
	testCertAccount model.Account
)

func init() {
	testCertPartner = model.Partner{
		Name:    "test_account_partner",
		Address: "test_account_partner_address",
		Port:    1,
		Type:    "sftp",
	}
	if err := testDb.Create(&testCertPartner); err != nil {
		panic(err)
	}

	testCertAccount = model.Account{
		Username:  "test_cert_account",
		PartnerID: testCertPartner.ID,
		Password:  []byte("test_cert_account_password"),
	}
	if err := testDb.Create(&testCertAccount); err != nil {
		panic(err)
	}
}

func TestCertCreate(t *testing.T) {

	Convey("Testing the certificate creation function", t, func() {
		testCert := &model.CertChain{
			Name:       "test_cert_create",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}
		existingCert := &model.CertChain{
			Name:       "test_cert_existing",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		c := certificateCreateCommand{}

		err := testDb.Create(existingCert)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(existingCert)
			So(err, ShouldBeNil)
			err = testDb.Delete(testCert)
			So(err, ShouldBeNil)
		})

		Convey("Given correct values", func() {
			args := []string{"-n", testCert.Name,
				"-i", strconv.FormatUint(testCert.AccountID, 10),
				"--private_key=" + string(testCert.PrivateKey),
				"--public_key=" + string(testCert.PublicKey),
				"--cert=" + string(testCert.Cert),
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given already existing values", func() {
			args := []string{"-n", existingCert.Name,
				"-i", strconv.FormatUint(existingCert.AccountID, 10),
				"--private_key=" + string(existingCert.PrivateKey),
				"--public_key=" + string(existingCert.PublicKey),
				"--cert=" + string(existingCert.Cert),
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a non-existent account id", func() {
			args := []string{"-n", testCert.Name,
				"-i", "1000",
				"--private_key=" + string(testCert.PrivateKey),
				"--public_key=" + string(testCert.PublicKey),
				"--cert=" + string(testCert.Cert),
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a non-numeric account id", func() {
			args := []string{"-n", testCert.Name,
				"-i", "not_an_id",
				"--private_key=" + string(testCert.PrivateKey),
				"--public_key=" + string(testCert.PublicKey),
				"--cert=" + string(testCert.Cert),
			}
			_, err := flags.ParseArgs(&c, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestCertGet(t *testing.T) {

	Convey("Testing the account get function", t, func() {
		testCert := &model.CertChain{
			Name:       "test_cert_get",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		c := certificateGetCommand{}

		err := testDb.Create(testCert)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(testCert)
			So(err, ShouldBeNil)
		})

		Convey("Given a correct id", func() {
			args := []string{strconv.FormatUint(testCert.ID, 10)}
			err := c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an non-existent id", func() {
			args := []string{"1000"}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a no id", func() {
			args := []string{}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestCertSelect(t *testing.T) {

	Convey("Testing the account listing function", t, func() {
		testCert1 := &model.CertChain{
			Name:       "test_cert_create1",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}
		testCert2 := &model.CertChain{
			Name:       "test_cert_create2",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}
		testCert3 := &model.CertChain{
			Name:       "test_cert_create3",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}
		testCert4 := &model.CertChain{
			Name:       "test_cert_create4",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		c := certificateListCommand{}

		err := testDb.Create(testCert1)
		So(err, ShouldBeNil)
		err = testDb.Create(testCert2)
		So(err, ShouldBeNil)
		err = testDb.Create(testCert3)
		So(err, ShouldBeNil)
		err = testDb.Create(testCert4)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(testCert1)
			So(err, ShouldBeNil)
			err = testDb.Delete(testCert2)
			So(err, ShouldBeNil)
			err = testDb.Delete(testCert3)
			So(err, ShouldBeNil)
			err = testDb.Delete(testCert4)
			So(err, ShouldBeNil)
		})

		Convey("Given no flags", func() {
			args := []string{}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a limit flag", func() {
			args := []string{"-l", "2"}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an offset flag", func() {
			args := []string{"-o", "2"}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a sort flag", func() {
			args := []string{"-s", "name"}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an order flag", func() {
			args := []string{"-d"}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestCertDelete(t *testing.T) {

	Convey("Testing the account deletion function", t, func() {
		testCert := &model.CertChain{
			Name:       "test_cert_delete",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		c := certificateDeleteCommand{}

		err := testDb.Create(testCert)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(testCert)
			So(err, ShouldBeNil)
		})

		Convey("Given a correct id", func() {
			args := []string{strconv.FormatUint(testCert.ID, 10)}
			err := c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a non-existent id", func() {
			args := []string{"1000"}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a non-numeric id", func() {
			args := []string{"not_an_id"}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a no id", func() {
			args := []string{}
			err := c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestCertUpdate(t *testing.T) {

	Convey("Testing the account update function", t, func() {
		testCertBefore := &model.CertChain{
			Name:       "test_cert_before",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}
		testCertAfter := &model.CertChain{
			Name:       "test_cert_after",
			AccountID:  testCertAccount.ID,
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		c := certificateUpdateCommand{}

		err := testDb.Create(testCertBefore)
		So(err, ShouldBeNil)
		id := strconv.FormatUint(testCertBefore.ID, 10)

		Reset(func() {
			err := testDb.Delete(testCertBefore)
			So(err, ShouldBeNil)
			err = testDb.Delete(testCertAfter)
			So(err, ShouldBeNil)
		})

		Convey("Given correct values", func() {
			args := []string{"-n", testCertAfter.Name,
				"-i", strconv.FormatUint(testCertAfter.AccountID, 10),
				"--private_key=" + string(testCertAfter.PrivateKey),
				"--public_key=" + string(testCertAfter.PublicKey),
				"--cert=" + string(testCertAfter.Cert),
				id,
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a non-existent account id", func() {
			args := []string{"-n", testCertAfter.Name,
				"-i", "1000",
				"--private_key=" + string(testCertAfter.PrivateKey),
				"--public_key=" + string(testCertAfter.PublicKey),
				"--cert=" + string(testCertAfter.Cert),
				id,
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a non-numeric account id", func() {
			args := []string{"-n", testCertAfter.Name,
				"-i", "not_an_id",
				"--private_key=" + string(testCertAfter.PrivateKey),
				"--public_key=" + string(testCertAfter.PublicKey),
				"--cert=" + string(testCertAfter.Cert),
				id,
			}
			_, err := flags.ParseArgs(&c, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given no id", func() {
			args := []string{"-n", testCertAfter.Name,
				"-i", strconv.FormatUint(testCertAfter.AccountID, 10),
				"--private_key=" + string(testCertAfter.PrivateKey),
				"--public_key=" + string(testCertAfter.PublicKey),
				"--cert=" + string(testCertAfter.Cert),
			}
			args, err := flags.ParseArgs(&c, args)
			So(err, ShouldBeNil)
			err = c.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestDisplayCertificate(t *testing.T) {

	Convey("Given a certificate", t, func() {
		testCert := &model.CertChain{
			ID:         123,
			AccountID:  789,
			Name:       "test_cert",
			PrivateKey: []byte("private_key"),
			PublicKey:  []byte("public_key"),
			Cert:       []byte("cert"),
		}
		id := strconv.FormatUint(testCert.ID, 10)
		accID := strconv.FormatUint(testCert.AccountID, 10)

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
					"Certificate n°" + id + ":\n" +
						"├─Name: " + testCert.Name + "\n" +
						"├─AccountID: " + accID + "\n" +
						"├─Private Key: " + string(testCert.PrivateKey) + "\n" +
						"├─Public Key: " + string(testCert.PublicKey) + "\n" +
						"└─Cert: " + string(testCert.Cert) + "\n"

				So(string(result), ShouldEqual, expected)
			})
		})
	})
}
