package r66

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestLocalAgentAfterRead(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains 1 R66 local agent", func() {
			oldAgent := model.LocalAgent{
				Owner:    "test_gateway",
				Name:     "old",
				Protocol: R66,
				Address:  types.Addr("localhost", 2022),
				ProtoConfig: map[string]any{
					"serverLogin":    "foo",
					"serverPassword": "bar",
				},
			}
			So(db.Insert(&oldAgent).Run(), ShouldBeNil)

			Convey("When retrieving the agent from the database", func() {
				var check model.LocalAgent
				So(db.Get(&check, "id=?", oldAgent.ID).Run(), ShouldBeNil)

				Convey("Then it should have decrypted the R66 server password", func() {
					So(check.ProtoConfig, ShouldContainKey, "serverPassword")
					So(check.ProtoConfig["serverPassword"], ShouldEqual, "bar")
				})
			})
		})
	})
}
