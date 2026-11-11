package model

import (
	"errors"
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// The acknowledgement of a transfer is read on the transfer itself, in its
// information (#873): a partner option, a SETINFO pre-task or the information
// given at the creation of the transfer set InfoAckExpected; the pipeline, the
// PeSIT module and the SENDMESSAGE task write the other keys. The cases below
// are the ones the lists filter on, the same in the web interface, the REST
// API and the command line.

// AckCase is a case of the acknowledgement of a transfer.
type AckCase string

const (
	// AckExpected: the transfer waits for an acknowledgement it has not
	// received, and is not over (the wait not started yet, or under way).
	AckExpected AckCase = "expected"
	// AckReceived: the transfer received its acknowledgement.
	AckReceived AckCase = "received"
	// AckMissed: the transfer is over without the acknowledgement it waited for.
	AckMissed AckCase = "missed"
	// AckSent: the gateway acknowledged the transfer to the partner.
	AckSent AckCase = "sent"
	// AckNone: no acknowledgement is expected, received or sent.
	AckNone AckCase = "none"
)

// ErrUnknownAckCase says that a name is not one of the acknowledgement cases.
var ErrUnknownAckCase = errors.New("unknown acknowledgement case")

// AckCases lists the cases, in the order the lists offer them.
func AckCases() []AckCase {
	return []AckCase{AckExpected, AckReceived, AckMissed, AckSent, AckNone}
}

// AckCaseNames writes the names of the cases, separated by commas: the values
// a parameter accepts.
func AckCaseNames() string {
	names := make([]string, 0, len(AckCases()))
	for _, c := range AckCases() {
		names = append(names, string(c))
	}

	return strings.Join(names, ", ")
}

// ParseAckCase reads a case from its name.
func ParseAckCase(name string) (AckCase, error) {
	for _, c := range AckCases() {
		if string(c) == name {
			return c, nil
		}
	}

	return "", fmt.Errorf("%w: %q (accepted: %s)", ErrUnknownAckCase, name, AckCaseNames())
}

// The JSON text of a true value in the transfer information: the keys of the
// acknowledgement are booleans, and a key explicitly set to false counts as
// not set.
const ackTrue = "true"

// A transfer owns its information rows by transfer_id while it runs and by
// history_id once over, under one id from one table to the other: the
// normalized view of the information gives that owner. The conditions below
// apply to the transfers table, to the history and to the normalized view of
// the transfers alike, which all name the transfer "id" and its "status".
const (
	ackHas   = "id IN (SELECT owner_id FROM " + ViewNormalizedTransferInfo + " WHERE name = ? AND value = ?)"
	ackLacks = "id NOT IN (SELECT owner_id FROM " + ViewNormalizedTransferInfo + " WHERE name = ? AND value = ?)"
	ackOver  = "status IN (?, ?, ?)"
	ackAlive = "status NOT IN (?, ?, ?)"
)

// Condition gives the SQL condition that keeps the transfers of the case,
// with its arguments, for the transfers table, the history or the normalized
// view of the transfers.
func (c AckCase) Condition() (string, []any) {
	over := []any{types.StatusDone, types.StatusError, types.StatusCancelled}

	switch c {
	case AckExpected:
		return ackHas + " AND " + ackLacks + " AND " + ackAlive,
			append([]any{InfoAckExpected, ackTrue, InfoAckReceived, ackTrue}, over...)
	case AckReceived:
		return ackHas, []any{InfoAckReceived, ackTrue}
	case AckMissed:
		return ackHas + " AND " + ackLacks + " AND " + ackOver,
			append([]any{InfoAckExpected, ackTrue, InfoAckReceived, ackTrue}, over...)
	case AckSent:
		return ackHas, []any{InfoAckSent, ackTrue}
	case AckNone:
		return ackLacks + " AND " + ackLacks + " AND " + ackLacks,
			[]any{InfoAckExpected, ackTrue, InfoAckReceived, ackTrue, InfoAckSent, ackTrue}
	default:
		// An unknown case keeps nothing: the callers parse the case first.
		return "1 = 0", nil
	}
}
