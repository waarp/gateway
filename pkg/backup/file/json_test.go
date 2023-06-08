package file

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"
)

func shouldBeHashOf(hash, pswd string) {
	So(bcrypt.CompareHashAndPassword([]byte(hash), []byte(pswd)), ShouldBeNil)
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
			"protocol": "http",
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

				shouldBeHashOf(acc1.PasswordHash, acc1.Password)
				shouldBeHashOf(acc2.PasswordHash, acc2.Password)
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
				var conf map[string]any
				So(json.Unmarshal(partner.Configuration, &conf), ShouldBeNil)

				So(conf, ShouldContainKey, "serverPassword")
				//nolint:forcetypeassert //this is a test
				So(bcrypt.CompareHashAndPassword([]byte(conf["serverPassword"].(string)),
					[]byte("bar")), ShouldBeNil)
			})
		})
	})
}
