package stores

import "code.waarp.fr/apps/gateway/gateway/pkg/model"

type OperationStore interface {
	InsertOperation(op *model.EbicsOperation) error
	UpdateOperation(op *model.EbicsOperation) error
	GetOperationByID(id int64) (*model.EbicsOperation, error)
	GetOperationByCorrelationID(owner, correlationID string) (*model.EbicsOperation, error)
}
