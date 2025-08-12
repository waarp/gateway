package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestTransferStatusFromString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		status   string
		valid    bool
		expected TransferStatus
	}{
		{"PLANNED", true, StatusPlanned},
		{"AVAILABLE", true, StatusAvailable},
		{"RUNNING", true, StatusRunning},
		{"INTERRUPTED", true, StatusInterrupted},
		{"PAUSED", true, StatusPaused},
		{"CANCELLED", true, StatusCancelled},
		{"DONE", true, StatusDone},
		{"ERROR", true, StatusError},
		{"toto", false, ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.status, func(t *testing.T) {
			t.Parallel()

			status, valid := StatusFromString(testCase.status)
			assert.Equal(t, testCase.valid, valid)
			assert.Equal(t, testCase.expected, status)
		})
	}
}

func TestTransferStatusValidateForTransfer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		status   TransferStatus
		expected bool
	}{
		{StatusPlanned, true},
		{StatusAvailable, true},
		{StatusRunning, true},
		{StatusDone, false},
		{StatusError, true},
		{StatusCancelled, false},
		{"toto", false},
	}

	for _, testCase := range testCases {
		t.Run(string(testCase.status), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, ValidateStatusForTransfer(testCase.status))
		})
	}
}

func TestTransferStatusValidateForHistory(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		status   TransferStatus
		expected bool
	}{
		{StatusPlanned, false},
		{StatusAvailable, false},
		{StatusRunning, false},
		{StatusDone, true},
		{StatusError, false},
		{StatusCancelled, true},
		{"toto", false},
	}

	for _, testCase := range testCases {
		t.Run(string(testCase.status), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, ValidateStatusForHistory(testCase.status))
		})
	}
}
