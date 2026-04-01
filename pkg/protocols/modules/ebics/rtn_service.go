package ebics

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path"
	"strings"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/rtn"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	RTNServiceName = "EBICS RTN"

	defaultRTNRetryDelay = time.Minute
	rtnProviderLoopCount = 2

	contractValidationStatusMatched = "MATCHED"
)

var (
	errRTNAllProvidersFailed      = errors.New("failed to start all configured EBICS RTN providers")
	errUnsupportedRTNProcessing   = errors.New("unsupported RTN processing plan")
	errRTNAutoPullNoRule          = errors.New("no Gateway rule could be resolved for the RTN auto-pull")
	errRTNAutoPullAmbiguousRule   = errors.New("multiple Gateway rules match the RTN auto-pull request")
	errRTNAutoPullNoClient        = errors.New("no enabled EBICS client is available for the RTN auto-pull")
	errRTNAutoPullAmbiguousClient = errors.New(
		"multiple enabled EBICS clients match the RTN auto-pull request",
	)
)

type autoPullRuntime struct {
	resolved      *ebicsruntime.ResolvedPayloadRequest
	profile       *model.EbicsPayloadProfile
	rule          *model.Rule
	client        *model.Client
	remoteAgent   *model.RemoteAgent
	remoteAccount *model.RemoteAccount
	operation     *model.EbicsOperation
	transfer      *model.Transfer
}

type rtnProviderFactory func(*model.EbicsRTNProvider) (rtn.Provider, error)

type managedRTNProvider struct {
	config     *model.EbicsRTNProvider
	host       *model.EbicsHost
	subscriber *model.EbicsSubscriber
	provider   rtn.Provider
	mutex      sync.Mutex
}

// RTNService runs EBICS RTN providers in the background and persists incoming
// events plus their derived auto-pull operations.
type RTNService struct {
	db      *database.DB
	logger  *log.Logger
	state   utils.State
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mutex   sync.Mutex
	running map[int64]*managedRTNProvider

	providerFactory rtnProviderFactory
}

// NewRTNService instantiates the EBICS RTN background service.
func NewRTNService(db *database.DB) *RTNService {
	return &RTNService{
		db:              db,
		state:           utils.NewState(utils.StateOffline, ""),
		running:         map[int64]*managedRTNProvider{},
		providerFactory: newRTNProvider,
	}
}

// Start starts all enabled RTN providers and their ingestion loops.
func (s *RTNService) Start() error {
	if s.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	if s.logger == nil {
		s.logger = logging.NewLogger(RTNServiceName)
	}

	providers, err := s.loadManagedProviders()
	if err != nil {
		s.state.Set(utils.StateError, err.Error())
		return err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	started := 0
	var failures []string

	for _, managed := range providers {
		if err = managed.provider.Start(runCtx); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", managed.config.Name, err))
			s.logger.Warningf("Failed to start EBICS RTN provider %q: %v", managed.config.Name, err)
			if updateErr := s.updateProviderFailure(managed, err); updateErr != nil {
				s.logger.Warningf("Failed to persist RTN provider startup failure for %q: %v", managed.config.Name, updateErr)
			}

			continue
		}

		started++
		if updateErr := s.updateProviderConnection(managed, time.Time{}); updateErr != nil {
			s.logger.Warningf("Failed to persist RTN provider startup state for %q: %v", managed.config.Name, updateErr)
		}

		s.running[managed.config.ID] = managed
		s.wg.Add(rtnProviderLoopCount)
		go s.consumeProviderEvents(runCtx, managed)
		go s.consumeProviderErrors(runCtx, managed)
	}

	if len(providers) > 0 && started == 0 {
		cancel()
		err = fmt.Errorf("%w: %s", errRTNAllProvidersFailed, strings.Join(failures, "; "))
		s.state.Set(utils.StateError, err.Error())

		return err
	}

	s.cancel = cancel
	reason := ""
	if len(failures) > 0 {
		reason = fmt.Sprintf("%d provider(s) failed to start", len(failures))
	}
	s.state.Set(utils.StateRunning, reason)

	return nil
}

// Stop stops all RTN providers and waits for the ingestion loops to end.
func (s *RTNService) Stop(ctx context.Context) error {
	if !s.state.IsRunning() {
		return utils.ErrNotRunning
	}

	s.mutex.Lock()
	running := make([]*managedRTNProvider, 0, len(s.running))
	for _, managed := range s.running {
		running = append(running, managed)
	}
	s.running = map[int64]*managedRTNProvider{}
	cancel := s.cancel
	s.cancel = nil
	s.mutex.Unlock()

	if cancel != nil {
		cancel()
	}

	for _, managed := range running {
		if err := managed.provider.Stop(ctx); err != nil && !errors.Is(err, utils.ErrNotRunning) {
			s.logger.Warningf("Failed to stop EBICS RTN provider %q: %v", managed.config.Name, err)
		}
	}

	s.wg.Wait()
	s.state.Set(utils.StateOffline, "")

	return nil
}

// State returns the current Gateway service state for EBICS RTN orchestration.
func (s *RTNService) State() (utils.StateCode, string) {
	return s.state.Get()
}

func (s *RTNService) consumeProviderEvents(ctx context.Context, managed *managedRTNProvider) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case raw, ok := <-managed.provider.Events():
			if !ok {
				return
			}

			if err := s.handleProviderEvent(managed, &raw); err != nil {
				s.logger.Warningf("Failed to process RTN event from %q: %v", managed.config.Name, err)
				if updateErr := s.updateProviderFailure(managed, err); updateErr != nil {
					s.logger.Warningf("Failed to persist RTN provider event failure for %q: %v", managed.config.Name, updateErr)
				}
			}
		}
	}
}

func (s *RTNService) consumeProviderErrors(ctx context.Context, managed *managedRTNProvider) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-managed.provider.Errors():
			if !ok {
				return
			}
			if err == nil {
				continue
			}

			s.logger.Warningf("EBICS RTN provider %q reported an error: %v", managed.config.Name, err)
			if updateErr := s.updateProviderFailure(managed, err); updateErr != nil {
				s.logger.Warningf("Failed to persist RTN provider runtime failure for %q: %v", managed.config.Name, updateErr)
			}
		}
	}
}

func (s *RTNService) handleProviderEvent(managed *managedRTNProvider, raw *rtn.RawEvent) error {
	enriched := *raw
	enriched.Metadata = maps.Clone(raw.Metadata)
	if enriched.Metadata == nil {
		enriched.Metadata = map[string]any{}
	}
	enriched.Metadata["ebicsHostID"] = managed.host.ID
	enriched.Metadata["ebicsSubscriberID"] = managed.subscriber.ID
	enriched.Metadata["autoPullPolicy"] = managed.config.AutoPullPolicy
	enriched.Metadata["providerName"] = managed.config.Name
	enriched.Metadata["hostID"] = managed.host.HostID
	enriched.Metadata["partnerID"] = managed.subscriber.PartnerID
	enriched.Metadata["userID"] = managed.subscriber.UserID

	result, err := ebicsruntime.IngestRTNEvent(conf.GlobalConfig.GatewayName, &enriched, "", s)
	if err != nil {
		return fmt.Errorf("ingest RTN event: %w", err)
	}

	if err = s.updateProviderConnection(managed, valueOrNowUTC(enriched.ReceivedAt)); err != nil {
		s.logger.Warningf("Failed to persist RTN provider connection state for %q: %v", managed.config.Name, err)
	}

	if result.IsDuplicate {
		return nil
	}

	switch result.ProcessingPlan {
	case ebicsruntime.RTNProcessingPlanManualOnly:
		return nil
	case ebicsruntime.RTNProcessingPlanAutoPull, ebicsruntime.RTNProcessingPlanAutoFiltered:
		return s.executeAutoPull(managed, result.Event)
	default:
		return fmt.Errorf("%w: %q", errUnsupportedRTNProcessing, result.ProcessingPlan)
	}
}

func (s *RTNService) executeAutoPull(managed *managedRTNProvider, event *model.EbicsRTNEvent) error {
	event.Status = "PROCESSING"
	event.Attempts++
	event.NextRetryAt = time.Time{}
	event.LastError = ""
	if err := s.UpdateRTNEvent(event); err != nil {
		return fmt.Errorf("persist RTN event processing state: %w", err)
	}

	plan, err := ebicsruntime.BuildAutoPullPlan(event, managed.config)
	if err != nil {
		return s.failAutoPull(event, err)
	}

	runtime, err := s.prepareAutoPullRuntime(managed, event, plan)
	if err != nil {
		return s.failAutoPull(event, err)
	}

	if err = s.persistAutoPullRuntime(runtime); err != nil {
		return s.failAutoPull(event, err)
	}

	if event.PayloadMap == nil {
		event.PayloadMap = map[string]any{}
	}
	event.PayloadMap["autoPullOperationID"] = runtime.operation.ID
	event.PayloadMap["autoPullTransferID"] = runtime.transfer.ID
	event.PayloadMap["autoPullOrderType"] = runtime.operation.OrderType
	event.PayloadMap["autoPullStatus"] = runtime.operation.Status
	event.PayloadMap["autoPullOutcome"] = runtime.operation.GatewayOutcome
	event.PayloadMap["autoPullRetry"] = runtime.operation.RetryDecision
	if err = s.UpdateRTNEvent(event); err != nil {
		return fmt.Errorf("persist RTN event scheduled auto-pull state: %w", err)
	}

	return nil
}

func (s *RTNService) failAutoPull(event *model.EbicsRTNEvent, cause error) error {
	event.LastError = strings.TrimSpace(cause.Error())
	event.ProcessedAt = time.Time{}

	if database.IsValidationError(cause) {
		event.Status = "FAILED"
		event.NextRetryAt = time.Time{}
	} else {
		event.Status = "RETRYABLE"
		event.NextRetryAt = time.Now().UTC().Add(defaultRTNRetryDelay)
	}

	if err := s.UpdateRTNEvent(event); err != nil {
		return fmt.Errorf("persist RTN auto-pull failure state: %w", err)
	}

	return cause
}

func (s *RTNService) prepareAutoPullRuntime(
	managed *managedRTNProvider,
	event *model.EbicsRTNEvent,
	plan *ebicsruntime.AutoPullPlan,
) (*autoPullRuntime, error) {
	input := &ebicsruntime.PayloadRequestInput{
		ProfileName: strings.TrimSpace(plan.ProfileName),
		Subscriber: ebicsruntime.PayloadSubscriberRef{
			HostID:    managed.host.HostID,
			PartnerID: managed.subscriber.PartnerID,
			UserID:    managed.subscriber.UserID,
		},
		Service: &ebicsruntime.PayloadServiceRef{
			OrderType:     strings.TrimSpace(plan.OrderType),
			ServiceName:   readMetadataString(event.PayloadMap, "serviceName"),
			ServiceOption: readMetadataString(event.PayloadMap, "serviceOption"),
			Scope:         readMetadataString(event.PayloadMap, "scope"),
			MsgName:       readMetadataString(event.PayloadMap, "msgName"),
			ContainerType: readMetadataString(event.PayloadMap, "containerType"),
		},
		Metadata: maps.Clone(event.PayloadMap),
	}

	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	input.Metadata["correlationID"] = plan.CorrelationID
	input.Metadata["rtnEventID"] = event.ID
	input.Metadata["rtnProvider"] = managed.config.Name

	if targetDir := readMetadataString(event.PayloadMap, "targetDirectory"); targetDir != "" {
		input.Target = &ebicsruntime.PayloadTargetRef{Directory: targetDir}
	}

	resolved, err := ebicsruntime.ResolvePayloadRequest(
		input,
		"",
		map[string]any{},
		&runtimeProfileResolver{db: s.db},
	)
	if err != nil {
		return nil, fmt.Errorf("resolve RTN auto-pull payload request: %w", err)
	}

	validation, err := ebicsruntime.ValidateResolvedPayloadRequest(
		conf.GlobalConfig.GatewayName,
		resolved,
		&transferContractViewResolver{db: s.db},
	)
	if err != nil {
		return nil, fmt.Errorf("validate RTN auto-pull contract: %w", err)
	}

	if validation.Status != contractValidationStatusMatched {
		return nil, database.NewValidationError(validation.Message)
	}

	rule, err := s.resolveAutoPullRule(event.PayloadMap, resolved.Profile)
	if err != nil {
		return nil, err
	}
	if model.IsEbicsPayloadDownloadOrder(resolved.OrderType) && rule.IsSend {
		return nil, database.NewValidationError(
			"the RTN auto-pull rule must be a Gateway receive rule for a BTD payload")
	}
	if !model.IsEbicsPayloadDownloadOrder(resolved.OrderType) && !rule.IsSend {
		return nil, database.NewValidationError(
			"the RTN auto-pull rule must be a Gateway send rule for an upload payload")
	}

	remoteAccount, err := s.resolveAutoPullRemoteAccount(managed.subscriber)
	if err != nil {
		return nil, err
	}

	remoteAgent, err := s.resolveAutoPullRemoteAgent(remoteAccount.RemoteAgentID)
	if err != nil {
		return nil, err
	}

	client, err := s.resolveAutoPullClient(event.PayloadMap)
	if err != nil {
		return nil, err
	}

	if startErr := s.ensureAutoPullClientStarted(client); startErr != nil {
		return nil, startErr
	}

	operation, err := ebicsruntime.NewPayloadOperation(&ebicsruntime.OperationMappingInput{
		Owner:             conf.GlobalConfig.GatewayName,
		ClientID:          client.ID,
		RemoteAgentID:     remoteAgent.ID,
		RemoteAccountID:   remoteAccount.ID,
		EbicsHostID:       managed.host.ID,
		EbicsSubscriberID: managed.subscriber.ID,
		OrderType:         resolved.OrderType,
		OperationType:     model.EbicsOperationTypePayloadForRuntime(),
		Direction:         deriveRTNAutoPullDirection(resolved.OrderType),
		TransportMode:     model.EbicsTransportModeAutoTriggeredForRuntime(),
		CorrelationID:     plan.CorrelationID,
		ContractViewID:    resolved.ContractViewID,
		ResolvedRequest:   resolved,
	})
	if err != nil {
		return nil, fmt.Errorf("map RTN auto-pull operation: %w", err)
	}

	operation.RequestID = resolveRTNAutoPullRequestID(event, plan)
	operation.TransactionID = resolveRTNAutoPullTransactionID(resolved.OrderType, plan)
	operation.EbicsVersion = managed.host.ProtocolVersion
	operation.Status = model.EbicsOperationStatusWaitingPayloadTransferForRuntime()
	operation.RTNEventID = utils.NewNullInt64(event.ID)
	operation.MetadataMap["rtnProviderName"] = managed.config.Name
	operation.MetadataMap["rtnSource"] = event.Source
	operation.MetadataMap["autoPullReason"] = plan.Reason

	transfer := &model.Transfer{
		RuleID:          rule.ID,
		ClientID:        utils.NewNullInt64(client.ID),
		RemoteAccountID: utils.NewNullInt64(remoteAccount.ID),
		SrcFilename:     resolveRTNAutoPullFilename(event, plan),
		DestFilename:    resolveRTNAutoPullOutputName(event, plan),
		Start:           time.Now().UTC(),
		Status:          types.StatusPlanned,
		TransferInfo:    map[string]any{},
	}

	if target := strings.TrimSpace(resolvedTargetDirectory(resolved)); target != "" {
		transfer.LocalPath = path.Join(target, transfer.DestFilename)
	}

	return &autoPullRuntime{
		resolved:      resolved,
		profile:       resolved.Profile,
		rule:          rule,
		client:        client,
		remoteAgent:   remoteAgent,
		remoteAccount: remoteAccount,
		operation:     operation,
		transfer:      transfer,
	}, nil
}

func (s *RTNService) persistAutoPullRuntime(runtime *autoPullRuntime) error {
	if runtime == nil {
		return database.NewValidationError("the RTN auto-pull runtime is missing")
	}

	return s.db.Transaction(func(ses *database.Session) error {
		if err := ses.Insert(runtime.operation).Run(); err != nil {
			return fmt.Errorf("insert RTN auto-pull operation: %w", err)
		}

		if err := ses.Insert(runtime.transfer).Run(); err != nil {
			return fmt.Errorf("insert RTN auto-pull transfer: %w", err)
		}

		if err := ebicsruntime.BindTransferToOperation(runtime.operation, runtime.transfer.ID); err != nil {
			return fmt.Errorf("bind RTN auto-pull operation to transfer %d: %w", runtime.transfer.ID, err)
		}

		if err := ses.Update(runtime.operation).Run(); err != nil {
			return fmt.Errorf("persist RTN auto-pull operation transfer binding: %w", err)
		}

		return nil
	})
}

func deriveRTNAutoPullDirection(orderType string) string {
	if model.NormalizeEbicsPayloadOrderType(orderType) == "BTD" {
		return model.EbicsOperationDirectionInboundForRuntime()
	}

	return model.EbicsOperationDirectionOutboundForRuntime()
}

func (s *RTNService) resolveAutoPullRule(
	metadata map[string]any,
	profile *model.EbicsPayloadProfile,
) (*model.Rule, error) {
	if ruleID, ok := readMetadataInt64(metadata, "ruleID"); ok {
		rule := &model.Rule{}
		if err := s.db.Get(rule, "id=?", ruleID).Run(); err != nil {
			return nil, fmt.Errorf("load RTN auto-pull rule %d: %w", ruleID, err)
		}

		return rule, nil
	}

	if ruleName := readMetadataString(metadata, "ruleName"); ruleName != "" {
		var rules model.Rules
		if err := s.db.Select(&rules).Owner().Where("name=?", ruleName).Run(); err != nil {
			return nil, fmt.Errorf("load RTN auto-pull rule %q: %w", ruleName, err)
		}

		switch len(rules) {
		case 0:
			return nil, fmt.Errorf("%w: %q", errRTNAutoPullNoRule, ruleName)
		case 1:
			return rules[0], nil
		default:
			return nil, fmt.Errorf("%w: %q", errRTNAutoPullAmbiguousRule, ruleName)
		}
	}

	if profile != nil && profile.DefaultRuleID.Valid {
		rule := &model.Rule{}
		if err := s.db.Get(rule, "id=?", profile.DefaultRuleID.Int64).Run(); err != nil {
			return nil, fmt.Errorf(
				"load RTN auto-pull default rule %d for profile %q: %w",
				profile.DefaultRuleID.Int64,
				profile.Name,
				err,
			)
		}

		return rule, nil
	}

	return nil, errRTNAutoPullNoRule
}

func (s *RTNService) resolveAutoPullRemoteAccount(
	subscriber *model.EbicsSubscriber,
) (*model.RemoteAccount, error) {
	if subscriber == nil || !subscriber.RemoteAccountID.Valid {
		return nil, errTransferMissingRemoteAccount
	}

	account := &model.RemoteAccount{}
	if err := s.db.Get(account, "id=?", subscriber.RemoteAccountID.Int64).Run(); err != nil {
		return nil, fmt.Errorf(
			"load RTN auto-pull remote account %d for subscriber %q: %w",
			subscriber.RemoteAccountID.Int64,
			subscriber.Name,
			err,
		)
	}

	return account, nil
}

func (s *RTNService) resolveAutoPullRemoteAgent(remoteAgentID int64) (*model.RemoteAgent, error) {
	agent := &model.RemoteAgent{}
	if err := s.db.Get(agent, "id=?", remoteAgentID).Run(); err != nil {
		return nil, fmt.Errorf("load RTN auto-pull remote agent %d: %w", remoteAgentID, err)
	}

	return agent, nil
}

func (s *RTNService) resolveAutoPullClient(metadata map[string]any) (*model.Client, error) {
	clientName := readMetadataString(metadata, "clientName")
	var clients model.Clients

	query := s.db.Select(&clients).Owner().Where("protocol=? AND disabled=?", EBICS, false)
	if clientName != "" {
		query = query.Where("name=?", clientName)
	}

	if err := query.Run(); err != nil {
		if clientName != "" {
			return nil, fmt.Errorf("load RTN auto-pull client %q: %w", clientName, err)
		}

		return nil, fmt.Errorf("load enabled EBICS clients for RTN auto-pull: %w", err)
	}

	switch len(clients) {
	case 0:
		if clientName != "" {
			return nil, fmt.Errorf("%w: %q", errRTNAutoPullNoClient, clientName)
		}

		return nil, errRTNAutoPullNoClient
	case 1:
		return clients[0], nil
	default:
		if clientName != "" {
			return nil, fmt.Errorf("%w: %q", errRTNAutoPullAmbiguousClient, clientName)
		}

		return nil, errRTNAutoPullAmbiguousClient
	}
}

func (s *RTNService) ensureAutoPullClientStarted(client *model.Client) error {
	if client == nil {
		return errRTNAutoPullNoClient
	}

	service, ok := services.Clients[client.Name]
	if !ok || service == nil {
		service = NewClient(s.db, client)
		services.Clients[client.Name] = service
	}

	if state, _ := service.State(); state == utils.StateRunning {
		return nil
	}

	if err := service.Start(); err != nil && !errors.Is(err, utils.ErrAlreadyRunning) {
		return fmt.Errorf("start RTN auto-pull client %q: %w", client.Name, err)
	}

	return nil
}

func resolveRTNAutoPullRequestID(event *model.EbicsRTNEvent, plan *ebicsruntime.AutoPullPlan) string {
	if event != nil {
		if requestID := readMetadataString(event.PayloadMap, "requestID"); requestID != "" {
			return requestID
		}

		if correlationID := strings.TrimSpace(event.CorrelationID); correlationID != "" {
			return correlationID
		}

		if eventID := strings.TrimSpace(event.EventID); eventID != "" {
			return eventID
		}
	}

	if plan != nil && strings.TrimSpace(plan.CorrelationID) != "" {
		return strings.TrimSpace(plan.CorrelationID)
	}

	return ""
}

func resolveRTNAutoPullTransactionID(orderType string, plan *ebicsruntime.AutoPullPlan) string {
	if model.IsEbicsPayloadDownloadOrder(orderType) {
		return ""
	}
	if plan == nil || strings.TrimSpace(plan.CorrelationID) == "" {
		return ""
	}

	return "TX-" + strings.TrimSpace(plan.CorrelationID)
}

func resolveRTNAutoPullFilename(event *model.EbicsRTNEvent, plan *ebicsruntime.AutoPullPlan) string {
	if event != nil {
		for _, key := range []string{"srcFilename", "fileName", "remoteFilename", "documentName"} {
			if value := readMetadataString(event.PayloadMap, key); value != "" {
				return value
			}
		}
	}

	if plan != nil && strings.TrimSpace(plan.CorrelationID) != "" {
		return strings.TrimSpace(plan.CorrelationID) + ".xml"
	}

	return "ebics-rtn-autopull.xml"
}

func resolveRTNAutoPullOutputName(event *model.EbicsRTNEvent, plan *ebicsruntime.AutoPullPlan) string {
	if event != nil {
		for _, key := range []string{"outputName", "destFilename", "targetFileName"} {
			if value := readMetadataString(event.PayloadMap, key); value != "" {
				return value
			}
		}
	}

	return path.Base(resolveRTNAutoPullFilename(event, plan))
}

func resolvedTargetDirectory(resolved *ebicsruntime.ResolvedPayloadRequest) string {
	if resolved == nil || resolved.ResolvedTarget == nil {
		return ""
	}

	return strings.TrimSpace(resolved.ResolvedTarget.Directory)
}

func newRTNProvider(cfg *model.EbicsRTNProvider) (rtn.Provider, error) {
	if cfg == nil {
		return nil, database.NewValidationError("the RTN provider configuration is missing")
	}

	switch strings.ToUpper(strings.TrimSpace(cfg.Transport)) {
	case "WSS":
		endpoint, ok := readRTNConfigString(cfg.ConfigurationMap, "endpoint")
		if !ok || endpoint == "" {
			return nil, database.NewValidationError("the RTN provider endpoint is missing")
		}

		return rtn.NewWSSProvider(cfg.Name, endpoint, cfg.Enabled), nil
	default:
		return nil, database.NewValidationErrorf("%q is not a supported RTN transport", cfg.Transport)
	}
}

func (s *RTNService) loadManagedProviders() ([]*managedRTNProvider, error) {
	var rows model.EbicsRTNProviders
	if err := s.db.Select(&rows).Owner().Where("enabled=?", true).Run(); err != nil {
		return nil, fmt.Errorf("load enabled EBICS RTN providers: %w", err)
	}

	out := make([]*managedRTNProvider, 0, len(rows))
	for _, row := range rows {
		subscriber := &model.EbicsSubscriber{}
		if err := s.db.Get(subscriber, "id=?", row.EbicsSubscriberID).Run(); err != nil {
			return nil, fmt.Errorf("load RTN provider subscriber %d: %w", row.EbicsSubscriberID, err)
		}

		host := &model.EbicsHost{}
		if err := s.db.Get(host, "id=?", subscriber.EbicsHostID).Run(); err != nil {
			return nil, fmt.Errorf("load RTN provider host %d: %w", subscriber.EbicsHostID, err)
		}

		provider, err := s.providerFactory(row)
		if err != nil {
			return nil, fmt.Errorf("instantiate RTN provider %q: %w", row.Name, err)
		}

		out = append(out, &managedRTNProvider{
			config:     row,
			host:       host,
			subscriber: subscriber,
			provider:   provider,
		})
	}

	return out, nil
}

func (s *RTNService) updateProviderConnection(managed *managedRTNProvider, at time.Time) error {
	managed.mutex.Lock()
	defer managed.mutex.Unlock()

	if !at.IsZero() {
		managed.config.LastConnectionAt = at.UTC()
	}
	managed.config.LastError = ""

	if err := s.db.Update(managed.config).Run(); err != nil {
		return fmt.Errorf("update RTN provider connection state: %w", err)
	}

	return nil
}

func (s *RTNService) updateProviderFailure(managed *managedRTNProvider, cause error) error {
	managed.mutex.Lock()
	defer managed.mutex.Unlock()

	managed.config.LastError = strings.TrimSpace(cause.Error())

	if err := s.db.Update(managed.config).Run(); err != nil {
		return fmt.Errorf("update RTN provider failure state: %w", err)
	}

	return nil
}

// InsertRTNEvent persists a normalized RTN event.
func (s *RTNService) InsertRTNEvent(event *model.EbicsRTNEvent) error {
	return wrapRTNDBErr("insert RTN event", s.db.Insert(event).Run())
}

// UpdateRTNEvent persists a previously inserted RTN event.
func (s *RTNService) UpdateRTNEvent(event *model.EbicsRTNEvent) error {
	return wrapRTNDBErr("update RTN event", s.db.Update(event).Run())
}

// GetRTNEventByIdempotenceKey returns the RTN event matching the durable key.
//
//nolint:nilnil // a missing RTN event is the expected control-flow result here.
func (s *RTNService) GetRTNEventByIdempotenceKey(owner, key string) (*model.EbicsRTNEvent, error) {
	event := &model.EbicsRTNEvent{}
	if err := s.db.Get(event, "owner=? AND idempotence_key=?", owner, strings.TrimSpace(key)).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("retrieve RTN event by idempotence key: %w", err)
	}

	return event, nil
}

func readMetadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}

	raw, ok := metadata[key]
	if !ok {
		return ""
	}

	return strings.TrimSpace(fmt.Sprint(raw))
}

func readMetadataInt64(metadata map[string]any, key string) (int64, bool) {
	if metadata == nil {
		return 0, false
	}

	raw, ok := metadata[key]
	if !ok {
		return 0, false
	}

	switch value := raw.(type) {
	case int64:
		return value, true
	case int:
		return int64(value), true
	case float64:
		return int64(value), true
	case string:
		parsed, err := utils.ParseInt[int64](strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}

		return parsed, true
	default:
		return 0, false
	}
}

func readRTNConfigString(config map[string]any, key string) (string, bool) {
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

func laterOfNowUTC(reference time.Time) time.Time {
	now := time.Now().UTC()
	if reference.After(now) {
		return reference.UTC()
	}

	return now
}

func wrapRTNDBErr(action string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", action, err)
}

var _ ebicsruntime.RTNEventStore = (*RTNService)(nil)
