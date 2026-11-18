package model

import (
	"iter"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/ordered"
)

// The reserved keys of the transfer information (#816), of the form
// __key__ with a lower-case first letter: the gateway reads some from the
// user (set in the command that creates the transfer, in a TRANSFER or
// PREREGISTER task, by SETINFO), writes others during the transfer, a few
// both ways. Every package names its keys from here, so that the screens
// that offer them (the variables list, the name of an information) and the
// documentation follow one source: a test checks that no other package
// declares a reserved key of its own.

// InfoKeyFamily groups the reserved keys by what they concern.
type InfoKeyFamily string

const (
	// InfoFamilyTransfer: the follow ID, the user content, the synchronised
	// transfers.
	InfoFamilyTransfer InfoKeyFamily = "transfer"
	// InfoFamilyAck: the acknowledgements (SENDMESSAGE, PeSIT).
	InfoFamilyAck InfoKeyFamily = "ack"
)

// InfoKeySetter says who sets a reserved key.
type InfoKeySetter uint8

const (
	// InfoSetByUser: the user sets it, the gateway reads it.
	InfoSetByUser InfoKeySetter = iota + 1
	// InfoSetByGateway: the gateway writes it during the transfer.
	InfoSetByGateway
	// InfoSetByBoth: the user sets it on one side of the transfer, the
	// gateway writes it on the other.
	InfoSetByBoth
)

// The reserved keys. FollowID, SyncTransferID and SyncTransferRank keep the
// names the importers of the module know.
const (
	// FollowID: the R66 follow ID, which also links a transfer and its
	// acknowledgement.
	FollowID = "__followID__"
	// SyncTransferID and SyncTransferRank: the parent and the rank of a
	// synchronised child transfer.
	SyncTransferID   = "__syncTransferID__"
	SyncTransferRank = "__syncTransferRank__"

	// The acknowledgements: expected, how long to wait for one and since
	// when, received and when; sent, when, to which partner, as which
	// account.

	InfoAckExpected    = "__ackExpected__"
	InfoAckWaitTimeout = "__ackWaitTimeout__"
	InfoAckWaitSince   = "__ackWaitSince__"
	InfoAckReceived    = "__ackReceived__"
	InfoAckReceivedOn  = "__ackReceivedOn__"
	InfoAckSent        = "__ackSent__"
	InfoAckSentOn      = "__ackSentOn__"
	InfoAckSentTo      = "__ackSentTo__"
	InfoAckSentAs      = "__ackSentAs__"
)

// ReservedInfoKey describes a reserved key of the transfer information.
type ReservedInfoKey struct {
	Name  string
	SetBy InfoKeySetter
}

type ReservedInfoList ordered.Map[InfoKeyFamily, []ReservedInfoKey]

func (r *ReservedInfoList) Len() int {
	return utils.Count2(r.Iter())
}

func (r *ReservedInfoList) Add(family InfoKeyFamily, keys []ReservedInfoKey) {
	(*ordered.Map[InfoKeyFamily, []ReservedInfoKey])(r).Add(family, keys)
}

func (r *ReservedInfoList) Iter() iter.Seq2[InfoKeyFamily, ReservedInfoKey] {
	return func(yield func(InfoKeyFamily, ReservedInfoKey) bool) {
		for _, elem := range *r {
			for _, key := range elem.Val {
				if !yield(elem.Key, key) {
					return
				}
			}
		}
	}
}

// UserSets tells whether the user may set the key (the gateway reads it on
// at least one side of the transfer).
func (k ReservedInfoKey) UserSets() bool {
	return k.SetBy != InfoSetByGateway
}

// ReservedInfoKeys lists the reserved keys by family, in the order the
// screens show them.
//
//nolint:gochecknoglobals //global var is needed here
var ReservedInfoKeys = DefaultInfoList()

func DefaultInfoList() *ReservedInfoList {
	l := &ReservedInfoList{}
	l.Add(InfoFamilyTransfer, []ReservedInfoKey{
		{FollowID, InfoSetByBoth},
		{SyncTransferID, InfoSetByGateway},
		{SyncTransferRank, InfoSetByGateway},
	})
	l.Add(InfoFamilyAck, []ReservedInfoKey{
		{InfoAckExpected, InfoSetByBoth},
		{InfoAckWaitTimeout, InfoSetByBoth},
		{InfoAckWaitSince, InfoSetByGateway},
		{InfoAckReceived, InfoSetByGateway},
		{InfoAckReceivedOn, InfoSetByGateway},
		{InfoAckSent, InfoSetByGateway},
		{InfoAckSentOn, InfoSetByGateway},
		{InfoAckSentTo, InfoSetByGateway},
		{InfoAckSentAs, InfoSetByGateway},
	})

	return l
}
