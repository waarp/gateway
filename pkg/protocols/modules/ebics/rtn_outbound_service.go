package ebics

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/rtn"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	RTNOutboundServiceName = "EBICS RTN Outbound"
	defaultRTNOutboundPoll = 15 * time.Second
)

type outboundTicker interface {
	Chan() <-chan time.Time
	Stop()
}

type outboundTimeTicker struct {
	ticker *time.Ticker
}

func (t *outboundTimeTicker) Chan() <-chan time.Time { return t.ticker.C }
func (t *outboundTimeTicker) Stop()                  { t.ticker.Stop() }

type outboundNotifier interface {
	Publish(ctx context.Context, payload []byte) error
}

type outboundNotifierFactory func(*model.EbicsRTNOutboundProvider) (outboundNotifier, error)

type RTNOutboundService struct {
	db      *database.DB
	logger  *log.Logger
	state   utils.State
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mutex   sync.Mutex
	now     func() time.Time
	ticker  func(time.Duration) outboundTicker
	factory outboundNotifierFactory
}

func NewRTNOutboundService(db *database.DB) *RTNOutboundService {
	return &RTNOutboundService{
		db:    db,
		state: utils.NewState(utils.StateOffline, ""),
		now:   func() time.Time { return time.Now().UTC() },
		ticker: func(d time.Duration) outboundTicker {
			return &outboundTimeTicker{ticker: time.NewTicker(d)}
		},
		factory: newRTNOutboundNotifier,
	}
}

func (s *RTNOutboundService) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}
	if s.logger == nil {
		s.logger = logging.NewLogger(RTNOutboundServiceName)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.mutex.Lock()
	s.cancel = cancel
	s.mutex.Unlock()

	s.wg.Add(1)
	go s.loop(runCtx, s.ticker(defaultRTNOutboundPoll))
	s.state.Set(utils.StateRunning, "")

	return nil
}

func (s *RTNOutboundService) Stop(context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.mutex.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mutex.Unlock()

	if cancel != nil {
		cancel()
	}

	s.wg.Wait()
	s.state.Set(utils.StateOffline, "")

	return nil
}

func (s *RTNOutboundService) State() (utils.StateCode, string) {
	return s.state.Get()
}

func (s *RTNOutboundService) loop(ctx context.Context, ticker outboundTicker) {
	defer s.wg.Done()
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.Chan():
			if err := s.dispatchDueNotificationsAt(ctx, s.now()); err != nil {
				s.state.Set(utils.StateError, err.Error())
				s.logger.Warningf("EBICS outbound RTN dispatch failed: %v", err)

				continue
			}

			s.state.Set(utils.StateRunning, "")
		}
	}
}

func (s *RTNOutboundService) dispatchDueNotificationsAt(ctx context.Context, now time.Time) error {
	now = now.UTC()

	var notifications model.EbicsRTNOutboundNotifications
	if err := s.db.Select(&notifications).Owner().
		Where(
			"(status=? OR (status=? AND next_retry_at<=?))",
			model.EbicsRTNOutboundNotificationStatusPendingForRuntime(),
			model.EbicsRTNOutboundNotificationStatusRetryableForRuntime(),
			now,
		).
		OrderBy("created_at", true).
		Run(); err != nil {
		return fmt.Errorf("load due outbound RTN notifications: %w", err)
	}

	for _, notification := range notifications {
		if err := s.dispatchOne(ctx, now, notification); err != nil {
			return err
		}
	}

	return nil
}

func (s *RTNOutboundService) dispatchOne(
	ctx context.Context,
	now time.Time,
	notification *model.EbicsRTNOutboundNotification,
) error {
	var provider model.EbicsRTNOutboundProvider
	if err := s.db.Get(&provider, "id=?", notification.ProviderID).Owner().Run(); err != nil {
		return fmt.Errorf("load outbound RTN provider %d: %w", notification.ProviderID, err)
	}
	if !provider.Enabled {
		return s.failDispatch(
			notification,
			&provider,
			now,
			database.NewValidationError("the outbound RTN provider is disabled"),
		)
	}

	notification.Status = model.EbicsRTNOutboundNotificationStatusProcessingForRuntime()
	notification.Attempts++
	notification.LastError = ""
	notification.NextRetryAt = time.Time{}
	if err := s.db.Update(notification).Run(); err != nil {
		return fmt.Errorf("mark outbound RTN notification %d as processing: %w", notification.ID, err)
	}

	notifier, factoryErr := s.factory(&provider)
	if factoryErr != nil {
		return s.failDispatch(notification, &provider, now, factoryErr)
	}

	payload := []byte(notification.Payload)
	publishErr := notifier.Publish(ctx, payload)
	if publishErr != nil {
		return s.failDispatch(notification, &provider, now, publishErr)
	}

	notification.Status = model.EbicsRTNOutboundNotificationStatusSentForRuntime()
	notification.SentAt = now
	notification.LastError = ""
	updateNotificationErr := s.db.Update(notification).Run()
	if updateNotificationErr != nil {
		return fmt.Errorf(
			"persist outbound RTN notification %d success: %w",
			notification.ID,
			updateNotificationErr,
		)
	}

	provider.LastConnectionAt = now
	provider.LastError = ""
	updateProviderErr := s.db.Update(&provider).Run()
	if updateProviderErr != nil {
		return fmt.Errorf("persist outbound RTN provider %d success: %w", provider.ID, updateProviderErr)
	}

	return nil
}

func (s *RTNOutboundService) failDispatch(
	notification *model.EbicsRTNOutboundNotification,
	provider *model.EbicsRTNOutboundProvider,
	now time.Time,
	cause error,
) error {
	maxAttempts := model.DefaultEbicsRTNOutboundMaxAttempts
	retryDelay := time.Duration(model.DefaultEbicsRTNOutboundRetryDelaySeconds) * time.Second
	if provider != nil {
		maxAttempts = readOutboundProviderConfigInt(provider.ConfigurationMap, "maxAttempts", maxAttempts)
		retryDelay = time.Duration(
			readOutboundProviderConfigInt(provider.ConfigurationMap, "retryDelaySeconds", int(retryDelay/time.Second)),
		) * time.Second
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if retryDelay <= 0 {
		retryDelay = time.Duration(model.DefaultEbicsRTNOutboundRetryDelaySeconds) * time.Second
	}

	notification.LastError = strings.TrimSpace(cause.Error())
	if database.IsValidationError(cause) || notification.Attempts >= maxAttempts {
		notification.Status = model.EbicsRTNOutboundNotificationStatusFailedForRuntime()
		notification.NextRetryAt = time.Time{}
	} else {
		notification.Status = model.EbicsRTNOutboundNotificationStatusRetryableForRuntime()
		notification.NextRetryAt = now.Add(retryDelay)
	}
	if err := s.db.Update(notification).Run(); err != nil {
		return fmt.Errorf("persist outbound RTN notification %d failure: %w", notification.ID, err)
	}

	if provider != nil {
		provider.LastError = strings.TrimSpace(cause.Error())
		if err := s.db.Update(provider).Run(); err != nil {
			return fmt.Errorf("persist outbound RTN provider %d failure: %w", provider.ID, err)
		}
	}

	return nil
}

func QueueRTNOutboundNotification(
	db *database.DB,
	providerID int64,
	set *model.EbicsServerReportingSet,
	item *model.EbicsServerReportingItem,
) (*model.EbicsRTNOutboundNotification, error) {
	if set == nil {
		return nil, database.NewValidationError("the EBICS server reporting set is missing")
	}
	if item == nil {
		return nil, database.NewValidationError("the EBICS server reporting item is missing")
	}

	var subscriber model.EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", set.EbicsSubscriberID.Int64).Owner().Run(); err != nil {
		return nil, fmt.Errorf("load subscriber for outbound RTN notification: %w", err)
	}
	var host model.EbicsHost
	if err := db.Get(&host, "id=?", set.EbicsHostID).Owner().Run(); err != nil {
		return nil, fmt.Errorf("load host for outbound RTN notification: %w", err)
	}

	now := time.Now().UTC()
	correlationID := fmt.Sprintf(
		"rtn-outbound-%d-%s-%d",
		set.ID,
		strings.ToLower(strings.TrimSpace(item.ItemKey)),
		now.UnixNano(),
	)

	notification := &model.EbicsRTNOutboundNotification{
		ProviderID:             providerID,
		EventType:              "REPORT_AVAILABLE",
		SourceOrderType:        set.SourceOrderType,
		CorrelationID:          correlationID,
		ServerReportingSetID:   utils.NewNullInt64(set.ID),
		ServerReportingItemKey: item.ItemKey,
		Status:                 model.EbicsRTNOutboundNotificationStatusPendingForRuntime(),
		PayloadMap: map[string]any{
			"source":                 conf.GlobalConfig.GatewayName,
			"eventType":              "REPORT_AVAILABLE",
			"eventID":                correlationID,
			"correlationID":          correlationID,
			"occurredAt":             now.Format(time.RFC3339),
			"hostID":                 host.HostID,
			"partnerID":              subscriber.PartnerID,
			"userID":                 subscriber.UserID,
			"orderTypeHint":          set.SourceOrderType,
			"serverReportingSetID":   set.ID,
			"serverReportingItemKey": item.ItemKey,
			"orderID":                item.OrderID,
			"serviceName":            item.ServiceName,
			"serviceOption":          item.ServiceOption,
			"scope":                  item.Scope,
			"msgName":                item.MsgName,
			"containerType":          item.ContainerType,
			"availability":           "AVAILABLE",
		},
	}

	if err := db.Insert(notification).Run(); err != nil {
		return nil, fmt.Errorf("insert outbound RTN notification: %w", err)
	}

	return notification, nil
}

func newRTNOutboundNotifier(provider *model.EbicsRTNOutboundProvider) (outboundNotifier, error) {
	if provider == nil {
		return nil, database.NewValidationError("the outbound RTN provider is missing")
	}

	switch strings.ToUpper(strings.TrimSpace(provider.Transport)) {
	case "WSS":
		endpoint, ok := readOutboundProviderConfigString(provider.ConfigurationMap, "endpoint")
		if !ok || endpoint == "" {
			return nil, database.NewValidationError("the outbound RTN provider endpoint is missing")
		}

		return rtn.NewWSSNotifier(provider.Name, endpoint), nil
	default:
		return nil, database.NewValidationErrorf("%q is not a supported outbound RTN transport", provider.Transport)
	}
}

func readOutboundProviderConfigString(config map[string]any, key string) (string, bool) {
	if config == nil {
		return "", false
	}
	value, ok := config[key]
	if !ok {
		return "", false
	}
	raw, ok := value.(string)
	if !ok {
		return "", false
	}

	return strings.TrimSpace(raw), true
}

func readOutboundProviderConfigInt(config map[string]any, key string, fallback int) int {
	if config == nil {
		return fallback
	}
	value, ok := config[key]
	if !ok {
		return fallback
	}
	switch raw := value.(type) {
	case int:
		return raw
	case int64:
		return int(raw)
	case float64:
		return int(raw)
	default:
		return fallback
	}
}
