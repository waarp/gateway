package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMaskToPerm(t *testing.T) {
	Convey("Testing the permission mask converter", t, func() {
		Convey("Given a permission mask", func() {
			mask := PermTransfersRead | PermTransfersWrite |
				PermServersWrite |
				PermPartnersRead | PermPartnersDelete |
				PermRulesRead | PermRulesWrite | PermRulesDelete |
				PermUsersWrite | PermUsersDelete |
				PermAdminRead | PermAdminWrite

			Convey("When calling the MaskToPerms function", func() {
				perms := MaskToPerms(mask)

				Convey("Then it should return the correct permissions", func() {
					exp := &Permissions{
						Transfers:      "rw-",
						Servers:        "-w-",
						Partners:       "r-d",
						Rules:          "rwd",
						Users:          "-wd",
						Administration: "rw-",
					}
					So(perms, ShouldResemble, exp)
				})
			})
		})

		Convey("Given a full mask", func() {
			mask := PermAll

			Convey("When calling the MaskToPerms function", func() {
				perms := MaskToPerms(mask)

				Convey("Then it should return the correct permissions", func() {
					exp := &Permissions{
						Transfers:      "rw-",
						Servers:        "rwd",
						Partners:       "rwd",
						Rules:          "rwd",
						Users:          "rwd",
						Administration: "rwd",
					}
					So(perms, ShouldResemble, exp)
				})
			})
		})
	})
}

func TestPermsToMask(t *testing.T) {
	Convey("Testing the permission mask converter", t, func() {
		Convey("Given a permission mask and a permission string", func() {
			perms := Permissions{
				Transfers:      "r--",
				Servers:        "rw-",
				Partners:       "r-d",
				Rules:          "-wd",
				Users:          "-w-",
				Administration: "r--",
			}

			Convey("When calling the PermsToMask function", func() {
				newMask, err := PermsToMask(&perms)
				So(err, ShouldBeNil)

				Convey("Then it should return the correct mask", func() {
					exp := PermTransfersRead |
						PermServersRead | PermServersWrite |
						PermPartnersRead | PermPartnersDelete |
						PermRulesWrite | PermRulesDelete |
						PermUsersWrite |
						PermAdminRead

					actual := MaskToPerms(newMask)
					expected := MaskToPerms(exp)
					So(actual, ShouldResemble, expected)
				})
			})
		})
	})
}
