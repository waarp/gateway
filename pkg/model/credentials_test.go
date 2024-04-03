package model

import (
	"database/sql"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestCryptoTableName(t *testing.T) {
	Convey("Given an `Credential` instance", t, func() {
		tbl := &Credential{}

		Convey("When calling the 'TableName' method", func() {
			name := tbl.TableName()

			Convey("Then it should return the name of the auth table", func() {
				So(name, ShouldEqual, TableCredentials)
			})
		})
	})
}

func TestAuthBeforeWrite(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains 1 local agent", func() {
			parentAgent := &LocalAgent{
				Name: "parent", Protocol: testProtocol,
				Address: types.Addr("localhost", 6666),
			}
			So(db.Insert(parentAgent).Run(), ShouldBeNil)

			Convey("Given a new auth method", func() {
				newAuth := &Credential{
					LocalAgentID: utils.NewNullInt64(parentAgent.ID),
					Name:         "cert",
					Type:         testExternalAuth,
					Value:        "val",
				}

				shouldFailWith := func(errDesc string, expErr error) {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := newAuth.BeforeWrite(db)

						Convey("Then the error should say that "+errDesc, func() {
							So(err, ShouldBeError, expErr)
						})
					})
				}

				Convey("Given that the new auth method is valid", func() {
					Convey("When calling the 'BeforeWrite' function", func() {
						err := newAuth.BeforeWrite(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})

				Convey("Given that the new auth method is missing an owner", func() {
					newAuth.LocalAgentID = sql.NullInt64{}

					shouldFailWith("the owner is missing", database.NewValidationError(
						"the authentication method is missing an owner"))
				})

				Convey("Given that the new auth method has multiple owners", func() {
					newAuth.RemoteAgentID = utils.NewNullInt64(1)

					shouldFailWith("the owner ID is missing", database.NewValidationError(
						"the authentication method cannot have multiple targets"))
				})

				Convey("Given that the new auth method has an invalid owner ID", func() {
					newAuth.LocalAgentID = utils.NewNullInt64(1000)

					shouldFailWith("the owner ID is invalid", database.NewValidationError(
						`no server found with ID "1000"`))
				})

				Convey("Given that the new method's name is already taken", func() {
					otherAuth := &Credential{
						LocalAgentID: utils.NewNullInt64(parentAgent.ID),
						Name:         "other",
						Type:         testExternalAuth,
						Value:        "val",
					}
					So(db.Insert(otherAuth).Run(), ShouldBeNil)
					newAuth.Name = otherAuth.Name
					shouldFailWith("the name is taken", database.NewValidationError(
						"an authentication method with the same name %q already exist",
						newAuth.Name))
				})

				Convey("Given that the new method's name is already taken "+
					"but the owner is different", func() {
					oldOwner := conf.GlobalConfig.GatewayName
					conf.GlobalConfig.GatewayName = "other"

					defer func() { conf.GlobalConfig.GatewayName = oldOwner }()

					otherAgent := &LocalAgent{
						Name: "other", Protocol: testProtocol,
						Address: types.Addr("localhost", 6666),
					}
					So(db.Insert(otherAgent).Run(), ShouldBeNil)

					conf.GlobalConfig.GatewayName = oldOwner

					otherAuth := &Credential{
						LocalAgentID: utils.NewNullInt64(parentAgent.ID),
						Name:         "other",
						Type:         testExternalAuth,
						Value:        "val",
					}
					So(db.Insert(otherAuth).Run(), ShouldBeNil)

					Convey("When calling the 'BeforeWrite' function", func() {
						err := newAuth.BeforeWrite(db)

						Convey("Then it should NOT return an error", func() {
							So(err, ShouldBeNil)
						})
					})
				})
			})
		})
	})
}
