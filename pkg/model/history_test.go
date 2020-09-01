package model

import (
	"fmt"
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	. "github.com/smartystreets/goconvey/convey"
)

func TestHistoryTableName(t *testing.T) {
	Convey("Given a `TransferHistory` instance", t, func() {
		hist := &TransferHistory{}

		Convey("When calling the 'TableName' method", func() {
			name := hist.TableName()

			Convey("Then it should return the name of the history table", func() {
				So(name, ShouldEqual, "transfer_history")
			})
		})
	})
}

func TestHistoryValidate(t *testing.T) {
	Convey("Given a database", t, func() {
		db := database.GetTestDatabase()

		Convey("Given a new history entry", func() {
			hist := &TransferHistory{
				ID:             1,
				Rule:           "rule",
				IsServer:       true,
				IsSend:         true,
				Agent:          "from",
				Account:        "to",
				SourceFilename: "test/source/path",
				DestFilename:   "test/source/path",
				Start:          time.Now(),
				Stop:           time.Now(),
				Protocol:       "sftp",
				Status:         "DONE",
				Owner:          database.Owner,
			}

			Convey("Given that the new transfer is valid", func() {

				Convey("When calling the 'Validate' function", func() {
					err := hist.Validate(db)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the rule name is missing", func() {
				hist.Rule = ""

				Convey("When calling the 'Validate' function", func() {
					err := hist.Validate(db)

					Convey("Then the error should say the rule is missing", func() {
						So(err, ShouldBeError, "the transfer's rule "+
							"cannot be empty")
					})
				})
			})

			Convey("Given that the account is missing", func() {
				hist.Account = ""

				Convey("When calling the 'Validate' function", func() {
					err := hist.Validate(db)

					Convey("Then the error should say the source is missing", func() {
						So(err, ShouldBeError, "the transfer's account "+
							"cannot be empty")
					})
				})
			})

			Convey("Given that the agent is missing", func() {
				hist.Agent = ""

				Convey("When calling the 'Validate' function", func() {
					err := hist.Validate(db)

					Convey("Then the error should say the destination is missing", func() {
						So(err, ShouldBeError, "the transfer's agent "+
							"cannot be empty")
					})
				})
			})

			Convey("Given that the filename is missing", func() {
				hist.DestFilename = ""

				Convey("When calling the 'Validate' function", func() {
					err := hist.Validate(db)

					Convey("Then the error should say the filename is missing", func() {
						So(err, ShouldBeError, "the transfer's destination filename "+
							"cannot be empty")
					})
				})
			})

			Convey("Given that the protocol is invalid", func() {
				hist.Protocol = "invalid"

				Convey("When calling the 'Validate' function", func() {
					err := hist.Validate(db)

					Convey("Then the error should say the protocol is missing", func() {
						So(err, ShouldBeError, "'invalid' is not a valid protocol")
					})
				})
			})

			Convey("Given that the starting date is missing", func() {
				hist.Start = time.Time{}

				Convey("When calling the 'Validate' function", func() {
					err := hist.Validate(db)

					Convey("Then the error should say the start date is missing", func() {
						So(err, ShouldBeError, "the transfer's start "+
							"date cannot be empty")
					})
				})
			})

			Convey("Given that the end date is missing", func() {
				hist.Stop = time.Time{}

				Convey("When calling the 'Validate' function", func() {
					err := hist.Validate(db)

					Convey("Then the error should say the end date is missing", func() {
						So(err, ShouldBeError, "the transfer's end "+
							"date cannot be empty")
					})
				})
			})

			Convey("Given that the end date is before the ", func() {
				hist.Stop = hist.Start.AddDate(0, 0, -1)

				Convey("When calling the 'Validate' function", func() {
					err := hist.Validate(db)

					Convey("Then the error should say the end date is anterior", func() {
						So(err, ShouldBeError, "the transfer's end "+
							"date cannot be anterior to the start date")
					})
				})
			})

			statusTestCases := []statusTestCase{
				{StatusPlanned, false},
				{StatusRunning, false},
				{StatusDone, true},
				{StatusError, true},
				{"toto", false},
			}
			for _, tc := range statusTestCases {
				testTransferStatus(tc, "Validate", hist, db)
			}
		})
	})
}

//
// Test utils
//

type statusTestCase struct {
	status          TransferStatus
	expectedSuccess bool
}
type testInsertValidator interface {
	Validate(database.Accessor) error
}

func testTransferStatus(tc statusTestCase, method string, target interface{}, db *database.DB) {
	Convey(fmt.Sprintf("Given the status is set to '%s'", tc.status), func() {
		var typeName string
		if t, ok := target.(*TransferHistory); ok {
			t.Status = tc.status
			typeName = "transfer history"
		}
		if t, ok := target.(*Transfer); ok {
			t.Status = tc.status
			typeName = "transfer"
		}

		Convey(fmt.Sprintf("When the method `%s` is called", method), func() {
			var err error

			if t, ok := target.(testInsertValidator); ok && method == "Validate" {
				err = t.Validate(db)
			}

			if tc.expectedSuccess {
				Convey("Then it should not return any error", func() {
					So(err, ShouldBeNil)
				})
			} else {
				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the status is invalid", func() {
					expectedError := fmt.Sprintf(
						"'%s' is not a valid %s status",
						tc.status, typeName,
					)
					So(err, ShouldBeError, expectedError)
				})
			}
		})
	})
}
