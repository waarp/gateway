package file

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func shouldBeHashOf(hashed any, pswd string) {
	var hash []byte
	switch typed := hashed.(type) {
	case string:
		hash = []byte(typed)
	case []byte:
		hash = typed
	default:
		So(hashed, ShouldHaveSameTypeAs, "")
	}

	So(bcrypt.CompareHashAndPassword(hash, []byte(pswd)), ShouldBeNil)
}

func TestUserUnmarshalJSON(t *testing.T) {
	Convey("Given a JSON user", t, func() {
		input := []byte(`{"username": "foo", "password": "bar"}`)

		Convey("When unmarshalling the JSON", func() {
			var user User
			So(json.Unmarshal(input, &user), ShouldBeNil)

			Convey("Then it should have hashed the password", func() {
				shouldBeHashOf(user.PasswordHash, user.Password)
			})
		})
	})
}

func TestLocalAgentUnmarshalJSON(t *testing.T) {
	Convey("Given a JSON local agent", t, func() {
		input := []byte(`{
			"name": "foo", 
			"protocol": "r66",
			"address": "1.2.3.4:5",
			"accounts": [{
				"login": "foo1",
				"password": "sesame1"
			}, {
				"login": "foo2",
				"password": "sesame2"
			}]
		}`)

		Convey("When unmarshalling the JSON", func() {
			var server LocalAgent
			So(json.Unmarshal(input, &server), ShouldBeNil)

			Convey("Then it should have hashed the accounts' passwords", func() {
				So(server.Accounts, ShouldHaveLength, 2)

				acc1 := &server.Accounts[0]
				acc2 := &server.Accounts[1]

				So(acc1.Password, ShouldEqual, "sesame1")
				So(acc2.Password, ShouldEqual, "sesame2")
				shouldBeHashOf(acc1.PasswordHash, utils.R66Hash("sesame1"))
				shouldBeHashOf(acc2.PasswordHash, utils.R66Hash("sesame2"))
			})
		})
	})
}

func TestRemoteAgentUnmarshalJSON(t *testing.T) {
	Convey("Given a JSON R66 remote agent", t, func() {
		input := []byte(`{
			"name": "foo", 
			"protocol": "r66",
			"address": "1.2.3.4:5",
			"configuration": {
				"serverLogin": "foo",
				"serverPassword": "bar"
			}
		}`)

		Convey("When unmarshalling the JSON", func() {
			var partner RemoteAgent
			So(json.Unmarshal(input, &partner), ShouldBeNil)

			Convey("Then it should have hashed the server's password", func() {
				So(partner.Configuration, ShouldContainKey, "serverPassword")

				So(partner.Configuration, ShouldContainKey, "serverPassword")
				//nolint:forcetypeassert //type assertion always succeeds
				shouldBeHashOf(partner.Configuration["serverPassword"], utils.R66Hash("bar"))
			})
		})
	})

	Convey("Given a JSON R66 remote agent with hashed password", t, func() {
		input := []byte(`{
			"name": "foo", 
			"protocol": "r66",
			"address": "1.2.3.4:5",
			"configuration": {
				"serverLogin": "foo",
				"serverPassword": "$2a$04$KgPxfeImO46ddWHq9En28eHRmBN3TjQvmzJ3QLiFpxV2jmIk.ZSl6"
			}
		}`)

		Convey("When unmarshalling the JSON", func() {
			var partner RemoteAgent
			So(json.Unmarshal(input, &partner), ShouldBeNil)

			Convey("Then it should NOT have changed the server's password", func() {
				So(partner.Configuration, ShouldContainKey, "serverPassword")

				//nolint:forcetypeassert //type assertion always succeeds
				hash := partner.Configuration["serverPassword"]
				So(hash, ShouldEqual, "$2a$04$KgPxfeImO46ddWHq9En28eHRmBN3TjQvmzJ3QLiFpxV2jmIk.ZSl6")
				shouldBeHashOf(hash, utils.R66Hash("bar"))
			})
		})
	})
}
