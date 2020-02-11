package model

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// Commented out when the function it tests has been commented out
// func TestTransferStatusIsValid(t *testing.T) {
// 	testCases := []struct {
// 		status   TransferStatus
// 		expected bool
// 	}{
// 		{StatusPlanned, true},
// 		{StatusTransfer, true},
// 		{StatusDone, true},
// 		{StatusError, true},
// 		{"toto", false},
// 	}
//
// 	Convey("Given a TransferStatus", t, func() {
// 		var status TransferStatus
//
// 		for _, tc := range testCases {
// 			Convey(fmt.Sprintf("When the status is %s", tc.status), func() {
// 				status = tc.status
//
// 				if tc.expected == true {
// 					Convey("Then it is valid", func() {
// 						So(status.isValid(), ShouldBeTrue)
// 					})
// 				} else {
// 					Convey("Then it is not valid", func() {
// 						So(status.isValid(), ShouldBeFalse)
// 					})
// 				}
// 			})
// 		}
// 	})
// }

func TestTransferStatusValidateForTransfer(t *testing.T) {
	testCases := []struct {
		status   TransferStatus
		expected bool
	}{
		{StatusPlanned, true},
		{StatusRunning, true},
		{StatusDone, false},
		{StatusError, false},
		{"toto", false},
	}

	Convey("Given a TransferStatus", t, func() {
		var status TransferStatus

		for _, tc := range testCases {
			Convey(fmt.Sprintf("When the status is %s", tc.status), func() {
				status = tc.status

				if tc.expected == true {
					Convey("Then it is valid for a transfer", func() {
						So(validateStatusForTransfer(status), ShouldBeTrue)
					})
				} else {
					Convey("Then it is not valid for a transfer", func() {
						So(validateStatusForTransfer(status), ShouldBeFalse)
					})
				}
			})
		}
	})
}

func TestTransferStatusValidateForHistory(t *testing.T) {
	testCases := []struct {
		status   TransferStatus
		expected bool
	}{
		{StatusPlanned, false},
		{StatusRunning, false},
		{StatusDone, true},
		{StatusError, true},
		{"toto", false},
	}

	Convey("Given a TransferStatus", t, func() {
		var status TransferStatus

		for _, tc := range testCases {
			Convey(fmt.Sprintf("When the status is %s", tc.status), func() {
				status = tc.status

				if tc.expected == true {
					Convey("Then it is valid for a transfer", func() {
						So(validateStatusForHistory(status), ShouldBeTrue)
					})
				} else {
					Convey("Then it is not valid for a transfer", func() {
						So(validateStatusForHistory(status), ShouldBeFalse)
					})
				}
			})
		}
	})
}
