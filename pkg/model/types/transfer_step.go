package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// TransferStep represents the different steps of a transfer.
//
//go:generate stringer -type TransferStep
type TransferStep uint8

const (
	// StepNone is the state of a transfer before the transfer start, or after
	// the transfer end (if no error occurred).
	StepNone TransferStep = iota // NONE

	// StepSetup is the state of a transfer while connecting and making the request.
	StepSetup // SETUP

	// StepPreTasks is the state of a transfer while it's running the pre tasks.
	StepPreTasks // PRE TASKS

	// StepData is the state of a transfer while transferring the file data.
	StepData // DATA

	// StepPostTasks is the state of a transfer while it's running the post tasks.
	StepPostTasks // POST TASKS

	// StepErrorTasks is the state of a transfer while it's running the error tasks.
	StepErrorTasks // ERROR TASKS

	// StepFinalization is the state of a transfer while finalizing.
	StepFinalization // FINALIZATION
)

// tsFromString gets the right TransferStep from the given string using
// the code generated by stringer.
func tsFromString(s string) TransferStep {
	if len(s) == 0 {
		return StepNone
	}

	idx := strings.Index(_TransferStep_name, s)
	if idx == -1 {
		return StepNone
	}

	tsInt := 1

	for i, v := range _TransferStep_index {
		if int(v) == idx {
			tsInt = i
		}
	}

	return TransferStep(tsInt)
}

// IsValid returns true if the transfer step is defined.
func (ts TransferStep) IsValid() bool {
	return ts <= StepFinalization
}

// MarshalJSON implements json.Marshaler. To have more significance, the JSON
// representation of a TransferStep is not its int value, but its string
// representation.
func (ts TransferStep) MarshalJSON() ([]byte, error) {
	return []byte(`"` + ts.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (ts *TransferStep) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return fmt.Errorf("%w", err)
	}

	*ts = tsFromString(str)

	return nil
}

// Value implements database/driver.Valuer. To have more significance, the
// representation of a TransferStep is not its int value, but its string
// representation.
func (ts TransferStep) Value() (driver.Value, error) {
	return ts.String(), nil
}

// Scan implements database/sql.Scanner. It takes a string and returns the matching
// TransferStep.
func (ts *TransferStep) Scan(v interface{}) error {
	switch v := v.(type) {
	case string:
		*ts = tsFromString(v)
	case []byte:
		*ts = tsFromString(string(v))
	default:
		//nolint:goerr113 // too specific to have a base error
		return fmt.Errorf("cannot scan %+v of type %T into a TransferStep",
			v, v)
	}

	return nil
}

// FromDB implements xorm/core.Conversion. As xorm ignores standard converters
// for non-struct types (Value() and Scan()), thus must be mapped to xorm own
// conversion interface.
func (ts *TransferStep) FromDB(v []byte) error {
	return ts.Scan(v)
}

// ToDB implements xorm/core.Conversion. As xorm ignores standard converters
// for non-struct types (Value() and Scan()), thus must be mapped to xorm own
// conversion interface.
func (ts TransferStep) ToDB() ([]byte, error) {
	v, err := ts.Value()

	//nolint:forcetypeassert //no need, the type assertion will always succeed
	return []byte(v.(string)), err
}
