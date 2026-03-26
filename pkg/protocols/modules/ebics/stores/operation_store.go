package stores

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

// OperationStore defines the persistence contract for EBICS operations.
type OperationStore interface {
	InsertOperation(op *model.EbicsOperation) error
	UpdateOperation(op *model.EbicsOperation) error
	GetOperationByID(id int64) (*model.EbicsOperation, error)
	GetOperationByCorrelationID(owner, correlationID string) (*model.EbicsOperation, error)
}
