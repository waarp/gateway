package ebics

import (
	"context"
	"fmt"
	"time"

	libebics "code.waarp.fr/lib/ebics/ebics"
	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	ebicsClientRecoveryStatusRunning    = "RUNNING"
	ebicsClientRecoveryStatusRecovering = "RECOVERING"
)

type clientRecoveryStore struct {
	db           *database.DB
	operationID  int64
	transferID   int64
	hostID       int64
	subscriberID int64
	orderType    string
	direction    string
	totalSize    int64
}

func newClientRecoveryStore(
	db *database.DB,
	operation *model.EbicsOperation,
	transfer *model.Transfer,
) *clientRecoveryStore {
	store := &clientRecoveryStore{db: db}
	if operation != nil {
		store.operationID = operation.ID
		store.hostID = operation.EbicsHostID
		store.subscriberID = operation.EbicsSubscriberID
		store.orderType = operation.OrderType
		store.direction = operation.Direction
	}
	if transfer != nil {
		store.transferID = transfer.ID
		store.totalSize = transfer.Filesize
	}

	return store
}

func (s *clientRecoveryStore) PutSegment(
	_ context.Context,
	txID libebics.TransactionID,
	number int,
	hash []byte,
) error {
	row, err := s.getOrCreateTransaction(string(txID))
	if err != nil {
		return err
	}

	var segment model.EbicsTransactionSegment
	getErr := s.db.Get(
		&segment,
		"owner=? AND ebics_transaction_id=? AND segment_number=?",
		model.EbicsGatewayOwnerForRuntime(),
		row.ID,
		number,
	).Run()
	if getErr != nil && !database.IsNotFound(getErr) {
		return fmt.Errorf("load EBICS client recovery segment %d: %w", number, getErr)
	}

	if database.IsNotFound(getErr) {
		segment = model.EbicsTransactionSegment{
			EbicsTransactionID: row.ID,
			SegmentNumber:      number,
		}
	}

	segment.SegmentStatus = model.EbicsTransactionSegmentStatusStoredForRuntime()
	segment.Checksum = fmt.Sprintf("%x", hash)
	segment.MetadataMap = map[string]any{}

	if database.IsNotFound(getErr) {
		if err = s.db.Insert(&segment).Run(); err != nil {
			return fmt.Errorf("insert EBICS client recovery segment %d: %w", number, err)
		}
	} else if err = s.db.Update(&segment).Run(); err != nil {
		return fmt.Errorf("update EBICS client recovery segment %d: %w", number, err)
	}

	row.CurrentSegment = max(row.CurrentSegment, number)
	row.Status = ebicsClientRecoveryStatusRunning
	row.UpdatedAt = time.Now().UTC()

	if err = s.db.Update(row).Run(); err != nil {
		return fmt.Errorf("update EBICS client recovery transaction after segment %d: %w", number, err)
	}

	return nil
}

func (s *clientRecoveryStore) HasSegment(
	_ context.Context,
	txID libebics.TransactionID,
	number int,
) (bool, error) {
	row, err := s.getExistingTransaction(string(txID))
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}

	count, err := s.db.Count(&model.EbicsTransactionSegment{}).
		Where(
			"owner=? AND ebics_transaction_id=? AND segment_number=?",
			model.EbicsGatewayOwnerForRuntime(),
			row.ID,
			number,
		).
		Run()
	if err != nil {
		return false, fmt.Errorf("count EBICS client recovery segments: %w", err)
	}

	return count != 0, nil
}

func (s *clientRecoveryStore) UpdateRecovery(
	_ context.Context,
	txID libebics.TransactionID,
	point, counter int,
) error {
	row, err := s.getOrCreateTransaction(string(txID))
	if err != nil {
		return err
	}

	meta := row.MetadataMap
	if meta == nil {
		meta = map[string]any{}
	}
	meta["recoveryPoint"] = point
	meta["recoveryCounter"] = counter
	row.MetadataMap = meta
	row.Status = ebicsClientRecoveryStatusRecovering
	row.UpdatedAt = time.Now().UTC()

	if err = s.db.Update(row).Run(); err != nil {
		return fmt.Errorf("update EBICS client recovery state for transaction %q: %w", txID, err)
	}

	return nil
}

func (s *clientRecoveryStore) GetRecovery(
	_ context.Context,
	txID libebics.TransactionID,
) (point, counter int, err error) {
	row, err := s.getExistingTransaction(string(txID))
	if err != nil {
		return 0, 0, err
	}
	if row == nil {
		return 0, 0, nil
	}

	return readIntMetadata(row.MetadataMap, "recoveryPoint"),
		readIntMetadata(row.MetadataMap, "recoveryCounter"),
		nil
}

func (s *clientRecoveryStore) GetAttempts(_ context.Context, txID libebics.TransactionID) (int, error) {
	row, err := s.getExistingTransaction(string(txID))
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, nil
	}

	return readIntMetadata(row.MetadataMap, "recoveryAttempts"), nil
}

func (s *clientRecoveryStore) IncrementAttempts(_ context.Context, txID libebics.TransactionID) error {
	row, err := s.getOrCreateTransaction(string(txID))
	if err != nil {
		return err
	}

	meta := row.MetadataMap
	if meta == nil {
		meta = map[string]any{}
	}
	meta["recoveryAttempts"] = readIntMetadata(meta, "recoveryAttempts") + 1
	row.MetadataMap = meta
	row.UpdatedAt = time.Now().UTC()

	if err = s.db.Update(row).Run(); err != nil {
		return fmt.Errorf("increment EBICS client recovery attempts for transaction %q: %w", txID, err)
	}

	return nil
}

func (s *clientRecoveryStore) ResetAttempts(_ context.Context, txID libebics.TransactionID) error {
	row, err := s.getExistingTransaction(string(txID))
	if err != nil || row == nil {
		return err
	}

	meta := row.MetadataMap
	if meta == nil {
		meta = map[string]any{}
	}
	meta["recoveryAttempts"] = 0
	row.MetadataMap = meta
	row.UpdatedAt = time.Now().UTC()

	if err = s.db.Update(row).Run(); err != nil {
		return fmt.Errorf("reset EBICS client recovery attempts for transaction %q: %w", txID, err)
	}

	return nil
}

func (s *clientRecoveryStore) DeleteTransaction(_ context.Context, txID libebics.TransactionID) error {
	row, err := s.getExistingTransaction(string(txID))
	if err != nil || row == nil {
		return err
	}

	row.Status = model.EbicsTransactionStatusCompletedForRuntime()
	row.UpdatedAt = time.Now().UTC()

	if err = s.db.Update(row).Run(); err != nil {
		return fmt.Errorf("complete EBICS client recovery transaction %q: %w", txID, err)
	}

	return nil
}

func (s *clientRecoveryStore) getExistingTransaction(txID string) (*model.EbicsTransaction, error) {
	var row model.EbicsTransaction
	err := s.db.Get(
		&row,
		"owner=? AND transaction_id=?",
		model.EbicsGatewayOwnerForRuntime(),
		txID,
	).Run()
	if err != nil {
		if database.IsNotFound(err) {
			return nil, nil //nolint:nilnil // nil row means the client transaction is not yet materialized
		}

		return nil, fmt.Errorf("load EBICS client recovery transaction %q: %w", txID, err)
	}

	return &row, nil
}

func (s *clientRecoveryStore) getOrCreateTransaction(txID string) (*model.EbicsTransaction, error) {
	row, err := s.getExistingTransaction(txID)
	if err != nil || row != nil {
		return row, err
	}

	row = &model.EbicsTransaction{
		EbicsHostID:       s.hostID,
		EbicsSubscriberID: s.subscriberID,
		EbicsOperationID:  nullableID(s.operationID),
		TransactionID:     txID,
		OrderType:         s.orderType,
		TransferID:        nullableID(s.transferID),
		Status:            ebicsClientRecoveryStatusRunning,
		Direction:         s.direction,
		TotalSize:         s.totalSize,
		MetadataMap:       map[string]any{},
	}

	if err = s.db.Insert(row).Run(); err != nil {
		return nil, fmt.Errorf("insert EBICS client recovery transaction %q: %w", txID, err)
	}

	return row, nil
}

var (
	_ libebicsclient.RecoveryStore        = (*clientRecoveryStore)(nil)
	_ libebicsclient.RecoveryAttemptStore = (*clientRecoveryStore)(nil)
	_ libebicsclient.RecoveryCleanupStore = (*clientRecoveryStore)(nil)
)
