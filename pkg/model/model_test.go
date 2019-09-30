package model

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTableNames(t *testing.T) {

	Convey("Testing the database model", t, func() {
		So((&Account{}).TableName(), ShouldResemble, "accounts")
		So((&CertChain{}).TableName(), ShouldResemble, "certificates")
		So((&Partner{}).TableName(), ShouldResemble, "partners")
		So((&User{}).TableName(), ShouldResemble, "users")
	})
}
