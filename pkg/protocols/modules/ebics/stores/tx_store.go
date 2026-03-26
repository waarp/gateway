package stores

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

// TransactionStore defines the persistence contract for EBICS transactions.
type TransactionStore interface {
	InsertTransaction(tx *model.EbicsTransaction) error
	UpdateTransaction(tx *model.EbicsTransaction) error
	GetTransactionByTxID(owner, txID string) (*model.EbicsTransaction, error)
}

// SegmentStore defines the persistence contract for EBICS transaction segments.
type SegmentStore interface {
	InsertSegment(seg *model.EbicsTransactionSegment) error
	UpdateSegment(seg *model.EbicsTransactionSegment) error
	ListSegments(transactionID int64) ([]model.EbicsTransactionSegment, error)
}
