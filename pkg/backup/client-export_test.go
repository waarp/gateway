package backup

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestExportClients(t *testing.T) {
	Convey("Given a database", t, func(c C) {
		db := database.TestDatabase(c)

		Convey("Given the database contains clients & remote agents", func() {
			client1 := &model.Client{
				Name:                 "client1",
				Protocol:             testProtocol,
				LocalAddress:         types.Addr("localhost", 1111),
				NbOfAttempts:         5,
				FirstRetryDelay:      90,
				RetryIncrementFactor: 1.5,
			}
			So(db.Insert(client1).Run(), ShouldBeNil)

			client2 := &model.Client{
				Name:         "client2",
				Protocol:     testProtocol,
				LocalAddress: types.Addr("localhost", 2222),
				ProtoConfig:  map[string]any{"foo": "bar"},
			}
			So(db.Insert(client2).Run(), ShouldBeNil)

			Convey("When calling the exportClients function", func() {
				res, err := exportClients(discard(), db)
				So(err, ShouldBeNil)

				Convey("Then it should have exported the 2 clients", func() {
					So(res, ShouldHaveLength, 2)

					So(res[0], ShouldResemble, file.Client{
						Name:                 client1.Name,
						Protocol:             client1.Protocol,
						Disabled:             client1.Disabled,
						LocalAddress:         client1.LocalAddress.String(),
						ProtoConfig:          client1.ProtoConfig,
						NbOfAttempts:         client1.NbOfAttempts,
						FirstRetryDelay:      client1.FirstRetryDelay,
						RetryIncrementFactor: client1.RetryIncrementFactor,
					})

					So(res[1], ShouldResemble, file.Client{
						Name:                 client2.Name,
						Protocol:             client2.Protocol,
						Disabled:             client2.Disabled,
						LocalAddress:         client2.LocalAddress.String(),
						ProtoConfig:          client2.ProtoConfig,
						NbOfAttempts:         client2.NbOfAttempts,
						FirstRetryDelay:      client2.FirstRetryDelay,
						RetryIncrementFactor: client2.RetryIncrementFactor,
					})
				})
			})
		})
	})
}
