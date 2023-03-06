package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestImportClients(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given a database with some client", func() {
			client := &model.Client{
				Name:         "client",
				Protocol:     testProtocol,
				LocalAddress: "localhost:1111",
			}
			So(db.Insert(client).Run(), ShouldBeNil)

			other := &model.Client{
				Name:         "other",
				Protocol:     testProtocol,
				LocalAddress: "localhost:2222",
			}
			So(db.Insert(other).Run(), ShouldBeNil)

			Convey("Given a list of new clients", func() {
				newClient := file.Client{
					Name:         "new_cli",
					Protocol:     testProtocol,
					Disabled:     false,
					ProtoConfig:  map[string]any{},
					LocalAddress: "localhost:11111",
				}

				updatedClient := file.Client{
					Name:         client.Name,
					Protocol:     testProtocol,
					Disabled:     true,
					ProtoConfig:  map[string]any{},
					LocalAddress: "localhost:22222",
				}

				newClients := []file.Client{newClient, updatedClient}

				SkipConvey("When calling the importClients method", func() {
					So(importClients(discard(), db, newClients, false), ShouldBeNil)

					var dbClients model.Clients
					So(db.Select(&dbClients).OrderBy("id", true).Run(), ShouldBeNil)
					So(dbClients, ShouldHaveLength, 3)

					Convey("Then the new client should have been imported", func() {
						dbClient := dbClients[2]

						So(dbClient, ShouldResemble, &model.Client{
							ID:           3,
							Owner:        conf.GlobalConfig.GatewayName,
							Name:         newClient.Name,
							Protocol:     newClient.Protocol,
							LocalAddress: newClient.LocalAddress,
							ProtoConfig:  newClient.ProtoConfig,
							Disabled:     newClient.Disabled,
						})
					})

					Convey("Then the existing client should have been updated", func() {
						So(dbClients[0], ShouldResemble, &model.Client{
							ID:           client.ID,
							Owner:        client.Owner,
							Name:         updatedClient.Name,
							Protocol:     updatedClient.Protocol,
							LocalAddress: updatedClient.LocalAddress,
							ProtoConfig:  updatedClient.ProtoConfig,
							Disabled:     updatedClient.Disabled,
						})
					})

					Convey("Then the other client should be unchanged", func() {
						So(dbClients[1], ShouldResemble, other)
					})
				})

				Convey("When calling the importLocals method with reset ON", func() {
					So(importClients(discard(), db, newClients, true), ShouldBeNil)

					var dbClients model.Clients
					So(db.Select(&dbClients).OrderBy("id", true).Run(), ShouldBeNil)
					So(dbClients, ShouldHaveLength, 2)

					Convey("Then only the imported clients should be unchanged", func() {
						So(dbClients[0].Name, ShouldEqual, newClient.Name)
						So(dbClients[1].Name, ShouldEqual, updatedClient.Name)
					})
				})
			})
		})
	})
}
