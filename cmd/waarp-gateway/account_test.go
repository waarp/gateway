package main

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/jessevdk/go-flags"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

var testAccountPartner model.Partner

func init() {
	testAccountPartner = model.Partner{
		Name:    "test_account_partner",
		Address: "test_account_partner_address",
		Port:    1,
		Type:    "sftp",
	}
	if err := testDb.Create(&testAccountPartner); err != nil {
		panic(err)
	}
}

func TestAccountCreate(t *testing.T) {

	Convey("Testing the account creation function", t, func() {
		testAccount := &model.Account{
			Username:  "test_account_create",
			Password:  []byte("test_account_create_password"),
			PartnerID: testAccountPartner.ID,
		}
		existingAccount := &model.Account{
			Username:  "test_account_existing",
			Password:  []byte("test_account_existing_password"),
			PartnerID: testAccountPartner.ID,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		a := accountCreateCommand{}

		err := testDb.Create(existingAccount)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(existingAccount)
			So(err, ShouldBeNil)
			err = testDb.Delete(testAccount)
			So(err, ShouldBeNil)
		})

		Convey("Given correct values", func() {
			args := []string{"-n", testAccount.Username,
				"-p", string(testAccount.Password),
				"-i", strconv.FormatUint(testAccount.PartnerID, 10),
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a non-existent partner id", func() {
			args := []string{"-n", testAccount.Username,
				"-p", string(testAccount.Password),
				"-i", "1000",
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a non-numeric partner id", func() {
			args := []string{"-n", testAccount.Username,
				"-p", string(testAccount.Password),
				"-i", "not_an_id",
			}
			_, err := flags.ParseArgs(&a, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestAccountSelect(t *testing.T) {

	Convey("Testing the account listing function", t, func() {
		So(model.BcryptRounds, ShouldEqual, bcrypt.MinCost)

		testAccount1 := &model.Account{
			Username: "test_account_select1",
			Password: []byte("test_account_select1_password"),
		}
		testAccount2 := &model.Account{
			Username: "test_account_select2",
			Password: []byte("test_account_select2_password"),
		}
		testAccount3 := &model.Account{
			Username: "test_account_select3",
			Password: []byte("test_account_select3_password"),
		}
		testAccount4 := &model.Account{
			Username: "test_account_select4",
			Password: []byte("test_account_select4_password"),
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		a := accountListCommand{}

		err := testDb.Create(testAccount1)
		So(err, ShouldBeNil)
		err = testDb.Create(testAccount2)
		So(err, ShouldBeNil)
		err = testDb.Create(testAccount3)
		So(err, ShouldBeNil)
		err = testDb.Create(testAccount4)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(testAccount1)
			So(err, ShouldBeNil)
			err = testDb.Delete(testAccount2)
			So(err, ShouldBeNil)
			err = testDb.Delete(testAccount3)
			So(err, ShouldBeNil)
			err = testDb.Delete(testAccount4)
			So(err, ShouldBeNil)
		})

		Convey("Given no flags", func() {
			args := []string{}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a limit flag", func() {
			args := []string{"-l", "2"}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			So(a.Limit, ShouldEqual, 2)
			err = a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an offset flag", func() {
			args := []string{"-o", "2"}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a sort flag", func() {
			args := []string{"-s", "username"}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an order flag", func() {
			args := []string{"-d"}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestAccountDelete(t *testing.T) {

	Convey("Testing the account deletion function", t, func() {
		testAccount := &model.Account{
			Username:  "test_account_delete",
			Password:  []byte("test_account_delete_password"),
			PartnerID: testAccountPartner.ID,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		a := accountDeleteCommand{}

		err := testDb.Create(testAccount)
		So(err, ShouldBeNil)

		Reset(func() {
			err := testDb.Delete(testAccount)
			So(err, ShouldBeNil)
		})

		Convey("Given a correct id", func() {
			args := []string{strconv.FormatUint(testAccount.ID, 10)}
			err := a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given an non-existent id", func() {
			args := []string{"1000"}
			err := a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given an non-numeric id", func() {
			args := []string{"not_an_id"}
			err := a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a no id", func() {
			args := []string{}
			err := a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestAccountUpdate(t *testing.T) {

	Convey("Testing the account update function", t, func() {
		testAccountBefore := &model.Account{
			Username:  "test_account_update_before",
			Password:  []byte("test_account_update_before_password"),
			PartnerID: testAccountPartner.ID,
		}
		testAccountAfter := &model.Account{
			Username:  "test_account_update_after",
			Password:  []byte("test_account_update_after_password"),
			PartnerID: testAccountPartner.ID,
		}

		auth = ConnectionOptions{
			Address:  testServer.URL,
			Username: "admin",
		}
		a := accountUpdateCommand{}

		err := testDb.Create(testAccountBefore)
		So(err, ShouldBeNil)
		id := strconv.FormatUint(testAccountBefore.ID, 10)

		Reset(func() {
			err := testDb.Delete(testAccountBefore)
			So(err, ShouldBeNil)
			err = testDb.Delete(testAccountAfter)
			So(err, ShouldBeNil)
		})

		Convey("Given correct values", func() {
			args := []string{"-n", testAccountAfter.Username,
				"-p", string(testAccountAfter.Password),
				"-i", strconv.FormatUint(testAccountAfter.PartnerID, 10),
				id,
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should not return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("Given a non-existent partner id", func() {
			args := []string{"-n", testAccountAfter.Username,
				"-p", string(testAccountAfter.Password),
				"-i", "1000",
				id,
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a non-numeric partner id", func() {
			args := []string{"-n", testAccountAfter.Username,
				"-p", string(testAccountAfter.Password),
				"-i", "not_an_id",
				id,
			}
			_, err := flags.ParseArgs(&a, args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})

		Convey("Given a no partner id", func() {
			args := []string{"-n", testAccountAfter.Username,
				"-p", string(testAccountAfter.Password),
				"-i", strconv.FormatUint(testAccountAfter.PartnerID, 10),
			}
			args, err := flags.ParseArgs(&a, args)
			So(err, ShouldBeNil)
			err = a.Execute(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError)
			})
		})
	})
}

func TestDisplayAccount(t *testing.T) {

	Convey("Given an account", t, func() {
		testAccount := &model.Account{
			ID:        123,
			PartnerID: 789,
			Username:  "test_account",
			Password:  []byte("test_account_password"),
		}
		id := strconv.FormatUint(testAccount.ID, 10)
		parID := strconv.FormatUint(testAccount.PartnerID, 10)

		Convey("When calling the 'displayAccount' function", func() {
			out, err := ioutil.TempFile(".", "waarp_gateway")
			So(err, ShouldBeNil)
			err = displayAccount(out, testAccount)
			So(err, ShouldBeNil)
			err = out.Close()
			So(err, ShouldBeNil)

			Reset(func() {
				_ = os.Remove(out.Name())
			})

			Convey("Then it should display the account correctly", func() {
				in, err := os.Open(out.Name())
				So(err, ShouldBeNil)
				result, err := ioutil.ReadAll(in)
				So(err, ShouldBeNil)

				expected := "Account n°" + id + ":\n" +
					"├─Username: " + testAccount.Username + "\n" +
					"└─PartnerID: " + parID + "\n"

				So(string(result), ShouldEqual, expected)
			})
		})
	})
}
