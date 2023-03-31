package migrations

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/smartystreets/goconvey/convey"
)

// In some instances, we need some SQL identifier to be quoted. This will quote
// the given identifier with the appropriate character based on the dialect of
// the given database.
func quote(db Actions, identifier string) string {
	if db.GetDialect() == MySQL {
		return fmt.Sprintf("`%s`", identifier)
	}

	return fmt.Sprintf(`"%s"`, identifier)
}

func nextRowShouldBe(rows *sql.Rows, expectedVals ...any) {
	vals := make([]any, len(expectedVals))
	valsPtr := make([]any, len(expectedVals))

	for i := range vals {
		vals[i] = reflect.New(reflect.TypeOf(expectedVals[i])).Elem().Interface()
		valsPtr[i] = &vals[i]
	}

	convey.So(rows.Next(), convey.ShouldBeTrue)
	convey.So(rows.Scan(valsPtr...), convey.ShouldBeNil)

	for i := range vals {
		convey.So(vals[i], convey.ShouldEqual, expectedVals[i])
	}
}
