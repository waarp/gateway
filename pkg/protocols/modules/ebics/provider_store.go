package ebics

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	libebics "code.waarp.fr/lib/ebics/ebics"
	libstore "code.waarp.fr/lib/ebics/ebics/store"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const defaultNonceTTL = 15 * time.Minute

type providerStore struct {
	db *database.DB
}

func newProviderStore(db *database.DB) *providerStore {
	return &providerStore{db: db}
}

func (s *providerStore) GetBankKeyDigests(
	ctx context.Context,
	hostID libebics.HostID,
) (libebics.BankKeyDigests, error) {
	keys, err := s.GetBankPublicKeys(ctx, hostID)
	if err != nil {
		return libebics.BankKeyDigests{}, err
	}

	var digests libebics.BankKeyDigests
	if len(keys.AuthPublicKey) > 0 {
		sum := sha256.Sum256(keys.AuthPublicKey)
		digests.AuthKeyDigest = sum[:]
	}
	if len(keys.EncPublicKey) > 0 {
		sum := sha256.Sum256(keys.EncPublicKey)
		digests.EncKeyDigest = sum[:]
	}
	if len(keys.SigPublicKey) > 0 {
		sum := sha256.Sum256(keys.SigPublicKey)
		digests.SigKeyDigest = sum[:]
	}

	return digests, nil
}

func (s *providerStore) GetBankPublicKeys(
	_ context.Context,
	hostID libebics.HostID,
) (libebics.BankPublicKeys, error) {
	host, err := s.getHostByHostID(string(hostID))
	if err != nil {
		return libebics.BankPublicKeys{}, err
	}

	var keys model.EbicsBankKeys
	if selectErr := s.db.Select(&keys).
		Where("owner=? AND ebics_host_id=? AND state=?", host.Owner, host.ID, "validated").
		Run(); selectErr != nil {
		return libebics.BankPublicKeys{}, fmt.Errorf("load EBICS bank public keys: %w", selectErr)
	}

	var out libebics.BankPublicKeys
	for _, key := range keys {
		switch strings.ToUpper(key.KeyType) {
		case "AUTH":
			out.AuthPublicKey = []byte(key.PublicKey)
		case "ENCRYPT":
			out.EncPublicKey = []byte(key.PublicKey)
		case "SIGNATURE":
			out.SigPublicKey = []byte(key.PublicKey)
		}
	}

	return out, nil
}

func (s *providerStore) GetSubscriberKeys(
	_ context.Context,
	hostID libebics.HostID,
	partnerID libebics.PartnerID,
	userID libebics.UserID,
) (libebics.SubscriberKeys, error) {
	subscriber, err := s.getSubscriber(string(hostID), string(partnerID), string(userID))
	if err != nil {
		return libebics.SubscriberKeys{}, err
	}

	var materials model.EbicsSubscriberKeyMaterials
	if selectErr := s.db.Select(&materials).
		Where("owner=? AND ebics_subscriber_id=? AND state=?", subscriber.Owner, subscriber.ID, "ACTIVE").
		Run(); selectErr != nil {
		return libebics.SubscriberKeys{}, fmt.Errorf("load EBICS subscriber key materials: %w", selectErr)
	}

	var keys libebics.SubscriberKeys
	for _, material := range materials {
		switch strings.ToUpper(material.KeyUsage) {
		case model.EbicsKeyUsageAuthenticationForRuntime():
			keys.AuthPublicKey = []byte(material.PublicKey)
			keys.AuthCertificate = []byte(material.Certificate)
			keys.AuthCertificateVersion = material.CertificateVersion
		case model.EbicsKeyUsageEncryptionForRuntime():
			keys.EncPublicKey = []byte(material.PublicKey)
			keys.EncCertificate = []byte(material.Certificate)
			keys.EncCertificateVersion = material.CertificateVersion
		case model.EbicsKeyUsageSignatureForRuntime():
			keys.SigPublicKey = []byte(material.PublicKey)
			keys.SigCertificate = []byte(material.Certificate)
			keys.SigCertificateVersion = material.CertificateVersion
		}
	}

	return keys, nil
}

//nolint:gocritic // lib-ebics store interface imposes this value signature.
func (s *providerStore) PutSubscriberKeys(
	_ context.Context,
	hostID libebics.HostID,
	partnerID libebics.PartnerID,
	userID libebics.UserID,
	keys libebics.SubscriberKeys,
) error {
	subscriber, err := s.getSubscriber(string(hostID), string(partnerID), string(userID))
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *database.Session) error {
		authPayload := keyMaterialPayload{
			PublicKey:          string(keys.AuthPublicKey),
			Certificate:        string(keys.AuthCertificate),
			CertificateVersion: keys.AuthCertificateVersion,
		}
		if upsertErr := s.upsertSubscriberKeyMaterial(
			tx,
			subscriber,
			model.EbicsKeyUsageAuthenticationForRuntime(),
			authPayload,
		); upsertErr != nil {
			return upsertErr
		}

		encPayload := keyMaterialPayload{
			PublicKey:          string(keys.EncPublicKey),
			Certificate:        string(keys.EncCertificate),
			CertificateVersion: keys.EncCertificateVersion,
		}
		if upsertErr := s.upsertSubscriberKeyMaterial(
			tx,
			subscriber,
			model.EbicsKeyUsageEncryptionForRuntime(),
			encPayload,
		); upsertErr != nil {
			return upsertErr
		}

		sigPayload := keyMaterialPayload{
			PublicKey:          string(keys.SigPublicKey),
			Certificate:        string(keys.SigCertificate),
			CertificateVersion: keys.SigCertificateVersion,
		}

		return s.upsertSubscriberKeyMaterial(
			tx,
			subscriber,
			model.EbicsKeyUsageSignatureForRuntime(),
			sigPayload,
		)
	})
}

func (s *providerStore) GetSubscriber(
	_ context.Context,
	hostID libebics.HostID,
	partnerID libebics.PartnerID,
	userID libebics.UserID,
) (libebics.Subscriber, error) {
	subscriber, err := s.getSubscriber(string(hostID), string(partnerID), string(userID))
	if err != nil {
		return libebics.Subscriber{}, err
	}

	return libebics.Subscriber{
		HostID:    hostID,
		PartnerID: libebics.PartnerID(subscriber.PartnerID),
		UserID:    libebics.UserID(subscriber.UserID),
	}, nil
}

func (s *providerStore) PutSubscriber(_ context.Context, sub libebics.Subscriber) error {
	host, err := s.getHostByHostID(string(sub.HostID))
	if err != nil {
		return err
	}

	var subscriber model.EbicsSubscriber
	getErr := s.db.Get(&subscriber,
		"owner=? AND ebics_host_id=? AND partner_id=? AND user_id=?",
		host.Owner,
		host.ID,
		string(sub.PartnerID),
		string(sub.UserID),
	).Run()
	if getErr != nil && !database.IsNotFound(getErr) {
		return fmt.Errorf("load EBICS subscriber before upsert: %w", getErr)
	}

	if database.IsNotFound(getErr) {
		subscriber = model.EbicsSubscriber{
			EbicsHostID: host.ID,
			Name:        strings.TrimSpace(string(sub.PartnerID) + ":" + string(sub.UserID)),
			PartnerID:   string(sub.PartnerID),
			UserID:      string(sub.UserID),
			Enabled:     true,
		}

		if insertErr := s.db.Insert(&subscriber).Run(); insertErr != nil {
			return fmt.Errorf("insert EBICS subscriber from provider store: %w", insertErr)
		}

		return nil
	}

	subscriber.Enabled = true
	if updateErr := s.db.Update(&subscriber).Run(); updateErr != nil {
		return fmt.Errorf("update EBICS subscriber from provider store: %w", updateErr)
	}

	return nil
}

//nolint:gocritic // lib-ebics store interface imposes this value signature.
func (s *providerStore) CreateTransaction(_ context.Context, tx libebics.Transaction) error {
	subscriber, err := s.getSubscriber(string(tx.HostID), string(tx.PartnerID), string(tx.UserID))
	if err != nil {
		return err
	}

	row := model.EbicsTransaction{
		EbicsHostID:       subscriber.EbicsHostID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     string(tx.ID),
		OrderType:         string(tx.OrderType),
		Status:            normalizeLibTransactionStatus(tx.Status),
		Direction:         model.EbicsOperationDirectionInboundForRuntime(),
		SegmentCount:      tx.SegmentCnt,
		CreatedAt:         valueOrNowUTC(tx.CreatedAt),
		UpdatedAt:         valueOrNowUTC(tx.UpdatedAt),
	}

	if insertErr := s.db.Insert(&row).Run(); insertErr != nil {
		return fmt.Errorf("insert EBICS transaction from provider store: %w", insertErr)
	}

	return nil
}

func (s *providerStore) GetTransaction(_ context.Context, id libebics.TransactionID) (libebics.Transaction, error) {
	var row model.EbicsTransaction
	if err := s.db.Get(&row, "owner=? AND transaction_id=?", s.gatewayOwner(), string(id)).Run(); err != nil {
		if database.IsNotFound(err) {
			return libebics.Transaction{}, nil
		}

		return libebics.Transaction{}, fmt.Errorf("load EBICS transaction: %w", err)
	}

	subscriber, err := s.getSubscriberByID(row.EbicsSubscriberID)
	if err != nil {
		return libebics.Transaction{}, err
	}

	host, err := s.getHostByID(row.EbicsHostID)
	if err != nil {
		return libebics.Transaction{}, err
	}

	return libebics.Transaction{
		ID:         libebics.TransactionID(row.TransactionID),
		HostID:     libebics.HostID(host.HostID),
		PartnerID:  libebics.PartnerID(subscriber.PartnerID),
		UserID:     libebics.UserID(subscriber.UserID),
		OrderType:  libebics.OrderType(row.OrderType),
		SegmentCnt: row.SegmentCount,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

//nolint:gocritic // lib-ebics store interface imposes this value signature.
func (s *providerStore) UpdateTransaction(ctx context.Context, tx libebics.Transaction) error {
	var row model.EbicsTransaction
	if err := s.db.Get(&row, "owner=? AND transaction_id=?", s.gatewayOwner(), string(tx.ID)).Run(); err != nil {
		if database.IsNotFound(err) {
			return s.CreateTransaction(ctx, tx)
		}

		return fmt.Errorf("load EBICS transaction for update: %w", err)
	}

	row.Status = normalizeLibTransactionStatus(tx.Status)
	if tx.SegmentCnt > 0 {
		row.SegmentCount = max(row.SegmentCount, tx.SegmentCnt)
	}
	row.UpdatedAt = valueOrNowUTC(tx.UpdatedAt)

	if updateErr := s.db.Update(&row).Run(); updateErr != nil {
		return fmt.Errorf("update EBICS transaction from provider store: %w", updateErr)
	}

	return nil
}

func (s *providerStore) PurgeTransactionsBefore(_ context.Context, t time.Time) error {
	if err := s.db.DeleteAll(&model.EbicsTransaction{}).
		Where("owner=? AND updated_at<?", s.gatewayOwner(), t).
		Run(); err != nil {
		return fmt.Errorf("purge EBICS transactions: %w", err)
	}

	return nil
}

func (s *providerStore) AddSegment(
	_ context.Context,
	txID libebics.TransactionID,
	seg libebics.SegmentInfo,
	payloadHash []byte,
) error {
	var txRow model.EbicsTransaction
	if err := s.db.Get(&txRow, "owner=? AND transaction_id=?", s.gatewayOwner(), string(txID)).Run(); err != nil {
		return fmt.Errorf("load EBICS transaction before adding segment: %w", err)
	}

	var row model.EbicsTransactionSegment
	getErr := s.db.Get(&row,
		"owner=? AND ebics_transaction_id=? AND segment_number=?",
		s.gatewayOwner(), txRow.ID, seg.Number,
	).Run()
	if getErr != nil && !database.IsNotFound(getErr) {
		return fmt.Errorf("load EBICS segment before upsert: %w", getErr)
	}

	if database.IsNotFound(getErr) {
		row = model.EbicsTransactionSegment{
			EbicsTransactionID: txRow.ID,
			SegmentNumber:      seg.Number,
		}
	}

	row.SegmentStatus = segmentStatusFromSegmentInfo(seg)
	row.Checksum = fmt.Sprintf("%x", payloadHash)
	row.PayloadSize = 0
	row.MetadataMap = map[string]any{
		"last":            seg.Last,
		"total":           seg.Total,
		"recoveryPoint":   seg.RecoveryPoint,
		"recoveryCounter": seg.RecoveryCounter,
	}

	if database.IsNotFound(getErr) {
		if insertErr := s.db.Insert(&row).Run(); insertErr != nil {
			return fmt.Errorf("insert EBICS transaction segment: %w", insertErr)
		}
	} else if updateErr := s.db.Update(&row).Run(); updateErr != nil {
		return fmt.Errorf("update EBICS transaction segment: %w", updateErr)
	}

	txRow.CurrentSegment = max(txRow.CurrentSegment, seg.Number)
	if seg.Total > 0 {
		txRow.SegmentCount = max(txRow.SegmentCount, seg.Total)
	}
	txRow.SegmentCount = max(txRow.SegmentCount, txRow.CurrentSegment)
	txRow.Status = ebicsClientRecoveryStatusRunning
	txRow.UpdatedAt = time.Now().UTC()

	if updateErr := s.db.Update(&txRow).Run(); updateErr != nil {
		return fmt.Errorf("update EBICS transaction after segment upsert: %w", updateErr)
	}

	return nil
}

func (s *providerStore) UpdateRecovery(
	_ context.Context,
	txID libebics.TransactionID,
	point, counter int,
) error {
	var row model.EbicsTransaction
	if err := s.db.Get(&row, "owner=? AND transaction_id=?", s.gatewayOwner(), string(txID)).Run(); err != nil {
		return fmt.Errorf("load EBICS transaction before recovery update: %w", err)
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

	if updateErr := s.db.Update(&row).Run(); updateErr != nil {
		return fmt.Errorf("update EBICS transaction recovery: %w", updateErr)
	}

	return nil
}

func (s *providerStore) HasSegment(
	_ context.Context,
	txID libebics.TransactionID,
	number int,
) (bool, error) {
	var txRow model.EbicsTransaction
	if err := s.db.Get(&txRow, "owner=? AND transaction_id=?", s.gatewayOwner(), string(txID)).Run(); err != nil {
		if database.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("load EBICS transaction before segment existence check: %w", err)
	}

	count, err := s.db.Count(&model.EbicsTransactionSegment{}).
		Where("owner=? AND ebics_transaction_id=? AND segment_number=?", s.gatewayOwner(), txRow.ID, number).
		Run()
	if err != nil {
		return false, fmt.Errorf("count EBICS transaction segments: %w", err)
	}

	return count != 0, nil
}

func (s *providerStore) GetRecovery(
	_ context.Context,
	txID libebics.TransactionID,
) (point, counter int, err error) {
	var row model.EbicsTransaction
	if err = s.db.Get(&row, "owner=? AND transaction_id=?", s.gatewayOwner(), string(txID)).Run(); err != nil {
		if database.IsNotFound(err) {
			return 0, 0, nil
		}

		return 0, 0, fmt.Errorf("load EBICS transaction before recovery read: %w", err)
	}

	point = readIntMetadata(row.MetadataMap, "recoveryPoint")
	counter = readIntMetadata(row.MetadataMap, "recoveryCounter")

	return point, counter, nil
}

func (s *providerStore) SeenNonce(
	_ context.Context,
	hostID libebics.HostID,
	partnerID libebics.PartnerID,
	userID libebics.UserID,
	nonce string,
) (bool, error) {
	subscriber, err := s.getSubscriber(string(hostID), string(partnerID), string(userID))
	if err != nil {
		return false, err
	}

	count, err := s.db.Count(&model.EbicsNonce{}).
		Where(
			"owner=? AND ebics_subscriber_id=? AND nonce=?",
			subscriber.Owner,
			subscriber.ID,
			strings.TrimSpace(nonce),
		).
		Run()
	if err != nil {
		return false, fmt.Errorf("count EBICS nonces: %w", err)
	}

	return count != 0, nil
}

func (s *providerStore) StoreNonce(
	_ context.Context,
	hostID libebics.HostID,
	partnerID libebics.PartnerID,
	userID libebics.UserID,
	nonce string,
	ts time.Time,
) error {
	subscriber, err := s.getSubscriber(string(hostID), string(partnerID), string(userID))
	if err != nil {
		return err
	}

	row := model.EbicsNonce{
		EbicsSubscriberID: subscriber.ID,
		Nonce:             strings.TrimSpace(nonce),
		Timestamp:         ts.UTC(),
		ExpiresAt:         ts.UTC().Add(defaultNonceTTL),
	}

	if insertErr := s.db.Insert(&row).Run(); insertErr != nil {
		return fmt.Errorf("insert EBICS nonce: %w", insertErr)
	}

	return nil
}

func (s *providerStore) PurgeBefore(_ context.Context, t time.Time) error {
	if err := s.db.DeleteAll(&model.EbicsNonce{}).
		Where("owner=? AND expires_at<?", s.gatewayOwner(), t).
		Run(); err != nil {
		return fmt.Errorf("purge EBICS nonces: %w", err)
	}

	return nil
}

func (s *providerStore) getHostByHostID(hostID string) (*model.EbicsHost, error) {
	var host model.EbicsHost
	if err := s.db.Get(&host, "owner=? AND host_id=?", s.gatewayOwner(), strings.TrimSpace(hostID)).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, fmt.Errorf("EBICS host %q not found: %w", hostID, err)
		}

		return nil, fmt.Errorf("load EBICS host %q: %w", hostID, err)
	}

	return &host, nil
}

func (s *providerStore) getHostByID(hostID int64) (*model.EbicsHost, error) {
	var host model.EbicsHost
	if err := s.db.Get(&host, "id=?", hostID).Run(); err != nil {
		return nil, fmt.Errorf("load EBICS host by ID %d: %w", hostID, err)
	}

	return &host, nil
}

func (s *providerStore) getSubscriber(hostID, partnerID, userID string) (*model.EbicsSubscriber, error) {
	host, err := s.getHostByHostID(hostID)
	if err != nil {
		return nil, err
	}

	var subscriber model.EbicsSubscriber
	if getErr := s.db.Get(
		&subscriber,
		"owner=? AND ebics_host_id=? AND partner_id=? AND user_id=?",
		host.Owner,
		host.ID,
		strings.TrimSpace(partnerID),
		strings.TrimSpace(userID),
	).Run(); getErr != nil {
		if database.IsNotFound(getErr) {
			return nil, fmt.Errorf(
				"EBICS subscriber %s/%s not found on host %s: %w",
				partnerID,
				userID,
				hostID,
				getErr,
			)
		}

		return nil, fmt.Errorf(
			"load EBICS subscriber %s/%s on host %s: %w",
			partnerID,
			userID,
			hostID,
			getErr,
		)
	}

	return &subscriber, nil
}

func (s *providerStore) getSubscriberByID(subscriberID int64) (*model.EbicsSubscriber, error) {
	var subscriber model.EbicsSubscriber
	if err := s.db.Get(&subscriber, "id=?", subscriberID).Run(); err != nil {
		return nil, fmt.Errorf("load EBICS subscriber by ID %d: %w", subscriberID, err)
	}

	return &subscriber, nil
}

type keyMaterialPayload struct {
	PublicKey          string
	Certificate        string
	CertificateVersion string
}

func (s *providerStore) upsertSubscriberKeyMaterial(
	tx database.Access,
	subscriber *model.EbicsSubscriber,
	keyUsage string,
	payload keyMaterialPayload,
) error {
	var material model.EbicsSubscriberKeyMaterial
	getErr := tx.Get(&material,
		"owner=? AND ebics_subscriber_id=? AND key_usage=? AND state=?",
		subscriber.Owner, subscriber.ID, keyUsage, "ACTIVE",
	).Run()
	if getErr != nil && !database.IsNotFound(getErr) {
		return fmt.Errorf("load active EBICS subscriber key material: %w", getErr)
	}

	if database.IsNotFound(getErr) {
		material = model.EbicsSubscriberKeyMaterial{
			EbicsSubscriberID: subscriber.ID,
			KeyUsage:          keyUsage,
			State:             "ACTIVE",
		}
	}

	material.PublicKey = payload.PublicKey
	material.Certificate = payload.Certificate
	material.CertificateVersion = payload.CertificateVersion

	if material.PublicKey == "" && material.Certificate == "" {
		if !database.IsNotFound(getErr) {
			material.State = "RETIRED"
			if updateErr := tx.Update(&material).Run(); updateErr != nil {
				return fmt.Errorf("retire EBICS subscriber key material: %w", updateErr)
			}
		}

		return nil
	}

	if database.IsNotFound(getErr) {
		if insertErr := tx.Insert(&material).Run(); insertErr != nil {
			return fmt.Errorf("insert EBICS subscriber key material: %w", insertErr)
		}
	} else if updateErr := tx.Update(&material).Run(); updateErr != nil {
		return fmt.Errorf("update EBICS subscriber key material: %w", updateErr)
	}

	return nil
}

func (s *providerStore) gatewayOwner() string {
	return model.EbicsGatewayOwnerForRuntime()
}

func normalizeLibTransactionStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "RUNNING":
		return "RUNNING"
	case "RECOVERING":
		return "RECOVERING"
	case model.EbicsTransactionStatusCompletedForRuntime():
		return model.EbicsTransactionStatusCompletedForRuntime()
	case ebicsClientTransactionStatusFailed:
		return ebicsClientTransactionStatusFailed
	case "CANCELLED":
		return "CANCELLED"
	default:
		return "PLANNED"
	}
}

func segmentStatusFromSegmentInfo(seg libebics.SegmentInfo) string {
	if seg.Last {
		return model.EbicsTransactionSegmentStatusCompletedForRuntime()
	}

	return model.EbicsTransactionSegmentStatusStoredForRuntime()
}

func readIntMetadata(metadata map[string]any, key string) int {
	if metadata == nil {
		return 0
	}

	raw, ok := metadata[key]
	if !ok {
		return 0
	}

	switch value := raw.(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func valueOrNowUTC(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value.UTC()
}

var (
	_ libstore.KeyStore             = (*providerStore)(nil)
	_ libstore.SubscriberStore      = (*providerStore)(nil)
	_ libstore.TxStore              = (*providerStore)(nil)
	_ libstore.TxPurger             = (*providerStore)(nil)
	_ libstore.TxSegmentStore       = (*providerStore)(nil)
	_ libstore.TxSegmentQueryStore  = (*providerStore)(nil)
	_ libstore.TxRecoveryStore      = (*providerStore)(nil)
	_ libstore.TxRecoveryQueryStore = (*providerStore)(nil)
	_ libstore.NonceStore           = (*providerStore)(nil)
	_ libstore.NoncePurger          = (*providerStore)(nil)
)
