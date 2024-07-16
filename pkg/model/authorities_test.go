package model

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

func TestAuthorityBeforeWrite(t *testing.T) {
	Convey("Given a database with 1 existing authority", t, func(c C) {
		db := database.TestDatabase(c)

		existing := &Authority{
			Name:           "existing",
			Type:           testAuthority,
			PublicIdentity: "existing identity value",
			ValidHosts:     []string{"1.2.3.4", "example.com"},
		}
		So(db.Insert(existing).Run(), ShouldBeNil)

		newAuthority := &Authority{
			Name:           "new_authority",
			Type:           testAuthority,
			PublicIdentity: "new identity value",
			ValidHosts:     []string{"9.8.7.6", "waarp.org"},
		}

		Convey("Given a new valid new authority", func() {
			Convey("Then, calling the 'BeforeWrite' hook should return no error", func() {
				So(newAuthority.BeforeWrite(db), ShouldBeNil)
			})
		})

		Convey("Given a new authority which is missing a name", func() {
			newAuthority.Name = ""

			Convey("Then, calling the 'BeforeWrite' hook should return an error", func() {
				So(newAuthority.BeforeWrite(db), ShouldBeError, "the authority is missing a name")
			})
		})

		Convey("Given a new authority whose name is already taken", func() {
			newAuthority.Name = existing.Name

			Convey("Then, calling the 'BeforeWrite' hook should return an error", func() {
				So(newAuthority.BeforeWrite(db), ShouldBeError, fmt.Sprintf(
					"an %s named %q already exists", NameAuthority, existing.Name))
			})
		})

		Convey("Given a new authority which is missing a type", func() {
			newAuthority.Type = ""

			Convey("Then, calling the 'BeforeWrite' hook should return an error", func() {
				So(newAuthority.BeforeWrite(db), ShouldBeError, "the authority is missing a type")
			})
		})

		Convey("Given a new authority whose type is unknown", func() {
			newAuthority.Type = "unknown type"

			Convey("Then, calling the 'BeforeWrite' hook should return an error", func() {
				So(newAuthority.BeforeWrite(db), ShouldBeError,
					`"unknown type" is not a valid authority type`)
			})
		})

		Convey("Given a new authority whose value is missing", func() {
			newAuthority.PublicIdentity = ""

			Convey("Then, calling the 'BeforeWrite' hook should return an error", func() {
				So(newAuthority.BeforeWrite(db), ShouldBeError,
					"the authority is missing a public identity value")
			})
		})

		Convey("Given a new authority whose value is invalid", func() {
			newAuthority.PublicIdentity = invalidAuthorityVal

			Convey("Then, calling the 'BeforeWrite' hook should return an error", func() {
				So(newAuthority.BeforeWrite(db), ShouldBeError, fmt.Sprintf(
					"could not validate the authority's public identity value: %v",
					errInvalidAuthorityVal))
			})
		})
	})
}

func TestAuthorityAfterUpdate(t *testing.T) {
	Convey("Given a database with 2 existing authorities", t, func(c C) {
		db := database.TestDatabase(c)

		existing1 := &Authority{
			Name:           "existing1",
			Type:           testAuthority,
			PublicIdentity: "existing identity value 1",
			ValidHosts:     []string{"1.1.1.1", "2.2.2.2"},
		}
		So(db.Insert(existing1).Run(), ShouldBeNil)

		existing2 := &Authority{
			Name:           "existing2",
			Type:           testAuthority,
			PublicIdentity: "existing identity value 2",
			ValidHosts:     []string{"3.3.3.3", "4.4.4.4"},
		}
		So(db.Insert(existing2).Run(), ShouldBeNil)

		var oldHosts Hosts
		So(db.Select(&oldHosts).OrderBy("host", true).Run(), ShouldBeNil)
		So(oldHosts, ShouldHaveLength, 4)
		So(oldHosts[0].Host, ShouldEqual, existing1.ValidHosts[0])
		So(oldHosts[1].Host, ShouldEqual, existing1.ValidHosts[1])
		So(oldHosts[2].Host, ShouldEqual, existing2.ValidHosts[0])
		So(oldHosts[3].Host, ShouldEqual, existing2.ValidHosts[1])

		Convey("When calling the `AfterWrite` hook", func() {
			existing2.ValidHosts = []string{"waarp.org"}

			So(existing2.AfterUpdate(db), ShouldBeNil)

			Convey("Then it should have updated the valid hosts", func() {
				var newHosts Hosts
				So(db.Select(&newHosts).OrderBy("host", true).Run(), ShouldBeNil)
				So(newHosts, ShouldHaveLength, 3)
				So(newHosts[0].Host, ShouldEqual, existing1.ValidHosts[0])
				So(newHosts[1].Host, ShouldEqual, existing1.ValidHosts[1])
				So(newHosts[2].Host, ShouldEqual, existing2.ValidHosts[0])
			})
		})
	})
}

func TestAuthorityAfterRead(t *testing.T) {
	Convey("Given a database with 1 existing authority", t, func(c C) {
		db := database.TestDatabase(c)

		existing1 := &Authority{
			Name:           "existing1",
			Type:           testAuthority,
			PublicIdentity: "existing identity value 1",
			ValidHosts:     []string{"1.1.1.1", "2.2.2.2"},
		}
		So(db.Insert(existing1).Run(), ShouldBeNil)

		existing2 := &Authority{
			Name:           "existing2",
			Type:           testAuthority,
			PublicIdentity: "existing identity value 2",
			ValidHosts:     []string{"3.3.3.3", "4.4.4.4"},
		}
		So(db.Insert(existing2).Run(), ShouldBeNil)

		Convey("When calling the `AfterRead` hook", func() {
			authority := &Authority{
				ID:             existing2.ID,
				Name:           existing2.Name,
				Type:           existing2.Type,
				PublicIdentity: existing2.PublicIdentity,
			}
			So(authority.AfterRead(db), ShouldBeNil)

			Convey("Then it should have filled the valid hosts", func() {
				So(authority.ValidHosts, ShouldResemble, existing2.ValidHosts)
			})
		})
	})
}
