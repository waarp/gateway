package ebics

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"
	libebicscrypto "code.waarp.fr/lib/ebics/ebics/crypto"
	libreturncode "code.waarp.fr/lib/ebics/ebics/returncode"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	ebicsClientTransactionStatusRunning = "RUNNING"
	ebicsClientTransactionStatusFailed  = "FAILED"
)

var (
	errTransferExecutionContractMismatch = errors.New("EBICS contract does not allow the selected payload profile")
	errTransferMissingRemoteAccount      = errors.New("EBICS transfer is missing a remote account binding")
	errTransferNoSubscriber              = errors.New("no enabled EBICS subscriber is linked to the remote account")
	errTransferAmbiguousSubscriber       = errors.New("multiple EBICS subscribers are linked to the remote account")
	errTransferDisabledProfile           = errors.New("the EBICS payload profile is disabled")
	errTransferNoRuleProfile             = errors.New("no enabled EBICS payload profile is bound to the Gateway rule")
	errTransferAmbiguousRuleProfile      = errors.New(
		"multiple enabled EBICS payload profiles are bound to the Gateway rule",
	)
	errTransferNoEndpointURL = errors.New(
		"no EBICS endpoint URL could be resolved from transfer, subscriber, host, partner or client configuration",
	)
	errTransferNoActiveLifecycle      = errors.New("no activated EBICS key lifecycle found")
	errTransferAmbiguousLifecycle     = errors.New("multiple activated EBICS key lifecycles found")
	errTransferNoValidatedBankKey     = errors.New("no validated EBICS bank key exists")
	errTransferAmbiguousBankKey       = errors.New("multiple validated EBICS bank keys exist")
	errTransferMissingCredential      = errors.New("EBICS credential is missing")
	errTransferNonRSAKey              = errors.New("EBICS credential does not contain an RSA public key")
	errNoScheduledOperation           = errors.New("no scheduled EBICS operation is linked to the transfer")
	errScheduledOperationTypeMismatch = errors.New("scheduled EBICS operation order type mismatch")
)

type transferClient struct {
	parent                   *Client
	pip                      *pipeline.Pipeline
	exec                     *payloadExecution
	finished                 bool
	scheduledOperation       *model.EbicsOperation
	scheduledOperationLoaded bool
}

type payloadExecution struct {
	host           *model.EbicsHost
	subscriber     *model.EbicsSubscriber
	profile        *model.EbicsPayloadProfile
	operation      *model.EbicsOperation
	transaction    *model.EbicsTransaction
	service        ebicsruntime.PayloadServiceRef
	endpointURL    string
	orderType      string
	orderID        string
	transactionID  string
	client         *libebicsclient.Client
	recovery       *libebicsclient.RecoveryManager
	requestSigner  libebicscrypto.Signer
	responseSigner libebicscrypto.Signer
	uploadCipher   libebicscrypto.E002Cipher
	downloadCipher libebicscrypto.E002Cipher
}

type transferContractViewResolver struct {
	db *database.DB
}

func newTransferClient(parent *Client, pip *pipeline.Pipeline) *transferClient {
	return &transferClient{
		parent: parent,
		pip:    pip,
	}
}

func (c *transferClient) Request() *pipeline.Error {
	exec, err := c.resolveExecution()
	if err != nil {
		return c.failPipeline(types.TeInternal, "prepare EBICS transfer execution", err)
	}

	c.exec = exec

	return nil
}

func (c *transferClient) Send(file protocol.SendFile) *pipeline.Error {
	if c.exec == nil {
		return c.failPipeline(types.TeInternal, "EBICS transfer was not initialized", nil)
	}

	if model.IsEbicsPayloadDownloadOrder(c.exec.orderType) {
		return c.failPipeline(types.TeInternal, "EBICS client send path is incompatible with a download order", nil)
	}

	payload, err := io.ReadAll(file)
	if err != nil {
		return c.failPipeline(types.TeDataTransfer, "read payload for EBICS upload", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout())
	defer cancel()

	err = c.exec.client.UploadBTU(ctx, libebicsclient.FlowBTURequired{
		URL:           c.exec.endpointURL,
		HostID:        c.exec.host.HostID,
		PartnerID:     c.exec.subscriber.PartnerID,
		UserID:        c.exec.subscriber.UserID,
		OrderID:       c.exec.orderID,
		Params:        c.buildUploadParams(),
		TransactionID: c.exec.transactionID,
		SegmentSize:   int(c.parent.config.MaxSegmentSize),
	}, libebicsclient.FlowBTUOptional{
		Recovery:         c.exec.recovery,
		RequestSigner:    c.exec.requestSigner,
		ResponseSigner:   c.exec.responseSigner,
		ValidateCodeList: true,
		Cipher:           c.exec.uploadCipher,
	}, payload)
	if err != nil {
		return c.handleEBICSError("upload EBICS payload", err)
	}

	return c.completeSuccess(len(payload))
}

func (c *transferClient) Receive(file protocol.ReceiveFile) *pipeline.Error {
	if c.exec == nil {
		return c.failPipeline(types.TeInternal, "EBICS transfer was not initialized", nil)
	}

	if !model.IsEbicsPayloadDownloadOrder(c.exec.orderType) {
		return c.failPipeline(types.TeInternal, "EBICS client receive path is incompatible with an upload order", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout())
	defer cancel()

	payload, resp, err := c.exec.client.DownloadBTD(ctx, libebicsclient.FlowBTDRequired{
		URL:       c.exec.endpointURL,
		HostID:    c.exec.host.HostID,
		PartnerID: c.exec.subscriber.PartnerID,
		UserID:    c.exec.subscriber.UserID,
		OrderID:   c.exec.orderID,
		Params:    c.buildDownloadParams(),
	}, libebicsclient.FlowBTDOptional{
		TransactionID:    c.exec.transactionID,
		RequestSigner:    c.exec.requestSigner,
		ResponseSigner:   c.exec.responseSigner,
		ValidateCodeList: true,
		Cipher:           c.exec.downloadCipher,
	})
	if syncErr := c.syncDownloadTransactionID(resp); syncErr != nil {
		return c.failPipeline(types.TeInternal, "persist EBICS download transaction correlation", syncErr)
	}
	if err != nil {
		return c.handleEBICSError("download EBICS payload", err)
	}

	if _, err = io.Copy(file, strings.NewReader(string(payload))); err != nil {
		return c.failPipeline(types.TeDataTransfer, "write downloaded EBICS payload", err)
	}

	return c.completeSuccess(len(payload))
}

func (c *transferClient) EndTransfer() *pipeline.Error {
	if c.finished || c.exec == nil {
		return nil
	}

	c.exec.operation.Status = model.EbicsOperationStatusCompletedForRuntime()
	c.exec.operation.FinishedAt = time.Now().UTC()
	preserveArchivedTransferLink(c.exec.operation)
	if err := c.parent.db.Update(c.exec.operation).Run(); err != nil {
		return pipeline.NewErrorWith(types.TeInternal, "persist EBICS operation finalization", err)
	}
	if err := c.syncRTNEventExecutionState(); err != nil {
		return pipeline.NewErrorWith(types.TeInternal, "persist RTN auto-pull success state", err)
	}

	c.finished = true

	return nil
}

func (c *transferClient) SendError(_ types.TransferErrorCode, msg string) {
	if c.exec == nil {
		return
	}

	c.exec.operation.Status = model.EbicsOperationStatusFailedForRuntime()
	c.exec.operation.Severity = model.EbicsOperationSeverityErrorForRuntime()
	c.exec.operation.GatewayOutcome = model.EbicsGatewayOutcomeTechnicalFatalFailureForRuntime()
	c.exec.operation.RetryDecision = model.EbicsRetryDecisionNoRetryForRuntime()
	c.exec.operation.TechnicalReturnMessage = strings.TrimSpace(msg)
	c.exec.operation.FinishedAt = time.Now().UTC()
	if err := c.parent.db.Update(c.exec.operation).Run(); err != nil {
		c.parent.logger.Warningf("Failed to persist EBICS client error state: %v", err)
	}
	if err := c.syncRTNEventExecutionState(); err != nil {
		c.parent.logger.Warningf("Failed to persist EBICS RTN auto-pull error state: %v", err)
	}

	c.finished = true
}

func (c *transferClient) resolveExecution() (*payloadExecution, error) {
	host, subscriber, err := c.resolveSubscriber()
	if err != nil {
		return nil, err
	}

	profile, err := c.resolvePayloadProfile()
	if err != nil {
		return nil, err
	}

	resolved, validation, err := c.resolvePayloadRequest(host, subscriber, profile)
	if err != nil {
		return nil, err
	}
	if validation.Status != "MATCHED" {
		return nil, fmt.Errorf("%w: profile=%q subscriber=%q status=%s",
			errTransferExecutionContractMismatch, profile.Name, subscriber.Name, validation.Status)
	}

	endpointURL, err := c.resolveEndpointURL(host, subscriber)
	if err != nil {
		return nil, err
	}

	operation, err := c.createOperation(host, subscriber, resolved)
	if err != nil {
		return nil, err
	}

	transaction, err := c.createTransaction(operation)
	if err != nil {
		return nil, err
	}

	if persistErr := c.persistTransferCorrelation(
		operation,
		profile,
		endpointURL,
		resolved,
	); persistErr != nil {
		return nil, persistErr
	}

	httpClient, err := c.buildHTTPClient(endpointURL)
	if err != nil {
		return nil, err
	}

	ebicsClient, err := libebicsclient.New(libebicsclient.RequiredConfig{Transport: httpClient})
	if err != nil {
		return nil, fmt.Errorf("initialize lib-ebics transfer client: %w", err)
	}

	requestSigner, err := c.resolveRequestSigner(subscriber.ID)
	if err != nil {
		return nil, err
	}
	responseSigner, err := c.resolveResponseSigner(host.ID)
	if err != nil {
		return nil, err
	}
	uploadCipher, err := c.resolveUploadCipher(host.ID)
	if err != nil {
		return nil, err
	}
	downloadCipher, err := c.resolveDownloadCipher(subscriber.ID)
	if err != nil {
		return nil, err
	}

	return &payloadExecution{
		host:           host,
		subscriber:     subscriber,
		profile:        profile,
		operation:      operation,
		transaction:    transaction,
		service:        resolved.ResolvedService,
		endpointURL:    endpointURL,
		orderType:      resolved.OrderType,
		orderID:        operation.RequestID,
		transactionID:  operation.TransactionID,
		client:         ebicsClient,
		recovery:       c.buildRecoveryManager(operation),
		requestSigner:  requestSigner,
		responseSigner: responseSigner,
		uploadCipher:   uploadCipher,
		downloadCipher: downloadCipher,
	}, nil
}

func (c *transferClient) resolveSubscriber() (*model.EbicsHost, *model.EbicsSubscriber, error) {
	trans := c.pip.TransCtx.Transfer
	if operation, loadErr := c.loadBoundOperation(); loadErr != nil && !errors.Is(loadErr, errNoScheduledOperation) {
		return nil, nil, loadErr
	} else if loadErr == nil && operation.EbicsHostID > 0 && operation.EbicsSubscriberID > 0 {
		var host model.EbicsHost
		if getErr := c.parent.db.Get(&host, "id=?", operation.EbicsHostID).Run(); getErr != nil {
			return nil, nil, fmt.Errorf(
				"load EBICS host %d for operation %d: %w",
				operation.EbicsHostID,
				operation.ID,
				getErr,
			)
		}

		var subscriber model.EbicsSubscriber
		if getErr := c.parent.db.Get(&subscriber, "id=?", operation.EbicsSubscriberID).Run(); getErr != nil {
			return nil, nil, fmt.Errorf(
				"load EBICS subscriber %d for operation %d: %w",
				operation.EbicsSubscriberID,
				operation.ID,
				getErr,
			)
		}

		return &host, &subscriber, nil
	}

	if !trans.RemoteAccountID.Valid {
		return nil, nil, errTransferMissingRemoteAccount
	}

	var subscribers model.EbicsSubscribers
	if err := c.parent.db.Select(&subscribers).Owner().Where(
		"remote_account_id=? AND account_role=? AND enabled=?",
		trans.RemoteAccountID.Int64,
		"CLIENT",
		true,
	).Run(); err != nil {
		return nil, nil, fmt.Errorf("load EBICS subscribers bound to remote account %d: %w",
			trans.RemoteAccountID.Int64, err)
	}

	switch len(subscribers) {
	case 0:
		return nil, nil, fmt.Errorf("%w: remoteAccountID=%d",
			errTransferNoSubscriber, trans.RemoteAccountID.Int64)
	case 1:
	default:
		return nil, nil, fmt.Errorf("%w: remoteAccountID=%d",
			errTransferAmbiguousSubscriber, trans.RemoteAccountID.Int64)
	}

	var host model.EbicsHost
	if err := c.parent.db.Get(&host, "id=?", subscribers[0].EbicsHostID).Run(); err != nil {
		return nil, nil, fmt.Errorf("load EBICS host %d for subscriber %q: %w",
			subscribers[0].EbicsHostID, subscribers[0].Name, err)
	}

	return &host, subscribers[0], nil
}

//nolint:nlreturn // keeping the immediate return next to the validation branch is clearer here.
func (c *transferClient) resolvePayloadProfile() (*model.EbicsPayloadProfile, error) {
	if operation, loadErr := c.loadBoundOperation(); loadErr != nil && !errors.Is(loadErr, errNoScheduledOperation) {
		return nil, loadErr
	} else if loadErr == nil {
		if profileName := readTransferString(operation.MetadataMap, "profileName"); profileName != "" {
			var profile model.EbicsPayloadProfile
			if getErr := c.parent.db.Get(
				&profile,
				"owner=? AND name=?",
				conf.GlobalConfig.GatewayName,
				profileName,
			).Run(); getErr != nil {
				return nil, fmt.Errorf("load EBICS payload profile %q from operation %d: %w",
					profileName, operation.ID, getErr)
			}
			if !profile.IsEnabled {
				return nil, fmt.Errorf("%w: %q", errTransferDisabledProfile, profileName)
			}
			return &profile, nil
		}
	}

	orderType := model.NormalizeEbicsPayloadOrderType(c.defaultOrderType())
	var profiles model.EbicsPayloadProfiles
	if err := c.parent.db.Select(&profiles).Owner().Where(
		"default_rule_id=? AND order_type=? AND is_enabled=?",
		c.pip.TransCtx.Rule.ID,
		orderType,
		true,
	).Run(); err != nil {
		return nil, fmt.Errorf("load EBICS payload profiles for rule %q: %w", c.pip.TransCtx.Rule.Name, err)
	}

	switch len(profiles) {
	case 0:
		return nil, fmt.Errorf("%w: rule=%q order=%q",
			errTransferNoRuleProfile, c.pip.TransCtx.Rule.Name, orderType)
	case 1:
		return profiles[0], nil
	default:
		return nil, fmt.Errorf("%w: rule=%q order=%q",
			errTransferAmbiguousRuleProfile, c.pip.TransCtx.Rule.Name, orderType)
	}
}

func (c *transferClient) resolvePayloadRequest(
	host *model.EbicsHost,
	subscriber *model.EbicsSubscriber,
	profile *model.EbicsPayloadProfile,
) (*ebicsruntime.ResolvedPayloadRequest, *ebicsruntime.ContractValidationResult, error) {
	input := &ebicsruntime.PayloadRequestInput{
		ProfileName: profile.Name,
		RuleName:    c.pip.TransCtx.Rule.Name,
		Subscriber: ebicsruntime.PayloadSubscriberRef{
			HostID:    host.HostID,
			PartnerID: subscriber.PartnerID,
			UserID:    subscriber.UserID,
		},
		Metadata: buildTransferMetadata(c.pip.TransCtx.Transfer.TransferInfo),
	}

	if c.pip.TransCtx.Rule.IsSend {
		input.File = &ebicsruntime.PayloadFileRef{Path: c.pip.TransCtx.Transfer.LocalPath}
	} else {
		input.Target = &ebicsruntime.PayloadTargetRef{Directory: c.pip.TransCtx.Transfer.LocalPath}
	}

	resolved, err := ebicsruntime.ResolvePayloadRequest(
		input,
		c.parent.config.ProfilePolicy,
		profile.MetadataMap,
		&runtimeProfileResolver{db: c.parent.db},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve EBICS payload request for transfer %d: %w",
			c.pip.TransCtx.Transfer.ID, err)
	}

	validation, err := ebicsruntime.ValidateResolvedPayloadRequest(
		conf.GlobalConfig.GatewayName,
		resolved,
		&transferContractViewResolver{db: c.parent.db},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("validate EBICS payload request against active contract: %w", err)
	}

	return resolved, validation, nil
}

func (c *transferClient) resolveEndpointURL(
	host *model.EbicsHost,
	subscriber *model.EbicsSubscriber,
) (string, error) {
	if operation, err := c.loadBoundOperation(); err != nil && !errors.Is(err, errNoScheduledOperation) {
		return "", err
	} else if err == nil {
		if endpoint := readTransferString(operation.MetadataMap, "endpointURL"); endpoint != "" {
			return endpoint, nil
		}
	}
	if subscriber.TransportURL != "" {
		return subscriber.TransportURL, nil
	}

	partnerConf := defaultPartnerConfig()
	if err := utils.JSONConvert(c.pip.TransCtx.RemoteAgent.ProtoConfig, partnerConf); err == nil {
		if endpoint := strings.TrimSpace(partnerConf.EndpointURL); endpoint != "" {
			return endpoint, nil
		}
	}

	if host.DefaultBankURL != "" {
		return host.DefaultBankURL, nil
	}
	if c.parent.config.EndpointURL != "" {
		return c.parent.config.EndpointURL, nil
	}

	return "", errTransferNoEndpointURL
}

func (c *transferClient) createOperation(
	host *model.EbicsHost,
	subscriber *model.EbicsSubscriber,
	resolved *ebicsruntime.ResolvedPayloadRequest,
) (*model.EbicsOperation, error) {
	if existing, err := c.loadScheduledOperation(); err != nil && !errors.Is(err, errNoScheduledOperation) {
		return nil, err
	} else if err == nil {
		existing.RequestID = c.resolveOrderID()
		if txID := c.resolveTransactionID(); txID != "" || existing.TransactionID == "" {
			existing.TransactionID = txID
		}
		existing.EbicsVersion = c.parent.config.ProtocolVersion
		existing.Status = ebicsClientTransactionStatusRunning
		existing.StartedAt = time.Now().UTC()

		if err = ebicsruntime.BindTransferToOperation(existing, c.pip.TransCtx.Transfer.ID); err != nil {
			return nil, fmt.Errorf("bind scheduled EBICS operation to transfer %d: %w",
				c.pip.TransCtx.Transfer.ID, err)
		}

		if err = c.parent.db.Update(existing).Run(); err != nil {
			return nil, fmt.Errorf("persist scheduled EBICS operation reuse: %w", err)
		}

		return existing, nil
	}

	transportMode := model.EbicsTransportModeAsyncForRuntime()
	if c.pip.TransCtx.EbicsContext != nil && c.pip.TransCtx.EbicsContext.RTNEventID > 0 {
		transportMode = model.EbicsTransportModeAutoTriggeredForRuntime()
	}

	operation, err := ebicsruntime.NewPayloadOperation(&ebicsruntime.OperationMappingInput{
		Owner:             conf.GlobalConfig.GatewayName,
		ClientID:          c.pip.TransCtx.Client.ID,
		RemoteAgentID:     c.pip.TransCtx.RemoteAgent.ID,
		RemoteAccountID:   c.pip.TransCtx.RemoteAccount.ID,
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OrderType:         resolved.OrderType,
		OperationType:     model.EbicsOperationTypePayloadForRuntime(),
		Direction:         c.operationDirection(),
		TransportMode:     transportMode,
		CorrelationID:     c.resolveCorrelationID(),
		ContractViewID:    resolved.ContractViewID,
		ResolvedRequest:   resolved,
	})
	if err != nil {
		return nil, fmt.Errorf("build EBICS client operation: %w", err)
	}

	operation.RequestID = c.resolveOrderID()
	operation.TransactionID = c.resolveTransactionID()
	operation.EbicsVersion = c.parent.config.ProtocolVersion
	operation.Status = ebicsClientTransactionStatusRunning
	operation.StartedAt = time.Now().UTC()
	if c.pip.TransCtx.EbicsContext != nil && c.pip.TransCtx.EbicsContext.RTNEventID > 0 {
		rtnEventID := c.pip.TransCtx.EbicsContext.RTNEventID
		operation.RTNEventID = nullableID(rtnEventID)
	}

	if err = c.parent.db.Insert(operation).Run(); err != nil {
		return nil, fmt.Errorf("insert EBICS client operation: %w", err)
	}

	if err = ebicsruntime.BindTransferToOperation(operation, c.pip.TransCtx.Transfer.ID); err != nil {
		return nil, fmt.Errorf("bind EBICS operation to transfer %d: %w", c.pip.TransCtx.Transfer.ID, err)
	}

	if err = c.parent.db.Update(operation).Run(); err != nil {
		return nil, fmt.Errorf("persist EBICS client operation transfer binding: %w", err)
	}

	return operation, nil
}

func (c *transferClient) loadScheduledOperation() (*model.EbicsOperation, error) {
	operation, err := c.loadBoundOperation()
	if err != nil {
		return nil, err
	}

	expectedOrderType := ""
	if c.pip.TransCtx.EbicsContext != nil {
		expectedOrderType = model.NormalizeEbicsPayloadOrderType(c.pip.TransCtx.EbicsContext.OrderType)
	}
	if expectedOrderType == "" {
		expectedOrderType = model.NormalizeEbicsPayloadOrderType(c.defaultOrderType())
	}

	if operation.OrderType != expectedOrderType {
		return nil, fmt.Errorf("%w: operationID=%d", errScheduledOperationTypeMismatch, operation.ID)
	}

	return operation, nil
}

func (c *transferClient) loadBoundOperation() (*model.EbicsOperation, error) {
	if c.scheduledOperationLoaded {
		if c.scheduledOperation == nil {
			return nil, errNoScheduledOperation
		}

		return c.scheduledOperation, nil
	}

	c.scheduledOperationLoaded = true
	trans := c.pip.TransCtx.Transfer
	if trans == nil {
		return nil, errNoScheduledOperation
	}

	if trans.ID > 0 {
		operation := &model.EbicsOperation{}
		if err := c.parent.db.Get(operation, "transfer_id=?", trans.ID).Run(); err == nil {
			c.scheduledOperation = operation
			return operation, nil
		} else if !database.IsNotFound(err) {
			return nil, fmt.Errorf("load scheduled EBICS operation bound to transfer %d: %w", trans.ID, err)
		}
	}

	return nil, errNoScheduledOperation
}

func (c *transferClient) createTransaction(
	operation *model.EbicsOperation,
) (*model.EbicsTransaction, error) {
	if strings.TrimSpace(operation.TransactionID) == "" {
		return nil, nil //nolint:nilnil // nil row means the download has not materialized a server transaction yet
	}

	row := &model.EbicsTransaction{
		EbicsOperationID:  nullableID(operation.ID),
		EbicsHostID:       operation.EbicsHostID,
		EbicsSubscriberID: operation.EbicsSubscriberID,
		TransactionID:     operation.TransactionID,
		OrderType:         operation.OrderType,
		TransferID:        nullableID(c.pip.TransCtx.Transfer.ID),
		Status:            ebicsClientTransactionStatusRunning,
		Direction:         operation.Direction,
		TotalSize:         c.pip.TransCtx.Transfer.Filesize,
		MetadataMap:       map[string]any{},
	}

	if err := c.parent.db.Insert(row).Run(); err != nil {
		return nil, fmt.Errorf("insert EBICS client transaction: %w", err)
	}

	return row, nil
}

func (c *transferClient) persistTransferCorrelation(
	operation *model.EbicsOperation,
	profile *model.EbicsPayloadProfile,
	endpointURL string,
	resolved *ebicsruntime.ResolvedPayloadRequest,
) error {
	trans := c.pip.TransCtx.Transfer
	if operation.MetadataMap == nil {
		operation.MetadataMap = map[string]any{}
	}
	operation.MetadataMap["endpointURL"] = endpointURL
	operation.MetadataMap["profileName"] = profile.Name
	enrichOperationMetadata(operation, resolved)
	if err := c.parent.db.Update(operation).Run(); err != nil {
		return fmt.Errorf("update EBICS operation %d with runtime metadata: %w", operation.ID, err)
	}

	if err := c.parent.db.Update(trans).Run(); err != nil {
		return fmt.Errorf("update transfer %d after EBICS correlation enrichment: %w", trans.ID, err)
	}

	return nil
}

func (c *transferClient) buildHTTPClient(endpointURL string) (*http.Client, error) {
	parsedURL, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("parse EBICS endpoint URL %q: %w", endpointURL, err)
	}

	serverName := parsedURL.Hostname()
	if serverName == "" {
		return nil, fmt.Errorf("%w: %q", errMissingEndpointServerName, endpointURL)
	}

	rootCAs := utils.TLSCertPool()
	for _, cred := range c.pip.TransCtx.RemoteAgentCreds {
		if cred.Type != auth.TLSTrustedCertificate {
			continue
		}

		chain, parseErr := utils.ParsePEMCertChain(cred.Value)
		if parseErr != nil {
			return nil, fmt.Errorf("parse trusted TLS certificate %q for EBICS remote agent: %w", cred.Name, parseErr)
		}

		rootCAs.AddCert(chain[0])
	}

	globalTLSConf := &tls.Config{RootCAs: rootCAs}
	if err = auth.AddTLSAuthorities(c.pip.DB, globalTLSConf); err != nil {
		return nil, fmt.Errorf("load TLS authorities for EBICS client: %w", err)
	}

	certs := make([]tls.Certificate, 0, len(c.pip.TransCtx.RemoteAccountCreds))
	for _, cred := range c.pip.TransCtx.RemoteAccountCreds {
		if cred.Type != auth.TLSCertificate {
			continue
		}

		cert, parseErr := utils.X509KeyPair(cred.Value, cred.Value2)
		if parseErr != nil {
			return nil, fmt.Errorf("parse TLS client certificate %q for EBICS transfer: %w", cred.Name, parseErr)
		}

		certs = append(certs, cert)
	}

	httpClient, err := libebicsclient.NewProductionHTTPClient(
		libebicsclient.ProductionHTTPClientRequired{ServerName: serverName},
		libebicsclient.ProductionHTTPClientOptional{
			RootCAs:        rootCAs,
			ClientCerts:    certs,
			MinTLSVersion:  c.parent.config.MinTLSVersion.TLS(),
			RequestTimeout: c.requestTimeout(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("build EBICS HTTP transport for %q: %w", endpointURL, err)
	}

	return httpClient, nil
}

func (c *transferClient) resolveRequestSigner(subscriberID int64) (libebicscrypto.Signer, error) {
	credential, err := c.resolveActiveLifecycleCredential(subscriberID, model.EbicsKeyUsageAuthenticationForRuntime())
	if err != nil {
		return nil, err
	}

	return signerFromCredential(credential)
}

func (c *transferClient) resolveResponseSigner(hostID int64) (libebicscrypto.Signer, error) {
	if !c.parent.config.VerifyBankKeys {
		return nil, nil //nolint:nilnil // disabled bank-key verification intentionally disables response signature checks
	}

	key, err := c.resolveBankKey(hostID, "SIGNATURE")
	if err != nil {
		return nil, err
	}

	certs, parseErr := libebicscrypto.ParseCertificatesPEM([]byte(key.PublicKey))
	if parseErr != nil {
		return nil, fmt.Errorf("parse validated EBICS bank signature certificate for host %d: %w", hostID, parseErr)
	}

	return &libebicscrypto.XMLDSigSigner{
		Certificate:         certs[0],
		RequireReferenceURI: true,
	}, nil
}

func (c *transferClient) resolveUploadCipher(hostID int64) (libebicscrypto.E002Cipher, error) {
	key, err := c.resolveBankKey(hostID, "ENCRYPT")
	if err != nil {
		return nil, err
	}

	if certs, parseErr := libebicscrypto.ParseCertificatesPEM([]byte(key.PublicKey)); parseErr == nil && len(certs) != 0 {
		return libebicscrypto.NewE002CipherWithCertificate(certs[0]), nil
	}

	pub, parseErr := libebicscrypto.ParseRSAPublicKey([]byte(key.PublicKey))
	if parseErr != nil {
		return nil, fmt.Errorf("parse validated EBICS bank encryption public key for host %d: %w", hostID, parseErr)
	}

	return libebicscrypto.NewE002CipherWithPublicKey(pub), nil
}

func (c *transferClient) resolveDownloadCipher(subscriberID int64) (libebicscrypto.E002Cipher, error) {
	credential, err := c.resolveActiveLifecycleCredential(subscriberID, model.EbicsKeyUsageEncryptionForRuntime())
	if err != nil {
		return nil, err
	}

	priv, cert, pub, err := parseCredentialKeyPair(credential)
	if err != nil {
		return nil, err
	}

	return libebicscrypto.NewE002CipherWithKeys(pub, cert, priv), nil
}

func (c *transferClient) resolveActiveLifecycleCredential(subscriberID int64, usage string) (*model.Credential, error) {
	var lifecycles model.EbicsKeyLifecycles
	if err := c.parent.db.Select(&lifecycles).Owner().Where(
		"ebics_subscriber_id=? AND key_usage=? AND status=?",
		subscriberID,
		usage,
		"ACTIVATED",
	).Run(); err != nil {
		return nil, fmt.Errorf("load active EBICS key lifecycles for subscriber %d and usage %q: %w",
			subscriberID, usage, err)
	}

	switch len(lifecycles) {
	case 0:
		return nil, fmt.Errorf("%w: subscriberID=%d usage=%q",
			errTransferNoActiveLifecycle, subscriberID, usage)
	case 1:
	default:
		return nil, fmt.Errorf("%w: subscriberID=%d usage=%q",
			errTransferAmbiguousLifecycle, subscriberID, usage)
	}

	var credential model.Credential
	if err := c.parent.db.Get(&credential, "id=?", lifecycles[0].CurrentCredentialID).Run(); err != nil {
		return nil, fmt.Errorf("load EBICS credential %d for subscriber %d and usage %q: %w",
			lifecycles[0].CurrentCredentialID, subscriberID, usage, err)
	}

	return &credential, nil
}

func (c *transferClient) resolveBankKey(hostID int64, keyType string) (*model.EbicsBankKey, error) {
	var keys model.EbicsBankKeys
	if err := c.parent.db.Select(&keys).Owner().Where(
		"ebics_host_id=? AND key_type=? AND state=?",
		hostID,
		keyType,
		"validated",
	).Run(); err != nil {
		return nil, fmt.Errorf("load validated EBICS bank key %q for host %d: %w", keyType, hostID, err)
	}

	switch len(keys) {
	case 0:
		return nil, fmt.Errorf("%w: hostID=%d keyType=%q",
			errTransferNoValidatedBankKey, hostID, keyType)
	case 1:
		return keys[0], nil
	default:
		return nil, fmt.Errorf("%w: hostID=%d keyType=%q",
			errTransferAmbiguousBankKey, hostID, keyType)
	}
}

func (c *transferClient) buildRecoveryManager(operation *model.EbicsOperation) *libebicsclient.RecoveryManager {
	if !c.parent.config.AllowRecovery {
		return nil
	}

	return libebicsclient.NewProductionRecoveryManager(libebicsclient.ProductionRecoveryOptional{
		Store: newClientRecoveryStore(c.parent.db, operation, c.pip.TransCtx.Transfer),
	})
}

func (c *transferClient) buildUploadParams() *ebicsxml.BTUOrderParams {
	return &ebicsxml.BTUOrderParams{
		Service:  restrictedServiceFromPayloadService(&c.exec.service),
		FileName: c.resolveUploadFilename(),
	}
}

func (c *transferClient) buildDownloadParams() *ebicsxml.BTDOrderParams {
	return &ebicsxml.BTDOrderParams{
		Service: restrictedServiceFromPayloadService(&c.exec.service),
	}
}

func (c *transferClient) resolveUploadFilename() string {
	if name := strings.TrimSpace(c.pip.TransCtx.Transfer.SrcFilename); name != "" {
		return name
	}

	return strings.TrimSpace(c.pip.TransCtx.Transfer.RemoteTransferID + ".xml")
}

func (c *transferClient) resolveOrderID() string {
	if operation, err := c.loadBoundOperation(); err == nil {
		if value := strings.TrimSpace(operation.RequestID); value != "" {
			return value
		}
	}
	if c.pip.TransCtx.EbicsContext != nil {
		if value := strings.TrimSpace(c.pip.TransCtx.EbicsContext.RequestID); value != "" {
			return value
		}
	}

	return strings.TrimSpace(c.pip.TransCtx.Transfer.RemoteTransferID)
}

func (c *transferClient) resolveCorrelationID() string {
	if operation, err := c.loadBoundOperation(); err == nil {
		if value := strings.TrimSpace(operation.CorrelationID); value != "" {
			return value
		}
	}
	if c.pip.TransCtx.EbicsContext != nil {
		if value := strings.TrimSpace(c.pip.TransCtx.EbicsContext.CorrelationID); value != "" {
			return value
		}
	}

	return strings.TrimSpace(c.pip.TransCtx.Transfer.RemoteTransferID)
}

func (c *transferClient) resolveTransactionID() string {
	if operation, err := c.loadBoundOperation(); err == nil {
		if value := strings.TrimSpace(operation.TransactionID); value != "" {
			return value
		}
	}
	if c.pip.TransCtx.EbicsContext != nil {
		if value := strings.TrimSpace(c.pip.TransCtx.EbicsContext.TransactionID); value != "" {
			return value
		}
	}
	if model.IsEbicsPayloadDownloadOrder(c.defaultOrderType()) {
		return ""
	}

	return "TX-" + c.resolveCorrelationID()
}

func (c *transferClient) defaultOrderType() string {
	if c.pip.TransCtx.Rule.IsSend {
		return "BTU"
	}

	return "BTD"
}

func (c *transferClient) operationDirection() string {
	if c.pip.TransCtx.Rule.IsSend {
		return "OUTBOUND"
	}

	return "INBOUND"
}

func (c *transferClient) requestTimeout() time.Duration {
	return time.Duration(c.parent.config.RequestTimeout) * time.Second
}

func (c *transferClient) completeSuccess(payloadSize int) *pipeline.Error {
	if c.exec.transaction != nil {
		c.exec.transaction.Status = model.EbicsTransactionStatusCompletedForRuntime()
		c.exec.transaction.TotalSize = int64(payloadSize)
		if err := c.parent.db.Update(c.exec.transaction).Run(); err != nil {
			return c.failPipeline(types.TeInternal, "persist successful EBICS transaction", err)
		}
	}

	if err := ebicsruntime.UpdateOperationOutcomeFromReturnCodes(c.exec.operation, "", "", "", ""); err != nil {
		return c.failPipeline(types.TeInternal, "derive successful EBICS outcome", err)
	}

	c.exec.operation.FinishedAt = time.Now().UTC()
	if err := c.parent.db.Update(c.exec.operation).Run(); err != nil {
		return c.failPipeline(types.TeInternal, "persist successful EBICS operation", err)
	}

	return nil
}

func (c *transferClient) handleEBICSError(action string, err error) *pipeline.Error {
	if code, ok := libreturncode.FromError(err); ok {
		technicalCode, technicalMsg := "", ""
		businessCode, businessMsg := "", ""

		switch code.Scope {
		case libreturncode.ScopeTechnical:
			technicalCode = string(code.Value)
			technicalMsg = err.Error()
		case libreturncode.ScopeBusiness:
			businessCode = string(code.Value)
			businessMsg = err.Error()
		}

		if mapErr := ebicsruntime.UpdateOperationOutcomeFromReturnCodes(
			c.exec.operation,
			technicalCode,
			technicalMsg,
			businessCode,
			businessMsg,
		); mapErr != nil {
			return c.failPipeline(types.TeInternal, "derive EBICS return-code outcome", mapErr)
		}
	}

	return c.failPipeline(types.TeDataTransfer, action, err)
}

func (c *transferClient) failPipeline(code types.TransferErrorCode, details string, cause error) *pipeline.Error {
	if c.exec != nil {
		c.persistFailedExecution(details, cause)
	}

	if cause != nil {
		return pipeline.NewErrorWith(code, details, cause)
	}

	return pipeline.NewError(code, details)
}

func (c *transferClient) persistFailedExecution(details string, cause error) {
	c.exec.operation.Status = model.EbicsOperationStatusFailedForRuntime()
	if strings.TrimSpace(c.exec.operation.GatewayOutcome) == "" ||
		c.exec.operation.GatewayOutcome == model.EbicsGatewayOutcomePendingBankForRuntime() {
		c.exec.operation.GatewayOutcome = model.EbicsGatewayOutcomeTechnicalFatalFailureForRuntime()
	}

	if strings.TrimSpace(c.exec.operation.RetryDecision) == "" {
		c.exec.operation.RetryDecision = model.EbicsRetryDecisionNoRetryForRuntime()
	}

	if c.exec.operation.ManualActionRequired {
		c.exec.operation.Severity = model.EbicsOperationSeverityWarningForRuntime()
	} else {
		c.exec.operation.Severity = model.EbicsOperationSeverityErrorForRuntime()
	}

	if strings.TrimSpace(c.exec.operation.TechnicalReturnMessage) == "" {
		if cause != nil {
			c.exec.operation.TechnicalReturnMessage = strings.TrimSpace(cause.Error())
		} else {
			c.exec.operation.TechnicalReturnMessage = strings.TrimSpace(details)
		}
	}

	c.exec.operation.FinishedAt = time.Now().UTC()
	if err := c.parent.db.Update(c.exec.operation).Run(); err != nil {
		c.parent.logger.Warningf("Failed to persist EBICS operation failure state: %v", err)
	}

	if c.exec.transaction != nil {
		c.exec.transaction.Status = ebicsClientTransactionStatusFailed
		if err := c.parent.db.Update(c.exec.transaction).Run(); err != nil {
			c.parent.logger.Warningf("Failed to persist EBICS transaction failure state: %v", err)
		}
	}

	if err := c.syncRTNEventExecutionState(); err != nil {
		c.parent.logger.Warningf("Failed to persist EBICS RTN auto-pull failure state: %v", err)
	}

	c.finished = true
}

func (c *transferClient) syncDownloadTransactionID(resp *ebicsxml.EbicsResponse) error {
	if resp == nil || resp.Header == nil || resp.Header.Mutable == nil {
		return nil
	}

	txID := strings.TrimSpace(resp.Header.Mutable.TransactionID)
	if txID == "" || txID == c.exec.operation.TransactionID {
		return nil
	}

	c.exec.operation.TransactionID = txID
	if err := c.parent.db.Update(c.exec.operation).Run(); err != nil {
		return fmt.Errorf("update EBICS operation transaction ID %q: %w", txID, err)
	}

	if c.exec.transaction == nil {
		row, err := c.createTransaction(c.exec.operation)
		if err != nil {
			return fmt.Errorf("create EBICS download transaction %q: %w", txID, err)
		}
		c.exec.transaction = row
	} else {
		c.exec.transaction.TransactionID = txID
		if err := c.parent.db.Update(c.exec.transaction).Run(); err != nil {
			return fmt.Errorf("update EBICS transaction %d with transaction ID %q: %w",
				c.exec.transaction.ID, txID, err)
		}
	}

	c.exec.transactionID = txID

	return nil
}

func (c *transferClient) syncRTNEventExecutionState() error {
	if c.exec == nil {
		return nil
	}

	eventID := int64(0)
	if c.exec.operation != nil && c.exec.operation.RTNEventID.Valid {
		eventID = c.exec.operation.RTNEventID.Int64
	}
	if eventID <= 0 && c.pip != nil && c.pip.TransCtx != nil && c.pip.TransCtx.EbicsContext != nil {
		eventID = c.pip.TransCtx.EbicsContext.RTNEventID
	}
	if eventID <= 0 {
		return nil
	}

	event := &model.EbicsRTNEvent{}
	if err := c.parent.db.Get(event, "id=?", eventID).Run(); err != nil {
		return fmt.Errorf("load RTN event %d for EBICS auto-pull synchronization: %w", eventID, err)
	}
	if event.PayloadMap == nil {
		event.PayloadMap = map[string]any{}
	}

	event.PayloadMap["autoPullOperationID"] = c.exec.operation.ID
	if c.exec.operation.TransferID.Valid {
		event.PayloadMap["autoPullTransferID"] = c.exec.operation.TransferID.Int64
	} else if archivedTransferID := readTransferInt64(
		c.exec.operation.MetadataMap,
		operationMetadataKeyArchivedTransferID,
	); archivedTransferID > 0 {
		event.PayloadMap["autoPullTransferID"] = archivedTransferID
	}
	event.PayloadMap["autoPullOrderType"] = c.exec.operation.OrderType
	event.PayloadMap["autoPullStatus"] = c.exec.operation.Status
	event.PayloadMap["autoPullOutcome"] = c.exec.operation.GatewayOutcome
	event.PayloadMap["autoPullRetry"] = c.exec.operation.RetryDecision

	lastError := strings.TrimSpace(firstNonEmpty(
		c.exec.operation.TechnicalReturnMessage,
		c.exec.operation.BusinessReturnMessage,
		event.LastError,
	))

	switch c.exec.operation.Status {
	case model.EbicsOperationStatusCompletedForRuntime(),
		model.EbicsOperationStatusCompletedWithWarningsForRuntime():
		event.Status = "PROCESSED"
		event.ProcessedAt = laterOfNowUTC(event.ReceivedAt)
		event.NextRetryAt = time.Time{}
		event.LastError = ""
	case model.EbicsOperationStatusFailedForRuntime():
		event.ProcessedAt = time.Time{}
		event.LastError = lastError
		switch c.exec.operation.RetryDecision {
		case model.EbicsRetryDecisionAutoRetryAllowedForRuntime(),
			model.EbicsRetryDecisionRecoveryRequiredForRuntime():
			event.Status = "RETRYABLE"
			event.NextRetryAt = time.Now().UTC().Add(defaultRTNRetryDelay)
		default:
			event.Status = "FAILED"
			event.NextRetryAt = time.Time{}
		}
	default:
		event.Status = "PROCESSING"
		event.ProcessedAt = time.Time{}
		event.NextRetryAt = time.Time{}
		if event.LastError == "" {
			event.LastError = lastError
		}
	}

	if err := c.parent.db.Update(event).Run(); err != nil {
		return fmt.Errorf("persist RTN event %d after EBICS auto-pull execution: %w", eventID, err)
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

//nolint:nlreturn // nil contract snapshots are an expected control-flow outcome here.
func (r *transferContractViewResolver) GetActiveContractView(
	owner, hostID, partnerID, userID string,
) (*model.EbicsContractView, []model.EbicsContractViewItem, error) {
	host := &model.EbicsHost{}
	if err := r.db.Get(host, "host_id=? AND owner=?", hostID, owner).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, nil //nolint:nilnil // nil view is the expected "no active contract" signal
		}
		return nil, nil, fmt.Errorf("load EBICS host %q for contract resolution: %w", hostID, err)
	}

	subscriber := &model.EbicsSubscriber{}
	if err := r.db.Get(
		subscriber,
		"ebics_host_id=? AND partner_id=? AND user_id=? AND owner=?",
		host.ID,
		partnerID,
		userID,
		owner,
	).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, nil //nolint:nilnil // nil subscriber is the expected "no active contract" signal
		}
		return nil, nil, fmt.Errorf("load EBICS subscriber %q/%q for contract resolution: %w", partnerID, userID, err)
	}

	view := &model.EbicsContractView{}
	if err := r.db.Get(
		view,
		"ebics_host_id=? AND ebics_subscriber_id=? AND status=? AND owner=?",
		host.ID,
		subscriber.ID,
		"ACTIVE",
		owner,
	).OrderBy("fetched_at", false).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, nil //nolint:nilnil // nil active view is the expected "no active contract" signal
		}
		return nil, nil, fmt.Errorf("load active EBICS contract view for subscriber %q/%q: %w", partnerID, userID, err)
	}

	var rows model.EbicsContractViewItems
	if err := r.db.Select(&rows).Owner().Where("contract_view_id=?", view.ID).Run(); err != nil {
		return nil, nil, fmt.Errorf("load EBICS contract view items for view %d: %w", view.ID, err)
	}

	items := make([]model.EbicsContractViewItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, *row)
	}

	return view, items, nil
}

//nolint:nilnil // nil catalog is the expected "no active standard catalog" signal here.
func (r *transferContractViewResolver) GetActiveStandardBTFCatalog(
	owner, scope string,
) (*model.EbicsStandardBTFCatalog, []model.EbicsStandardBTFEntry, error) {
	catalog := &model.EbicsStandardBTFCatalog{}
	if err := r.db.Get(
		catalog,
		"owner=? AND scope=? AND status=?",
		owner,
		strings.ToUpper(strings.TrimSpace(scope)),
		"ACTIVE",
	).OrderBy("updated_at", false).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, nil, nil
		}

		return nil, nil, fmt.Errorf(
			"load active EBICS standard BTF catalog for scope %q: %w",
			scope,
			err,
		)
	}

	var rows model.EbicsStandardBTFEntries
	if err := r.db.Select(&rows).Owner().Where("catalog_id=?", catalog.ID).Run(); err != nil {
		return nil, nil, fmt.Errorf(
			"load EBICS standard BTF entries for catalog %d: %w",
			catalog.ID,
			err,
		)
	}

	entries := make([]model.EbicsStandardBTFEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, *row)
	}

	return catalog, entries, nil
}

type runtimeProfileResolver struct {
	db *database.DB
}

func (r *runtimeProfileResolver) GetPayloadProfile(owner, name string) (*model.EbicsPayloadProfile, error) {
	profile := &model.EbicsPayloadProfile{}
	if err := r.db.Get(profile, "owner=? AND name=?", owner, name).Run(); err != nil {
		return nil, fmt.Errorf("load EBICS payload profile %q for runtime resolution: %w", name, err)
	}

	return profile, nil
}

func signerFromCredential(credential *model.Credential) (libebicscrypto.Signer, error) {
	priv, cert, _, err := parseCredentialKeyPair(credential)
	if err != nil {
		return nil, err
	}

	return &libebicscrypto.XMLDSigSigner{
		PrivateKey:          priv,
		Certificate:         cert,
		RequireReferenceURI: true,
	}, nil
}

func parseCredentialKeyPair(credential *model.Credential) (*rsa.PrivateKey, *x509.Certificate, *rsa.PublicKey, error) {
	if credential == nil {
		return nil, nil, nil, errTransferMissingCredential
	}

	priv, err := libebicscrypto.ParsePrivateKeyPEM([]byte(credential.Value2))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse EBICS credential private key %q: %w", credential.Name, err)
	}

	certs, err := libebicscrypto.ParseCertificatesPEM([]byte(credential.Value))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse EBICS credential certificate %q: %w", credential.Name, err)
	}

	pub, ok := certs[0].PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%w: %q", errTransferNonRSAKey, credential.Name)
	}

	return priv, certs[0], pub, nil
}

func restrictedServiceFromPayloadService(service *ebicsruntime.PayloadServiceRef) ebicsxml.RestrictedService {
	if service == nil {
		return ebicsxml.RestrictedService{}
	}

	restricted := ebicsxml.RestrictedService{
		ServiceName:   strings.TrimSpace(service.ServiceName),
		Scope:         strings.TrimSpace(service.Scope),
		ServiceOption: strings.TrimSpace(service.ServiceOption),
		MsgName:       ebicsxml.MessageType{Value: strings.TrimSpace(service.MsgName)},
	}

	if containerType := strings.TrimSpace(service.ContainerType); containerType != "" {
		restricted.Container = &ebicsxml.ContainerFlag{ContainerType: containerType}
	}

	return restricted
}

func buildTransferMetadata(info map[string]any) map[string]any {
	metadata := map[string]any{}
	for key, value := range info {
		switch key {
		case "declaredAmount", "declaredCurrency", "correlationId", "correlationID":
			metadata[key] = value
		}
	}

	return metadata
}

func readTransferString(info map[string]any, key string) string {
	if info == nil {
		return ""
	}

	raw, ok := info[key]
	if !ok {
		return ""
	}

	return strings.TrimSpace(fmt.Sprint(raw))
}

func readTransferInt64(info map[string]any, key string) int64 {
	if info == nil {
		return 0
	}

	raw, ok := info[key]
	if !ok {
		return 0
	}

	switch value := raw.(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return 0
		}

		return parsed
	case string:
		parsed, err := utils.ParseInt[int64](strings.TrimSpace(value))
		if err != nil {
			return 0
		}

		return parsed
	default:
		return 0
	}
}

func nullableID(value int64) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: value, Valid: true}
}
