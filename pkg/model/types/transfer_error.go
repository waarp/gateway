package types

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// TransferErrorCode is a closed list of the defined error codes
//
//go:generate stringer -type TransferErrorCode
type TransferErrorCode uint8

const (
	// TeOk is the default value, it indicates that there is no error.
	TeOk TransferErrorCode = iota

	// TeUnknown indicates a valid but undefined error.
	TeUnknown

	// TeInternal indicates an error internal to the gateway (db connection
	// lost, etc.).
	TeInternal

	// TeUnimplemented indicates that a remote asked for a unimplemented feature.
	TeUnimplemented

	// TeConnection means that the connection to a remote could not be
	// established. It is the most general. A more specific error should be
	// used if applicable.
	TeConnection

	// TeConnectionReset means "connection reset by peer".
	TeConnectionReset

	// TeUnknownRemote should be used when a remote is not known in our
	// database. For Security reasons, its use is discouraged for incoming
	// connection and TeBadAuthentication should be preferred, as it leaks
	// information (the account exist, but the password is wrong).
	TeUnknownRemote

	// TeExceededLimit should be used when there are no available resources to
	// process the request (connection limit, transfer limit, etc.)
	TeExceededLimit

	// TeBadAuthentication indicates that the given credentials are false.
	TeBadAuthentication

	// TeDataTransfer should be used when an error occurred during the transfer
	// of data. A more precise error should be used if applicable.
	TeDataTransfer

	// TeIntegrity indicates that the integrity check did not pass.
	TeIntegrity

	// TeFinalization should be used when an error occurs during the
	// finalizaion of the transfer.
	TeFinalization

	// TeExternalOperation should be used when an error occurred during the
	// execution of the pre-/post-/error-tasks.
	TeExternalOperation

	// TeWarning should be used when a pre-/post-/error-task ended with a
	// warning code. This type of error does not stop the transfer.
	TeWarning

	// TeStopped should be used when the transfer has been stopped.
	TeStopped

	// TeCanceled should be used when the transfer has been canceled.
	TeCanceled

	// TeFileNotFound indicates that the asked file has not been found.
	TeFileNotFound

	// TeForbidden indicates that the remote has no rights to perform an action
	// (file reading/writing as well as administrative operations).
	TeForbidden

	// TeBadSize should be used whenever there is an error related to the size
	// of a file (it exceeds quota, there is not enough space left on the
	// destination drive, etc.)
	TeBadSize

	// TeShuttingDown should be used when the gateway is shutting down.
	TeShuttingDown

	// Used as a marker to know the number of defined errorcodes.
	endoferrorcodes
)

// TecFromString gets the right TransferErrorCode from the given string using
// the code generated by stringer.
func TecFromString(s string) TransferErrorCode {
	if len(s) == 0 {
		return TeOk
	}

	idx := strings.Index(_TransferErrorCode_name, s)
	if idx == -1 {
		return TeUnknown
	}

	tecInt := 1

	for i, v := range _TransferErrorCode_index {
		if int(v) == idx {
			tecInt = i
		}
	}

	return TransferErrorCode(tecInt)
}

// IsValid returns true if the errorcode is defined.
func (tec TransferErrorCode) IsValid() bool {
	return tec < endoferrorcodes
}

// MarshalJSON implements json.Marshaler. To have more significance, the JSON
// representation of a TransferErrorCode is not its int value, but its string
// representation.
func (tec TransferErrorCode) MarshalJSON() ([]byte, error) {
	return []byte(`"` + tec.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler. This operation is not supported:
// a user cannot modify the error code from an external API, so unmarshaling an
// error code from json is a noop.
func (tec TransferErrorCode) UnmarshalJSON([]byte) error {
	return nil
}

// Value implements database/driver.Valuer. To have more significance, the
// representation of a TransferErrorCode is not its int value, but its string
// representation.
func (tec TransferErrorCode) Value() (driver.Value, error) {
	return tec.String(), nil
}

// Scan implements database/sql.Scanner. It takes a string and returns the matching
// TransferErrorCode.
func (tec *TransferErrorCode) Scan(v interface{}) error {
	switch v := v.(type) {
	case string:
		*tec = TecFromString(v)
	case []byte:
		*tec = TecFromString(string(v))
	default:
		//nolint:goerr113 // too specific to have a base error
		return fmt.Errorf("cannot scan %+v of type %T into a TransferErrorCode",
			v, v)
	}

	return nil
}

// FromDB implements xorm/core.Conversion. As xorm ignores standard converters
// for non-struct types (Value() and Scan()), thos must be mapped to xorm own
// conversion interface.
func (tec *TransferErrorCode) FromDB(v []byte) error {
	return tec.Scan(v)
}

// ToDB implements xorm/core.Conversion. As xorm ignores standard converters
// for non-struct types (Value() and Scan()), thos must be mapped to xorm own
// conversion interface.
func (tec TransferErrorCode) ToDB() ([]byte, error) {
	v, err := tec.Value()

	//nolint:forcetypeassert //no need, the type assertion will always succeed
	return []byte(v.(string)), err
}

// R66Code returns the error code as a single character usable by R66.
//
//nolint:funlen,cyclop // cannot be shorten without adding complexity
func (tec TransferErrorCode) R66Code() rune {
	switch tec {
	case TeOk:
		return 'O'
	case TeInternal:
		return 'I'
	case TeUnimplemented:
		return 'U'
	case TeConnection:
		return 'C'
	case TeConnectionReset:
		return 'D'
	case TeUnknownRemote:
		return 'N'
	case TeExceededLimit:
		return 'l'
	case TeBadAuthentication:
		return 'A'
	case TeDataTransfer:
		return 'T'
	case TeIntegrity:
		return 'M'
	case TeFinalization:
		return 'F'
	case TeExternalOperation:
		return 'E'
	case TeWarning:
		return 'w'
	case TeStopped:
		return 'H'
	case TeCanceled:
		return 'K'
	case TeFileNotFound:
		return 'f'
	case TeForbidden:
		return 'a'
	case TeBadSize:
		return 'd'
	case TeShuttingDown:
		return 'S'
	default:
		return '.'
	}
}

// FromR66Code returns the TransferError equivalent to the given R66 error code.
//
//nolint:funlen,cyclop // cannot be shorten without adding complexity
func FromR66Code(c rune) TransferErrorCode {
	switch c {
	case 'O':
		return TeOk
	case 'I':
		return TeInternal
	case 'U':
		return TeUnimplemented
	case 'C':
		return TeConnection
	case 'D':
		return TeConnectionReset
	case 'N':
		return TeUnknownRemote
	case 'l':
		return TeExceededLimit
	case 'A':
		return TeBadAuthentication
	case 'T':
		return TeDataTransfer
	case 'M':
		return TeIntegrity
	case 'F':
		return TeFinalization
	case 'E':
		return TeExternalOperation
	case 'w':
		return TeWarning
	case 'H':
		return TeStopped
	case 'K':
		return TeCanceled
	case 'f':
		return TeFileNotFound
	case 'a':
		return TeForbidden
	case 'd':
		return TeBadSize
	case 'S':
		return TeShuttingDown
	default:
		return TeUnknown
	}
}

// TransferError represents any error that occurs during the transfer.
// It contains an error code and a message giving more info about the error.
//
// It is safe to use with errors.Is() and errors.As().
type TransferError struct {
	Code    TransferErrorCode `xorm:"error_code"`
	Details string            `xorm:"error_details"`
}

// NewTransferError creates a new transfer error with the given error code and
// details. It panics if an unknown error code is given, as it is probably a
// bug which should fixed be during development.
func NewTransferError(code TransferErrorCode, details string, args ...interface{},
) *TransferError {
	if !code.IsValid() {
		panic(fmt.Sprintf("%v is an invalid error code", code))
	}

	if code == TeOk {
		details = ""
	}

	return &TransferError{Code: code, Details: fmt.Sprintf(details, args...)}
}

func (te *TransferError) Error() string {
	rv := fmt.Sprintf("TransferError(%v)", te.Code)
	if te.Details != "" {
		rv += ": " + te.Details
	}

	return rv
}

// UnmarshalJSON implements json.Unmarshaler. It is a noop.
func (te TransferError) UnmarshalJSON([]byte) error {
	return nil
}
