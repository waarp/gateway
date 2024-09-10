package r66

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestRemoteAgentAfterInsert(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("When inserting a new R66 remote agent with a serverPassword", func() {
			const serverPswd = "bar"

			ag := &model.RemoteAgent{
				Name:     "r66-partner",
				Protocol: "r66",
				Address:  types.Addr("localhost", 6666),
				ProtoConfig: map[string]any{
					"serverLogin":    "foo",
					"serverPassword": serverPswd,
				},
			}
			So(db.Insert(ag).Run(), ShouldBeNil)

			Convey("Then the password should have been stored in the database", func() {
				var pswd model.Credential
				So(db.Get(&pswd, "remote_agent_id=? AND type=?", ag.ID,
					auth.Password).Run(), ShouldBeNil)

				So(bcrypt.CompareHashAndPassword([]byte(pswd.Value),
					[]byte(CryptPass(serverPswd))), ShouldBeNil)
			})
		})
	})
}

func TestRemoteAgentAfterUpdate(t *testing.T) {
	Convey("Given a database with an existing R66 remote agent", t, func(c C) {
		db := database.TestDatabase(c)

		ag := &model.RemoteAgent{
			Name:     "r66-partner",
			Protocol: "r66",
			Address:  types.Addr("localhost", 6666),
			ProtoConfig: map[string]any{
				"serverLogin":    "foo",
				"serverPassword": "bar",
			},
		}
		So(db.Insert(ag).Run(), ShouldBeNil)

		Convey("When updating the R66 remote agent with a serverPassword", func() {
			const serverPswd = "baz"
			ag.ProtoConfig["serverPassword"] = serverPswd

			So(db.Update(ag).Run(), ShouldBeNil)

			Convey("Then the password should have been stored in the database", func() {
				var pswd model.Credential
				So(db.Get(&pswd, "remote_agent_id=? AND type=?", ag.ID,
					auth.Password).Run(), ShouldBeNil)

				So(bcrypt.CompareHashAndPassword([]byte(pswd.Value),
					[]byte(CryptPass(serverPswd))), ShouldBeNil)
			})
		})
	})
}
