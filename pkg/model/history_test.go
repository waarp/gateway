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

func TestHistoryBeforeInsert(t *testing.T) {
	Convey("Given a `Transfer` instance", t, func() {
		trans := &TransferHistory{}

		Convey("When calling the `BeforeInsert` method", func() {
			err := trans.BeforeInsert(nil)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})

			Convey("Then the transfer owner should be 'test_gateway'", func() {
				So(trans.Owner, ShouldEqual, "test_gateway")
			})
		})
	})
}

func TestHistoryValidateInsert(t *testing.T) {
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

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the owner is missing", func() {
				hist.Owner = ""

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the error should say the owner is missing", func() {
						So(err.Error(), ShouldEqual, "The transfer's owner cannot "+
							"be empty")
					})
				})
			})

			Convey("Given that the rule name is missing", func() {
				hist.Rule = ""

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the error should say the rule is missing", func() {
						So(err.Error(), ShouldEqual, "The transfer's rule "+
							"cannot be empty")
					})
				})
			})

			Convey("Given that the account is missing", func() {
				hist.Account = ""

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the error should say the source is missing", func() {
						So(err.Error(), ShouldEqual, "The transfer's account "+
							"cannot be empty")
					})
				})
			})

			Convey("Given that the agent is missing", func() {
				hist.Agent = ""

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the error should say the destination is missing", func() {
						So(err.Error(), ShouldEqual, "The transfer's agent "+
							"cannot be empty")
					})
				})
			})

			Convey("Given that the filename is missing", func() {
				hist.DestFilename = ""

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the error should say the filename is missing", func() {
						So(err.Error(), ShouldEqual, "The transfer's destination filename "+
							"cannot be empty")
					})
				})
			})

			Convey("Given that the protocol is missing", func() {
				hist.Protocol = ""

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the error should say the protocol is missing", func() {
						So(err.Error(), ShouldEqual, "'' is not a valid protocol")
					})
				})
			})

			Convey("Given that the starting date is missing", func() {
				hist.Start = time.Time{}

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the error should say the start date is missing", func() {
						So(err.Error(), ShouldEqual, "The transfer's start "+
							"date cannot be empty")
					})
				})
			})

			Convey("Given that the end date is missing", func() {
				hist.Stop = time.Time{}

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the error should say the end date is missing", func() {
						So(err.Error(), ShouldEqual, "The transfer's end "+
							"date cannot be empty")
					})
				})
			})

			Convey("Given that the end date is before the ", func() {
				hist.Stop = hist.Start.AddDate(0, 0, -1)

				Convey("When calling the 'ValidateInsert' function", func() {
					ses, err := db.BeginTransaction()
					So(err, ShouldBeNil)

					err = hist.ValidateInsert(ses)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the error should say the end date is anterior", func() {
						So(err.Error(), ShouldEqual, "The transfer's end "+
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
				testTransferStatus(tc, "ValidateInsert", hist, db)
			}
		})
	})
}

func TestHistoryValidateUpdate(t *testing.T) {
	Convey("Given a `Transfer` instance", t, func() {
		hist := &TransferHistory{
			Status: StatusDone,
			Start:  time.Now(),
			Stop:   time.Now().AddDate(0, 0, 1),
		}

		Convey("Given that the entry is valid", func() {

			Convey("When calling the `ValidateUpdate` method", func() {
				err := hist.ValidateUpdate(nil, 0)

				Convey("Then it should not return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that the entry changes the ID", func() {
			hist.ID = 1

			Convey("When calling the `ValidateUpdate` method", func() {
				err := hist.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the ID cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's ID cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the rule", func() {
			hist.Rule = "rule"

			Convey("When calling the `ValidateUpdate` method", func() {
				err := hist.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the rule cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's rule cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the account", func() {
			hist.Account = "source"

			Convey("When calling the `ValidateUpdate` method", func() {
				err := hist.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the source cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's account cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the agent", func() {
			hist.Agent = "dest"

			Convey("When calling the `ValidateUpdate` method", func() {
				err := hist.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the destination cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's agent "+
						"cannot be changed")
				})
			})
		})

		Convey("Given that the entry changes the owner", func() {
			hist.Owner = "owner"

			Convey("When calling the `ValidateUpdate` method", func() {
				err := hist.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the owner cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's owner cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the filename", func() {
			hist.SourceFilename = "file"

			Convey("When calling the `ValidateUpdate` method", func() {
				err := hist.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the filename cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's source filename cannot be "+
						"changed")
				})
			})
		})

		Convey("Given that the entry changes the protocol", func() {
			hist.Protocol = "sftp"

			Convey("When calling the `ValidateUpdate` method", func() {
				err := hist.ValidateUpdate(nil, 0)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
				})

				Convey("Then the error should say that the protocol cannot be changed", func() {
					So(err.Error(), ShouldEqual, "The transfer's protocol "+
						"cannot be changed")
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
			testTransferStatus(tc, "ValidateUpdate", hist, nil)
		}
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
	ValidateInsert(database.Accessor) error
}
type testUpdateValidator interface {
	ValidateUpdate(database.Accessor, uint64) error
}

func testTransferStatus(tc statusTestCase, method string, target interface{}, db *database.Db) {
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
			var (
				ses database.Accessor
				err error
			)
			if db != nil {
				ses, err = db.BeginTransaction()
				So(err, ShouldBeNil)
			}

			if t, ok := target.(testInsertValidator); ok && method == "ValidateInsert" {
				err = t.ValidateInsert(ses)
			} else if t, ok := target.(testUpdateValidator); ok && method == "ValidateUpdate" {
				err = t.ValidateUpdate(ses, 0)
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
