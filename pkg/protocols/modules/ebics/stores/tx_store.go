package stores

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

type TransactionStore interface {
	InsertTransaction(tx *model.EbicsTransaction) error
	UpdateTransaction(tx *model.EbicsTransaction) error
	GetTransactionByTxID(owner, txID string) (*model.EbicsTransaction, error)
}

type SegmentStore interface {
	InsertSegment(seg *model.EbicsTransactionSegment) error
	UpdateSegment(seg *model.EbicsTransactionSegment) error
	ListSegments(transactionID int64) ([]model.EbicsTransactionSegment, error)
}
