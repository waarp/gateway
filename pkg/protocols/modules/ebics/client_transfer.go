package ebics

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
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
	transferInfoKeyEbicsProfileName     = "ebicsProfile"
	transferInfoKeyEbicsEndpointURL     = "ebicsEndpointURL"
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
	errTransferNoActiveLifecycle  = errors.New("no activated EBICS key lifecycle found")
	errTransferAmbiguousLifecycle = errors.New("multiple activated EBICS key lifecycles found")
	errTransferNoValidatedBankKey = errors.New("no validated EBICS bank key exists")
	errTransferAmbiguousBankKey   = errors.New("multiple validated EBICS bank keys exist")
	errTransferMissingCredential  = errors.New("EBICS credential is missing")
	errTransferNonRSAKey          = errors.New("EBICS credential does not contain an RSA public key")
)

type transferClient struct {
	parent   *Client
	pip      *pipeline.Pipeline
	exec     *payloadExecution
	finished bool
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

	payload, _, err := c.exec.client.DownloadBTD(ctx, libebicsclient.FlowBTDRequired{
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
	if err := c.parent.db.Update(c.exec.operation).Run(); err != nil {
		return pipeline.NewErrorWith(types.TeInternal, "persist EBICS operation finalization", err)
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
		transaction,
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
		transactionID:  transaction.TransactionID,
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
	hostID := readTransferString(trans.TransferInfo, transferInfoKeyEbicsHostID)
	partnerID := readTransferString(trans.TransferInfo, transferInfoKeyEbicsPartnerID)
	userID := readTransferString(trans.TransferInfo, transferInfoKeyEbicsUserID)

	if hostID != "" && partnerID != "" && userID != "" {
		var host model.EbicsHost
		if err := c.parent.db.Get(&host, "owner=? AND host_id=?", conf.GlobalConfig.GatewayName, hostID).Run(); err != nil {
			return nil, nil, fmt.Errorf("load EBICS host %q for transfer %d: %w", hostID, trans.ID, err)
		}

		var subscriber model.EbicsSubscriber
		if err := c.parent.db.Get(
			&subscriber,
			"owner=? AND ebics_host_id=? AND partner_id=? AND user_id=?",
			conf.GlobalConfig.GatewayName,
			host.ID,
			partnerID,
			userID,
		).Run(); err != nil {
			return nil, nil, fmt.Errorf("load EBICS subscriber %q/%q on host %q: %w", partnerID, userID, hostID, err)
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
	profileName := readTransferString(c.pip.TransCtx.Transfer.TransferInfo, transferInfoKeyEbicsProfileName)
	if profileName != "" {
		var profile model.EbicsPayloadProfile
		if err := c.parent.db.Get(
			&profile,
			"owner=? AND name=?",
			conf.GlobalConfig.GatewayName,
			profileName,
		).Run(); err != nil {
			return nil, fmt.Errorf("load EBICS payload profile %q: %w", profileName, err)
		}
		if !profile.IsEnabled {
			return nil, fmt.Errorf("%w: %q", errTransferDisabledProfile, profileName)
		}
		return &profile, nil
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
	if endpoint := readTransferString(
		c.pip.TransCtx.Transfer.TransferInfo,
		transferInfoKeyEbicsEndpointURL,
	); endpoint != "" {
		return endpoint, nil
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
	operation, err := ebicsruntime.NewPayloadOperation(&ebicsruntime.OperationMappingInput{
		Owner:             conf.GlobalConfig.GatewayName,
		ClientID:          c.pip.TransCtx.Client.ID,
		RemoteAgentID:     c.pip.TransCtx.RemoteAgent.ID,
		RemoteAccountID:   c.pip.TransCtx.RemoteAccount.ID,
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		OrderType:         resolved.OrderType,
		OperationType:     "REPORTING",
		Direction:         c.operationDirection(),
		TransportMode:     "ASYNC",
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

func (c *transferClient) createTransaction(
	operation *model.EbicsOperation,
) (*model.EbicsTransaction, error) {
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
	transaction *model.EbicsTransaction,
	profile *model.EbicsPayloadProfile,
	endpointURL string,
	resolved *ebicsruntime.ResolvedPayloadRequest,
) error {
	trans := c.pip.TransCtx.Transfer
	if trans.TransferInfo == nil {
		trans.TransferInfo = map[string]any{}
	}

	enrichTransferInfo(trans, operation, resolved)
	trans.TransferInfo[transferInfoKeyEbicsProfileName] = profile.Name
	trans.TransferInfo[transferInfoKeyEbicsEndpointURL] = endpointURL
	trans.TransferInfo["ebicsTransactionID"] = transaction.TransactionID

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
	if value := readTransferString(c.pip.TransCtx.Transfer.TransferInfo, transferInfoKeyEbicsRequestID); value != "" {
		return value
	}

	return strings.TrimSpace(c.pip.TransCtx.Transfer.RemoteTransferID)
}

func (c *transferClient) resolveCorrelationID() string {
	if value := readTransferString(c.pip.TransCtx.Transfer.TransferInfo, transferInfoKeyEbicsCorrelationID); value != "" {
		return value
	}

	return strings.TrimSpace(c.pip.TransCtx.Transfer.RemoteTransferID)
}

func (c *transferClient) resolveTransactionID() string {
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
	c.exec.transaction.Status = model.EbicsTransactionStatusCompletedForRuntime()
	c.exec.transaction.TotalSize = int64(payloadSize)
	if err := c.parent.db.Update(c.exec.transaction).Run(); err != nil {
		return c.failPipeline(types.TeInternal, "persist successful EBICS transaction", err)
	}

	if err := ebicsruntime.UpdateOperationOutcomeFromReturnCodes(c.exec.operation, "", "", "", ""); err != nil {
		return c.failPipeline(types.TeInternal, "derive successful EBICS outcome", err)
	}

	c.exec.operation.FinishedAt = time.Now().UTC()
	if err := c.parent.db.Update(c.exec.operation).Run(); err != nil {
		return c.failPipeline(types.TeInternal, "persist successful EBICS operation", err)
	}

	c.finished = true

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
		c.exec.operation.Status = model.EbicsOperationStatusFailedForRuntime()
		c.exec.operation.Severity = model.EbicsOperationSeverityErrorForRuntime()
		c.exec.operation.GatewayOutcome = model.EbicsGatewayOutcomeTechnicalFatalFailureForRuntime()
		c.exec.operation.RetryDecision = model.EbicsRetryDecisionNoRetryForRuntime()
		if cause != nil {
			c.exec.operation.TechnicalReturnMessage = strings.TrimSpace(cause.Error())
		} else {
			c.exec.operation.TechnicalReturnMessage = strings.TrimSpace(details)
		}
		c.exec.operation.FinishedAt = time.Now().UTC()
		if err := c.parent.db.Update(c.exec.operation).Run(); err != nil {
			c.parent.logger.Warningf("Failed to persist EBICS operation failure state: %v", err)
		}

		c.exec.transaction.Status = ebicsClientTransactionStatusFailed
		if err := c.parent.db.Update(c.exec.transaction).Run(); err != nil {
			c.parent.logger.Warningf("Failed to persist EBICS transaction failure state: %v", err)
		}

		c.finished = true
	}

	if cause != nil {
		return pipeline.NewErrorWith(code, details, cause)
	}

	return pipeline.NewError(code, details)
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

func nullableID(value int64) sql.NullInt64 {
	if value <= 0 {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: value, Valid: true}
}
