package model

import (
	"errors"
	"fmt"
	"iter"
	"strings"
	"time"

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

// AckDefaultWait is how long the pipeline waits for an acknowledgement when
// the transfer gives no limit (InfoAckWaitTimeout).
const AckDefaultWait = 10 * time.Minute

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

// ackOverStatuses are the statuses of a transfer that is over: an
// acknowledgement still expected then is missed.
func ackOverStatuses() []types.TransferStatus {
	return []types.TransferStatus{types.StatusDone, types.StatusError, types.StatusCancelled}
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
		query.Where("status NOT IN ?", ackOverStatuses())
	case AckReceived:
		hasInfo(InfoAckReceived, jsTrue)
	case AckMissed:
		hasInfo(InfoAckExpected, jsTrue)
		lacksInfo(InfoAckReceived, jsTrue)
		query.Where("status IN ?", ackOverStatuses())
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

// infoTrue tells whether a key of the transfer information is set to true,
// as the gateway writes the acknowledgement keys; a key set to false counts
// as absent, as in Conds.
func infoTrue(info map[string]any, key string) bool {
	value, ok := info[key].(bool)

	return ok && value
}

// AckOf is the case of a transfer already read, from its status and its
// information: the rules of Conds, in Go, for the lists that read the
// keys of a page in one query. Expected, received and missed exclude
// one another; a transfer the gateway acknowledged without one of them is
// "sent" (AckWasSent tells it whatever the case, a relay carrying both).
func AckOf(status types.TransferStatus, info map[string]any) AckCase {
	switch {
	case infoTrue(info, InfoAckReceived):
		return AckReceived
	case !infoTrue(info, InfoAckExpected):
		if AckWasSent(info) {
			return AckSent
		}

		return AckNone
	case status.IsOneOf(ackOverStatuses()...):
		return AckMissed
	default:
		return AckExpected
	}
}

// AckWasSent tells whether the gateway acknowledged the transfer to the
// partner (SENDMESSAGE), whatever its other case.
func AckWasSent(info map[string]any) bool {
	return infoTrue(info, InfoAckSent)
}
