package model

import (
	"errors"
	"fmt"
	"iter"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// The acknowledgement of a transfer is read on the transfer itself, in its
// information (#873): a partner option, a SETINFO pre-task or the information
// given at the creation of the transfer set InfoAckExpected; the pipeline, the
// PeSIT module and the SENDMESSAGE task write the other keys. The cases below
// are the ones the lists filter on, the same in the web interface, the REST
// API and the command line.

// AckCase is a case of the acknowledgement of a transfer.
type AckCase int8

//go:generate stringer -type AckCase -linecomment -output transfer_ack_string.go
const (
	// AckNone: no acknowledgement is expected, received or sent.
	AckNone AckCase = iota // none
	// AckExpected: the transfer waits for an acknowledgement it has not
	// received, and is not over (the wait not started yet, or under way).
	AckExpected // expected
	// AckReceived: the transfer received its acknowledgement.
	AckReceived // received
	// AckMissed: the transfer is over without the acknowledgement it waited for.
	AckMissed // missed
	// AckSent: the gateway acknowledged the transfer to the partner.
	AckSent // sent

	endAckCases
)

// ErrUnknownAckCase says that a name is not one of the acknowledgement cases.
var ErrUnknownAckCase = errors.New("unknown acknowledgement case")

// AckCases lists the cases, in the order the lists offer them.
func AckCases() iter.Seq[AckCase] {
	return func(yield func(AckCase) bool) {
		for i := range endAckCases {
			if !yield(i) {
				return
			}
		}
	}
}

// AckCaseNames retrieves all the names of the acknowledgement cases.
func AckCaseNames() []string {
	names := make([]string, 0, endAckCases)
	for c := range AckCases() {
		names = append(names, c.String())
	}

	return names
}

// ParseAckCase reads a case from its name.
func ParseAckCase(name string) (AckCase, error) {
	for c := range AckCases() {
		if strings.EqualFold(name, c.String()) {
			return c, nil
		}
	}

	return 0, fmt.Errorf("%w: %q (accepted: %s)", ErrUnknownAckCase, name,
		strings.Join(AckCaseNames(), ", "))
}

// Conds applies the SQL condition to the given query that keeps the
// transfers of the case, with its arguments, for the transfers table, the history
// or the normalized view of the transfers.
func (c AckCase) Conds(query *database.SelectQuery) {
	const jsTrue = "true"

	hasInfo := func(key, val string) {
		query.Where(`id IN (SELECT owner_id FROM `+ViewNormalizedTransferInfo+
			` WHERE name=? AND value=?)`, key, val)
	}
	lacksInfo := func(key, val string) {
		query.Where(`id NOT IN (SELECT owner_id FROM `+ViewNormalizedTransferInfo+
			` WHERE name=? AND value=?)`, key, val)
	}

	switch c {
	case AckExpected:
		hasInfo(InfoAckExpected, jsTrue)
		lacksInfo(InfoAckReceived, jsTrue)
		query.In("status NOT", types.StatusDone, types.StatusError, types.StatusCancelled)
	case AckReceived:
		hasInfo(InfoAckReceived, jsTrue)
	case AckMissed:
		hasInfo(InfoAckExpected, jsTrue)
		lacksInfo(InfoAckReceived, jsTrue)
		query.In("status", types.StatusDone, types.StatusError, types.StatusCancelled)
	case AckSent:
		hasInfo(InfoAckSent, jsTrue)
	case AckNone:
		lacksInfo(InfoAckExpected, jsTrue)
		lacksInfo(InfoAckReceived, jsTrue)
		lacksInfo(InfoAckSent, jsTrue)
	default:
		query.Where("1 = 0")
	}
}
