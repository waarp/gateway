package types

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTransferErrorCode(t *testing.T) {
	Convey("Given a valid transfer error", t, func() {
		ec := TeOk

		Convey("Then has a string representation", func() {
			So(ec.String(), ShouldEqual, "TeOk")
		})

		Convey("Then its valid method should return true", func() {
			So(ec.IsValid(), ShouldBeTrue)
		})

		Convey("When it is unmarshaled from json", func() {
			err := json.Unmarshal([]byte(`"TeUnknown"`), &ec)
			So(err, ShouldBeNil)

			Convey("Then its value is a string representation", func() {
				So(ec, ShouldEqual, TeUnknown)
			})
		})
	})

	Convey("Given an invalid transfer error", t, func() {
		ec := TransferErrorCode(212)

		Convey("Then has a string representation", func() {
			So(ec.String(), ShouldEqual, "TransferErrorCode(212)")
		})

		Convey("Then its valid method should return false", func() {
			So(ec.IsValid(), ShouldBeFalse)
		})
	})

	Convey("Testing JSON marshaling/unmarshaling", t, func() {
		Convey("It should be marshaled as a string", func() {
			v, err := json.Marshal(TeOk)
			So(err, ShouldBeNil)

			So(v, ShouldResemble, []byte(`"TeOk"`))
		})

		Convey("Unmarshaling from JSON should return the corresponding value", func() {
			var ec TransferErrorCode

			err := json.Unmarshal([]byte(`"TeUnknown"`), &ec)
			So(err, ShouldBeNil)

			So(ec, ShouldEqual, TeUnknown)
		})
	})
	Convey("Testing database serialization/deserialization", t, func() {
		Convey("It should be serialized as a string", func() {
			v, err := TeOk.Value()
			So(err, ShouldBeNil)

			So(v, ShouldResemble, `TeOk`)
		})

		Convey("It should scan a string representation from the database", func() {
			var ec TransferErrorCode

			err := ec.Scan(`TeUnknown`)
			So(err, ShouldBeNil)

			So(ec, ShouldEqual, TeUnknown)
		})
		Convey("It should scan a []byte representation from the database", func() {
			var ec TransferErrorCode

			err := ec.Scan([]byte(`TeFileNotFound`))
			So(err, ShouldBeNil)

			So(ec, ShouldEqual, TeFileNotFound)
		})
		Convey("It should return an error if the database returns another type", func() {
			var ec TransferErrorCode

			err := ec.Scan(10)
			So(err, ShouldBeError)
		})
		Convey("It should return TeOk from an empty string (no error)", func() {
			var ec TransferErrorCode

			err := ec.Scan(``)
			So(err, ShouldBeNil)

			So(ec, ShouldEqual, TeOk)
		})
		Convey("It should return TeUnknown if the error code does not exist", func() {
			var ec TransferErrorCode

			err := ec.Scan(`foobar`)
			So(err, ShouldBeNil)

			So(ec, ShouldEqual, TeUnknown)
		})
	})
	Convey("Testing xorm serialization/deserialization", t, func() {
		Convey("It should be serialized as a string", func() {
			v, err := TeOk.ToDB()
			So(err, ShouldBeNil)

			So(v, ShouldResemble, []byte(`TeOk`))
		})

		Convey("It should scan a []byte representation from the database", func() {
			var ec TransferErrorCode

			err := ec.FromDB([]byte(`TeFileNotFound`))
			So(err, ShouldBeNil)

			So(ec, ShouldEqual, TeFileNotFound)
		})
		Convey("It should return TeUnknown if the error code does not exist", func() {
			var ec TransferErrorCode

			err := ec.FromDB([]byte(`foobar`))
			So(err, ShouldBeNil)

			So(ec, ShouldEqual, TeUnknown)
		})
	})
}

func TestTransferError(t *testing.T) {
	Convey("Testing TransferError", t, func() {
		Convey("Given a complete and valid transfer error", func() {
			terr := NewTransferError(TeUnimplemented, "more info")

			Convey("Then it implements the error interface", func() {
				So(terr, ShouldBeError)
			})

			Convey("When Error() is called", func() {
				errStr := terr.Error()

				Convey("Then it should contain the error code", func() {
					So(errStr, ShouldContainSubstring, "TeUnimplemented")
				})
				Convey("Then it should contain the error detail", func() {
					So(errStr, ShouldContainSubstring, "more info")
				})
			})
		})

		Convey("Given a TransferError with no details", func() {
			terr := NewTransferError(TeOk, "")

			Convey("When Error() is called", func() {
				Convey("Then it should not contain any detail", func() {
					So(terr.Error(), ShouldEqual, "TransferError(TeOk)")
				})
			})
		})

		Convey("Creating an error with an invalid ErrorCode should panic", func() {
			So(func() { _ = NewTransferError(TransferErrorCode(212), "") }, ShouldPanic)
			So(func() { _ = NewTransferError(212, "") }, ShouldPanic)
		})

		Convey("A new transfer error with no error has no details", func() {
			e := NewTransferError(TeOk, "some details about the error")
			So(e.Details, ShouldEqual, "")
		})

		Convey("Given a new default TransferError", func() {
			var e TransferError

			Convey("Then it has no error", func() {
				So(e.Code, ShouldEqual, TeOk)
			})
			Convey("Then it has no details", func() {
				So(e.Details, ShouldEqual, "")
			})
		})

		Convey("When UnmarshalError is called with an error", func() {
			e1 := NewTransferError(TeOk, "")
			e2 := NewTransferError(TeUnknown, "error info")

			err := json.Unmarshal([]byte(`{"code":"TeBadSize","details":"foobar"}`), &e1)
			So(err, ShouldBeNil)
			err = json.Unmarshal([]byte(`{"code":"TeOk","details":""}`), &e2)
			So(err, ShouldBeNil)

			Convey("Then it must not change the TransferError", func() {
				So(e1.Code, ShouldEqual, TeOk)
				So(e1.Details, ShouldEqual, "")

				So(e2.Code, ShouldEqual, TeUnknown)
				So(e2.Details, ShouldEqual, "error info")
			})
		})
	})
}
