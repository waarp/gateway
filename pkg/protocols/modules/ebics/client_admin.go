package ebics

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"
	libebicscrypto "code.waarp.fr/lib/ebics/ebics/crypto"
	libreturncode "code.waarp.fr/lib/ebics/ebics/returncode"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"
	"github.com/google/uuid"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	errMissingOperationalEBICSClientID   = errors.New("the EBICS client identifier is missing")
	errInitializationMissingAccount      = errors.New("the EBICS subscriber has no remote account for client execution")
	errUnsupportedInitializationOrder    = errors.New("unsupported EBICS initialization order")
	errMissingKeyVersion                 = errors.New("missing key version")
	errCredentialWithoutCertificate      = errors.New("credential does not contain an X.509 certificate")
	errOperationalEBICSClientIsDisabled  = errors.New("the EBICS client is disabled")
	errOperationalEBICSClientBadProtocol = errors.New("the selected client does not use the EBICS protocol")
)

const adminClientStopTimeout = 5 * time.Second

type adminExecutionContext struct {
	host           *model.EbicsHost
	subscriber     *model.EbicsSubscriber
	remoteAccount  *model.RemoteAccount
	remoteAgent    *model.RemoteAgent
	endpointURL    string
	libClient      *libebicsclient.Client
	requestSigner  libebicscrypto.Signer
	responseSigner libebicscrypto.Signer
	downloadCipher libebicscrypto.E002Cipher
}

// ExecuteInitializationWorkflowAction runs a real EBICS initialization step
// against the bank for the provided workflow and returns extra evidence that
// can be persisted by the caller.
func ExecuteInitializationWorkflowAction(
	ctx context.Context,
	db *database.DB,
	clientID int64,
	workflow *model.EbicsInitializationWorkflow,
	action string,
) (map[string]any, error) {
	service, stop, err := startOperationalClient(ctx, db, clientID)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.executeInitializationWorkflowAction(workflow, action)
}

// SyncBankKeysForInitialization downloads the bank public keys (HPB) for the
// workflow subscriber and persists them in Gateway.
func SyncBankKeysForInitialization(
	ctx context.Context,
	db *database.DB,
	clientID int64,
	workflow *model.EbicsInitializationWorkflow,
) (*model.EbicsOperation, error) {
	service, stop, err := startOperationalClient(ctx, db, clientID)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.syncBankKeysForSubscriber(workflow.EbicsSubscriberID)
}

func startOperationalClient(parentCtx context.Context, db *database.DB, clientID int64) (*Client, func(), error) {
	dbClient, err := resolveOperationalClientModel(db, clientID)
	if err != nil {
		return nil, nil, err
	}

	service := NewClient(db, dbClient)
	if err = service.Start(); err != nil {
		return nil, nil, fmt.Errorf("start EBICS operational client %q: %w", dbClient.Name, err)
	}

	stop := func() {
		stopCtx, cancel := context.WithTimeout(parentCtx, adminClientStopTimeout)
		defer cancel()

		if stopErr := service.Stop(stopCtx); stopErr != nil && service.logger != nil {
			service.logger.Warningf("Failed to stop temporary EBICS operational client: %v", stopErr)
		}
	}

	return service, stop, nil
}

func resolveOperationalClientModel(db *database.DB, clientID int64) (*model.Client, error) {
	if clientID == 0 {
		return nil, database.NewValidationError(errMissingOperationalEBICSClientID.Error())
	}

	client := &model.Client{}
	if err := db.Get(client, "id=?", clientID).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, database.NewValidationErrorf("the EBICS client %d does not exist", clientID)
		}

		return nil, fmt.Errorf("load EBICS client %d: %w", clientID, err)
	}

	if client.Protocol != "ebics" {
		return nil, database.NewValidationErrorf("%s: clientID=%d protocol=%s",
			errOperationalEBICSClientBadProtocol, clientID, client.Protocol)
	}

	if client.Disabled {
		return nil, database.NewValidationErrorf("%s: clientID=%d", errOperationalEBICSClientIsDisabled, clientID)
	}

	return client, nil
}

func (c *Client) executeInitializationWorkflowAction(
	workflow *model.EbicsInitializationWorkflow,
	action string,
) (map[string]any, error) {
	if !c.state.IsRunning() {
		return nil, utils.ErrNotRunning
	}

	execCtx, err := c.newAdminExecutionContext(workflow.EbicsSubscriberID)
	if err != nil {
		return nil, err
	}

	switch action {
	case "SEND_INI":
		op, execErr := c.executeInitializationOrder(execCtx, "INI")
		if execErr != nil {
			return nil, execErr
		}

		workflow.IniOperationID = nullableID(op.ID)

		return map[string]any{}, nil
	case "SEND_HIA":
		op, execErr := c.executeInitializationOrder(execCtx, "HIA")
		if execErr != nil {
			return nil, execErr
		}

		workflow.HiaOperationID = nullableID(op.ID)

		return map[string]any{}, nil
	case "SEND_H3K":
		op, execErr := c.executeInitializationOrder(execCtx, "H3K")
		if execErr != nil {
			return nil, execErr
		}

		workflow.H3KOperationID = nullableID(op.ID)

		evidence := map[string]any{}
		if letter, letterErr := c.renderH3KLetter(execCtx.host, execCtx.subscriber); letterErr == nil &&
			strings.TrimSpace(letter) != "" {
			evidence["h3kLetter"] = letter
		}

		return evidence, nil
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedInitializationOrder, action)
	}
}

func (c *Client) executeInitializationOrder(
	execCtx *adminExecutionContext,
	orderType string,
) (*model.EbicsOperation, error) {
	operation, err := c.insertNonPayloadOperation(execCtx, "INITIALIZATION", orderType, "OUTBOUND")
	if err != nil {
		return nil, err
	}

	requestCtx, cancel := context.WithTimeout(context.Background(), c.adminRequestTimeout())
	defer cancel()

	orderData, err := c.buildInitializationOrderData(execCtx.subscriber, orderType)
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "build initialization order data", err)
	}

	var execErr error
	switch orderType {
	case "INI":
		execErr = execCtx.libClient.UploadINI(requestCtx, libebicsclient.FlowINIRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		}, libebicsclient.FlowKeyMgmtOptional{}, orderData)
	case "HIA":
		execErr = execCtx.libClient.UploadHIA(requestCtx, libebicsclient.FlowHIARequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		}, libebicsclient.FlowKeyMgmtOptional{}, orderData)
	case "H3K":
		execErr = execCtx.libClient.UploadH3K(requestCtx, libebicsclient.FlowH3KRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		}, libebicsclient.FlowKeyMgmtOptional{}, orderData)
	default:
		execErr = errUnsupportedInitializationOrder
	}

	if execErr != nil {
		return nil, c.failNonPayloadOperation(operation, "execute initialization order", execErr)
	}

	if completeErr := c.completeNonPayloadOperation(operation, "", ""); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func (c *Client) syncBankKeysForSubscriber(subscriberID int64) (*model.EbicsOperation, error) {
	if !c.state.IsRunning() {
		return nil, utils.ErrNotRunning
	}

	execCtx, err := c.newAdminExecutionContext(subscriberID)
	if err != nil {
		return nil, err
	}

	operation, err := c.insertNonPayloadOperation(execCtx, "KEY_MANAGEMENT", "HPB", "INBOUND")
	if err != nil {
		return nil, err
	}

	requestCtx, cancel := context.WithTimeout(context.Background(), c.adminRequestTimeout())
	defer cancel()

	payload, _, execErr := execCtx.libClient.DownloadHPB(requestCtx, libebicsclient.FlowHPBRequired{
		URL:       execCtx.endpointURL,
		HostID:    execCtx.host.HostID,
		PartnerID: execCtx.subscriber.PartnerID,
		UserID:    execCtx.subscriber.UserID,
	}, libebicsclient.FlowHPBOptional{
		ResponseSigner: execCtx.responseSigner,
		Cipher:         execCtx.downloadCipher,
	})
	if execErr != nil {
		return nil, c.failNonPayloadOperation(operation, "download HPB bank keys", execErr)
	}

	if err = c.persistBankKeys(execCtx.host, payload); err != nil {
		return nil, c.failNonPayloadOperation(operation, "persist HPB bank keys", err)
	}

	operation.MetadataMap["bankKeySync"] = map[string]any{
		"payloadSize": len(payload),
	}

	if completeErr := c.completeNonPayloadOperation(operation, "", ""); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func (c *Client) newAdminExecutionContext(subscriberID int64) (*adminExecutionContext, error) {
	subscriber := &model.EbicsSubscriber{}
	if err := c.db.Get(subscriber, "id=?", subscriberID).Owner().Run(); err != nil {
		return nil, fmt.Errorf("load EBICS subscriber %d for client execution: %w", subscriberID, err)
	}

	if !subscriber.RemoteAccountID.Valid {
		return nil, errInitializationMissingAccount
	}

	host := &model.EbicsHost{}
	if err := c.db.Get(host, "id=?", subscriber.EbicsHostID).Run(); err != nil {
		return nil, fmt.Errorf("load EBICS host %d for subscriber %d: %w", subscriber.EbicsHostID, subscriberID, err)
	}

	remoteAccount := &model.RemoteAccount{}
	if err := c.db.Get(remoteAccount, "id=?", subscriber.RemoteAccountID.Int64).Run(); err != nil {
		return nil, fmt.Errorf("load remote account %d for subscriber %d: %w",
			subscriber.RemoteAccountID.Int64, subscriberID, err)
	}

	remoteAgent := &model.RemoteAgent{}
	if err := c.db.Get(remoteAgent, "id=?", remoteAccount.RemoteAgentID).Run(); err != nil {
		return nil, fmt.Errorf("load remote agent %d for subscriber %d: %w",
			remoteAccount.RemoteAgentID, subscriberID, err)
	}

	endpointURL, err := c.resolveAdminEndpointURL(host, subscriber, remoteAgent)
	if err != nil {
		return nil, err
	}

	httpClient, err := c.buildAdminHTTPClient(endpointURL, remoteAgent, remoteAccount)
	if err != nil {
		return nil, err
	}

	libClient, err := libebicsclient.New(libebicsclient.RequiredConfig{Transport: httpClient})
	if err != nil {
		return nil, fmt.Errorf("initialize lib-ebics admin client: %w", err)
	}

	responseSigner, err := c.resolveAdminResponseSigner(host.ID)
	if err != nil {
		return nil, err
	}

	requestSigner, err := c.resolveAdminRequestSigner(subscriber.ID)
	if err != nil {
		return nil, err
	}

	downloadCipher, err := c.resolveAdminDownloadCipher(subscriber.ID)
	if err != nil {
		return nil, err
	}

	return &adminExecutionContext{
		host:           host,
		subscriber:     subscriber,
		remoteAccount:  remoteAccount,
		remoteAgent:    remoteAgent,
		endpointURL:    endpointURL,
		libClient:      libClient,
		requestSigner:  requestSigner,
		responseSigner: responseSigner,
		downloadCipher: downloadCipher,
	}, nil
}

func (c *Client) resolveAdminEndpointURL(
	host *model.EbicsHost,
	subscriber *model.EbicsSubscriber,
	remoteAgent *model.RemoteAgent,
) (string, error) {
	if subscriber.TransportURL != "" {
		return subscriber.TransportURL, nil
	}

	partnerConf := defaultPartnerConfig()
	if err := utils.JSONConvert(remoteAgent.ProtoConfig, partnerConf); err == nil {
		if endpoint := strings.TrimSpace(partnerConf.EndpointURL); endpoint != "" {
			return endpoint, nil
		}
	}

	if host.DefaultBankURL != "" {
		return host.DefaultBankURL, nil
	}
	if c.config != nil && c.config.EndpointURL != "" {
		return c.config.EndpointURL, nil
	}

	return "", errTransferNoEndpointURL
}

func (c *Client) buildAdminHTTPClient(
	endpointURL string,
	remoteAgent *model.RemoteAgent,
	remoteAccount *model.RemoteAccount,
) (*http.Client, error) {
	parsedURL, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("parse EBICS endpoint URL %q: %w", endpointURL, err)
	}

	serverName := parsedURL.Hostname()
	if serverName == "" {
		return nil, fmt.Errorf("%w: %q", errMissingEndpointServerName, endpointURL)
	}

	rootCAs := utils.TLSCertPool()

	agentCreds, err := remoteAgent.GetCredentials(c.db)
	if err != nil {
		return nil, fmt.Errorf("load remote agent credentials for EBICS admin execution: %w", err)
	}

	for _, cred := range agentCreds {
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
	if err = auth.AddTLSAuthorities(c.db, globalTLSConf); err != nil {
		return nil, fmt.Errorf("load TLS authorities for EBICS admin client: %w", err)
	}

	accountCreds, err := remoteAccount.GetCredentials(c.db)
	if err != nil {
		return nil, fmt.Errorf("load remote account credentials for EBICS admin execution: %w", err)
	}

	certs := make([]tls.Certificate, 0, len(accountCreds))
	for _, cred := range accountCreds {
		if cred.Type != auth.TLSCertificate {
			continue
		}

		cert, parseErr := utils.X509KeyPair(cred.Value, cred.Value2)
		if parseErr != nil {
			return nil, fmt.Errorf("parse TLS client certificate %q for EBICS admin execution: %w", cred.Name, parseErr)
		}

		certs = append(certs, cert)
	}

	httpClient, err := libebicsclient.NewProductionHTTPClient(
		libebicsclient.ProductionHTTPClientRequired{ServerName: serverName},
		libebicsclient.ProductionHTTPClientOptional{
			RootCAs:        rootCAs,
			ClientCerts:    certs,
			MinTLSVersion:  c.config.MinTLSVersion.TLS(),
			RequestTimeout: c.adminRequestTimeout(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("build EBICS HTTP transport for admin execution on %q: %w", endpointURL, err)
	}

	return httpClient, nil
}

func (c *Client) resolveAdminResponseSigner(hostID int64) (libebicscrypto.Signer, error) {
	if !c.config.VerifyBankKeys {
		return nil, nil //nolint:nilnil // bank-key verification is intentionally disabled
	}

	var keys model.EbicsBankKeys
	if err := c.db.Select(&keys).Owner().Where(
		"ebics_host_id=? AND key_type=? AND state=?",
		hostID,
		"SIGNATURE",
		"validated",
	).Run(); err != nil {
		return nil, fmt.Errorf("load validated EBICS bank signature key for host %d: %w", hostID, err)
	}

	switch len(keys) {
	case 0:
		return nil, nil //nolint:nilnil // initialization may happen before bank signature keys are known
	case 1:
	default:
		return nil, fmt.Errorf("%w: hostID=%d keyType=%q", errTransferAmbiguousBankKey, hostID, "SIGNATURE")
	}

	certs, err := libebicscrypto.ParseCertificatesPEM([]byte(keys[0].PublicKey))
	if err != nil {
		return nil, fmt.Errorf("parse validated EBICS bank signature certificate for host %d: %w", hostID, err)
	}

	return &libebicscrypto.XMLDSigSigner{
		Certificate:         certs[0],
		RequireReferenceURI: true,
	}, nil
}

func (c *Client) resolveAdminRequestSigner(subscriberID int64) (libebicscrypto.Signer, error) {
	credential, err := c.resolveLifecycleCredential(subscriberID, model.EbicsKeyUsageAuthenticationForRuntime())
	if err != nil {
		return nil, err
	}

	return signerFromCredential(credential)
}

func (c *Client) resolveAdminDownloadCipher(subscriberID int64) (libebicscrypto.E002Cipher, error) {
	credential, err := c.resolveLifecycleCredential(subscriberID, model.EbicsKeyUsageEncryptionForRuntime())
	if err != nil {
		return nil, err
	}

	priv, cert, pub, err := parseCredentialKeyPair(credential)
	if err != nil {
		return nil, err
	}

	return libebicscrypto.NewE002CipherWithKeys(pub, cert, priv), nil
}

func (c *Client) resolveLifecycleCredential(subscriberID int64, usage string) (*model.Credential, error) {
	var lifecycles model.EbicsKeyLifecycles
	if err := c.db.Select(&lifecycles).Owner().Where(
		"ebics_subscriber_id=? AND key_usage=? AND status NOT IN (?, ?, ?)",
		subscriberID,
		usage,
		"RETIRED",
		"CANCELLED",
		"REJECTED",
	).OrderBy("id", false).Run(); err != nil {
		return nil, fmt.Errorf("load EBICS key lifecycle for subscriber %d and usage %q: %w", subscriberID, usage, err)
	}

	if len(lifecycles) == 0 {
		return nil, fmt.Errorf("%w: subscriberID=%d usage=%q", errTransferNoActiveLifecycle, subscriberID, usage)
	}

	var credential model.Credential
	if err := c.db.Get(&credential, "id=?", lifecycles[0].CurrentCredentialID).Run(); err != nil {
		return nil, fmt.Errorf("load current EBICS credential %d for subscriber %d and usage %q: %w",
			lifecycles[0].CurrentCredentialID, subscriberID, usage, err)
	}

	return &credential, nil
}

func (c *Client) insertNonPayloadOperation(
	execCtx *adminExecutionContext,
	operationType,
	orderType,
	direction string,
) (*model.EbicsOperation, error) {
	operation := &model.EbicsOperation{
		ClientID:          nullableID(c.client.ID),
		RemoteAgentID:     nullableID(execCtx.remoteAgent.ID),
		RemoteAccountID:   nullableID(execCtx.remoteAccount.ID),
		EbicsHostID:       execCtx.host.ID,
		EbicsSubscriberID: execCtx.subscriber.ID,
		OperationType:     operationType,
		OrderType:         orderType,
		Direction:         direction,
		TransportMode:     "ASYNC",
		RequestID:         uuid.NewString(),
		CorrelationID:     uuid.NewString(),
		EbicsVersion:      c.config.ProtocolVersion,
		Status:            "RUNNING",
		Severity:          model.EbicsOperationSeverityInfoForRuntime(),
		GatewayOutcome:    model.EbicsGatewayOutcomePendingBankForRuntime(),
		RetryDecision:     model.EbicsRetryDecisionNoRetryForRuntime(),
		StartedAt:         time.Now().UTC(),
		MetadataMap: map[string]any{
			"endpointURL": execCtx.endpointURL,
		},
	}

	if err := c.db.Insert(operation).Run(); err != nil {
		return nil, fmt.Errorf("insert EBICS non-payload operation %q: %w", orderType, err)
	}

	return operation, nil
}

func (c *Client) completeNonPayloadOperation(
	operation *model.EbicsOperation,
	technicalCode,
	businessCode string,
) error {
	if err := ebicsruntime.UpdateOperationOutcomeFromReturnCodes(
		operation,
		technicalCode,
		"",
		businessCode,
		"",
	); err != nil {
		return fmt.Errorf("derive EBICS outcome for operation %d: %w", operation.ID, err)
	}

	operation.FinishedAt = time.Now().UTC()
	if err := c.db.Update(operation).Run(); err != nil {
		return fmt.Errorf("persist successful EBICS non-payload operation %d: %w", operation.ID, err)
	}

	return nil
}

func (c *Client) failNonPayloadOperation(
	operation *model.EbicsOperation,
	phase string,
	cause error,
) error {
	if operation == nil {
		return fmt.Errorf("%s: %w", phase, cause)
	}

	technicalCode, technicalMessage, businessCode, businessMessage := "", "", "", ""
	if code, ok := libreturncode.FromError(cause); ok {
		switch code.Scope {
		case libreturncode.ScopeTechnical:
			technicalCode = string(code.Value)
			technicalMessage = cause.Error()
		case libreturncode.ScopeBusiness:
			businessCode = string(code.Value)
			businessMessage = cause.Error()
		}
	}

	if technicalCode == "" && businessCode == "" {
		technicalMessage = strings.TrimSpace(cause.Error())
	}

	if err := ebicsruntime.UpdateOperationOutcomeFromReturnCodes(
		operation,
		technicalCode,
		technicalMessage,
		businessCode,
		businessMessage,
	); err != nil {
		operation.Status = model.EbicsOperationStatusFailedForRuntime()
		operation.Severity = model.EbicsOperationSeverityErrorForRuntime()
		operation.GatewayOutcome = model.EbicsGatewayOutcomeTechnicalFatalFailureForRuntime()
		operation.RetryDecision = model.EbicsRetryDecisionNoRetryForRuntime()
		operation.TechnicalReturnMessage = strings.TrimSpace(cause.Error())
	}

	operation.FinishedAt = time.Now().UTC()
	if err := c.db.Update(operation).Run(); err != nil {
		return fmt.Errorf("%s: %w (persist operation %d failure: %v)", phase, cause, operation.ID, err)
	}

	return fmt.Errorf("%s: %w", phase, cause)
}

func (c *Client) buildInitializationOrderData(
	subscriber *model.EbicsSubscriber,
	orderType string,
) ([]byte, error) {
	switch orderType {
	case "INI":
		return c.buildINIOrderData(subscriber)
	case "HIA":
		return c.buildHIAOrderData(subscriber)
	case "H3K":
		return c.buildH3KOrderData(subscriber)
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedInitializationOrder, orderType)
	}
}

func (c *Client) buildINIOrderData(subscriber *model.EbicsSubscriber) ([]byte, error) {
	signatureCred, err := c.resolveLifecycleCredential(subscriber.ID, model.EbicsKeyUsageSignatureForRuntime())
	if err != nil {
		return nil, err
	}

	cert, err := certificatePairFromCredential(signatureCred)
	if err != nil {
		return nil, err
	}

	doc := ebicsxml.SignatureKeyRequestOrderData{
		XMLName: xml.Name{Space: ebicsxml.NamespaceH005, Local: "INIRequestOrderData"},
		SignaturePubKeyInfo: ebicsxml.RawElement{
			InnerXML: signatureKeyInnerXML(cert, "A006"),
		},
		PartnerID: subscriber.PartnerID,
		UserID:    subscriber.UserID,
	}

	data, marshalErr := xml.Marshal(doc)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal INI request order data: %w", marshalErr)
	}

	return data, nil
}

func (c *Client) buildHIAOrderData(subscriber *model.EbicsSubscriber) ([]byte, error) {
	authCred, err := c.resolveLifecycleCredential(subscriber.ID, model.EbicsKeyUsageAuthenticationForRuntime())
	if err != nil {
		return nil, err
	}
	encCred, err := c.resolveLifecycleCredential(subscriber.ID, model.EbicsKeyUsageEncryptionForRuntime())
	if err != nil {
		return nil, err
	}

	authCert, err := certificatePairFromCredential(authCred)
	if err != nil {
		return nil, err
	}
	encCert, err := certificatePairFromCredential(encCred)
	if err != nil {
		return nil, err
	}

	doc := ebicsxml.KeyPairRequestOrderData{
		XMLName: xml.Name{Space: ebicsxml.NamespaceH005, Local: "HIARequestOrderData"},
		AuthenticationPubKeyInfo: ebicsxml.RawElement{
			InnerXML: typedKeyInnerXML(authCert, "AuthenticationVersion", "X002"),
		},
		EncryptionPubKeyInfo: ebicsxml.RawElement{
			InnerXML: typedKeyInnerXML(encCert, "EncryptionVersion", "E002"),
		},
		PartnerID: subscriber.PartnerID,
		UserID:    subscriber.UserID,
	}

	data, marshalErr := xml.Marshal(doc)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal HIA request order data: %w", marshalErr)
	}

	return data, nil
}

func (c *Client) buildH3KOrderData(subscriber *model.EbicsSubscriber) ([]byte, error) {
	signCred, err := c.resolveLifecycleCredential(subscriber.ID, model.EbicsKeyUsageSignatureForRuntime())
	if err != nil {
		return nil, err
	}
	authCred, err := c.resolveLifecycleCredential(subscriber.ID, model.EbicsKeyUsageAuthenticationForRuntime())
	if err != nil {
		return nil, err
	}
	encCred, err := c.resolveLifecycleCredential(subscriber.ID, model.EbicsKeyUsageEncryptionForRuntime())
	if err != nil {
		return nil, err
	}

	signCert, err := certificatePairFromCredential(signCred)
	if err != nil {
		return nil, err
	}
	authCert, err := certificatePairFromCredential(authCred)
	if err != nil {
		return nil, err
	}
	encCert, err := certificatePairFromCredential(encCred)
	if err != nil {
		return nil, err
	}

	doc := ebicsxml.H3KRequestOrderData{
		SignatureCertificateInfo: ebicsxml.SignatureCertificateInfo{
			X509Data:         ebicsxml.X509Data{InnerXML: x509CertificateInnerXML(signCert)},
			SignatureVersion: "A006",
		},
		AuthenticationCertificateInfo: ebicsxml.AuthenticationCertificateInfo{
			X509Data:              ebicsxml.X509Data{InnerXML: x509CertificateInnerXML(authCert)},
			AuthenticationVersion: "X002",
		},
		EncryptionCertificateInfo: ebicsxml.EncryptionCertificateInfo{
			X509Data:          ebicsxml.X509Data{InnerXML: x509CertificateInnerXML(encCert)},
			EncryptionVersion: "E002",
		},
		PartnerID: subscriber.PartnerID,
		UserID:    subscriber.UserID,
	}

	data, marshalErr := xml.Marshal(doc)
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal H3K request order data: %w", marshalErr)
	}

	return data, nil
}

func (c *Client) persistBankKeys(host *model.EbicsHost, payload []byte) error {
	doc, err := ebicsxml.ParseHPBResponseOrderData(payload)
	if err != nil {
		return fmt.Errorf("parse HPB response order data: %w", err)
	}

	keys := []struct {
		keyType string
		raw     []byte
	}{
		{keyType: "AUTH", raw: doc.AuthenticationPubKeyInfo},
		{keyType: "ENCRYPT", raw: doc.EncryptionPubKeyInfo},
		{keyType: "SIGNATURE", raw: doc.SignaturePubKeyInfo},
	}

	for _, current := range keys {
		version, versionErr := extractKeyVersion(current.raw)
		if versionErr != nil {
			return fmt.Errorf("extract HPB %s key version: %w", current.keyType, versionErr)
		}

		hash := sha256.Sum256(current.raw)
		row := &model.EbicsBankKey{}
		getErr := c.db.Get(
			row,
			"owner=? AND ebics_host_id=? AND key_type=? AND version=?",
			conf.GlobalConfig.GatewayName,
			host.ID,
			current.keyType,
			version,
		).Run()
		if getErr != nil && !database.IsNotFound(getErr) {
			return fmt.Errorf("load existing EBICS bank key %s/%s: %w", current.keyType, version, getErr)
		}

		if database.IsNotFound(getErr) {
			row = &model.EbicsBankKey{
				EbicsHostID:   host.ID,
				KeyType:       current.keyType,
				Version:       version,
				PublicKey:     string(current.raw),
				PublicKeyHash: fmt.Sprintf("%x", hash[:]),
				State:         "validated",
				ValidFrom:     time.Now().UTC(),
			}
			if err = c.db.Insert(row).Run(); err != nil {
				return fmt.Errorf("insert EBICS bank key %s/%s: %w", current.keyType, version, err)
			}
		} else {
			row.PublicKey = string(current.raw)
			row.PublicKeyHash = fmt.Sprintf("%x", hash[:])
			row.State = "validated"
			if row.ValidFrom.IsZero() {
				row.ValidFrom = time.Now().UTC()
			}
			if err = c.db.Update(row).Run(); err != nil {
				return fmt.Errorf("update EBICS bank key %s/%s: %w", current.keyType, version, err)
			}
		}

		var retired model.EbicsBankKeys
		if err = c.db.Select(&retired).Owner().Where(
			"ebics_host_id=? AND key_type=? AND version<>? AND state<>?",
			host.ID,
			current.keyType,
			version,
			"retired",
		).Run(); err != nil {
			return fmt.Errorf(
				"load older EBICS bank keys for retirement on host %d and type %s: %w",
				host.ID,
				current.keyType,
				err,
			)
		}

		for _, previous := range retired {
			previous.State = "retired"
			if previous.ValidTo.IsZero() {
				previous.ValidTo = time.Now().UTC()
			}
			if err = c.db.Update(previous).Run(); err != nil {
				return fmt.Errorf("retire EBICS bank key %d: %w", previous.ID, err)
			}
		}
	}

	return nil
}

func (c *Client) renderH3KLetter(host *model.EbicsHost, subscriber *model.EbicsSubscriber) (string, error) {
	signCred, err := c.resolveLifecycleCredential(subscriber.ID, model.EbicsKeyUsageSignatureForRuntime())
	if err != nil {
		return "", err
	}
	authCred, err := c.resolveLifecycleCredential(subscriber.ID, model.EbicsKeyUsageAuthenticationForRuntime())
	if err != nil {
		return "", err
	}
	encCred, err := c.resolveLifecycleCredential(subscriber.ID, model.EbicsKeyUsageEncryptionForRuntime())
	if err != nil {
		return "", err
	}

	signCert, err := certificateLeafFromCredential(signCred)
	if err != nil {
		return "", err
	}
	authCert, err := certificateLeafFromCredential(authCred)
	if err != nil {
		return "", err
	}
	encCert, err := certificateLeafFromCredential(encCred)
	if err != nil {
		return "", err
	}

	signLetterCert, err := libebicsclient.BuildLetterCertificate(signCert)
	if err != nil {
		return "", fmt.Errorf("build H3K signature letter certificate: %w", err)
	}
	authLetterCert, err := libebicsclient.BuildLetterCertificate(authCert)
	if err != nil {
		return "", fmt.Errorf("build H3K authentication letter certificate: %w", err)
	}
	encLetterCert, err := libebicsclient.BuildLetterCertificate(encCert)
	if err != nil {
		return "", fmt.Errorf("build H3K encryption letter certificate: %w", err)
	}

	letter, err := libebicsclient.RenderH3KLetter(libebicsclient.LetterH3KData{
		Header: libebicsclient.LetterHeader{
			UserName:  subscriber.Name,
			Date:      time.Now().UTC(),
			HostID:    strings.TrimSpace(host.HostID),
			BankName:  strings.TrimSpace(host.Name),
			UserID:    subscriber.UserID,
			PartnerID: subscriber.PartnerID,
		},
		SignatureVersion:  "A006",
		AuthVersion:       "X002",
		EncryptionVersion: "E002",
		SignatureCert:     signLetterCert,
		AuthCert:          authLetterCert,
		EncryptionCert:    encLetterCert,
		Note:              "H3K generated by Waarp Gateway during EBICS initialization.",
	})
	if err != nil {
		return "", fmt.Errorf("render H3K letter: %w", err)
	}

	return letter, nil
}

func x509CertificateInnerXML(cert *tls.Certificate) string {
	if cert == nil || len(cert.Certificate) == 0 {
		return ""
	}

	return "<ds:X509Certificate>" + base64.StdEncoding.EncodeToString(cert.Certificate[0]) + "</ds:X509Certificate>"
}

func signatureKeyInnerXML(cert *tls.Certificate, version string) string {
	return x509CertificateInnerXML(cert) +
		"<SignatureVersion>" + strings.TrimSpace(version) + "</SignatureVersion>"
}

func typedKeyInnerXML(cert *tls.Certificate, versionElement, version string) string {
	return x509CertificateInnerXML(cert) +
		"<" + versionElement + ">" + strings.TrimSpace(version) + "</" + versionElement + ">"
}

func extractKeyVersion(raw []byte) (string, error) {
	for _, field := range []string{"AuthenticationVersion", "EncryptionVersion", "SignatureVersion"} {
		value, ok, err := ebicsxml.ExtractElement(raw, field)
		if err != nil || !ok {
			continue
		}

		var node string
		if unmarshalErr := xml.Unmarshal(value, &node); unmarshalErr != nil {
			return "", fmt.Errorf("unmarshal key version fragment: %w", unmarshalErr)
		}

		if trimmed := strings.TrimSpace(node); trimmed != "" {
			return trimmed, nil
		}
	}

	return "", errMissingKeyVersion
}

func (c *Client) adminRequestTimeout() time.Duration {
	return time.Duration(c.config.RequestTimeout) * time.Second
}

func certificatePairFromCredential(credential *model.Credential) (*tls.Certificate, error) {
	if credential == nil {
		return nil, errTransferMissingCredential
	}

	pair, err := utils.X509KeyPair(credential.Value, credential.Value2)
	if err != nil {
		return nil, fmt.Errorf("parse certificate pair for credential %q: %w", credential.Name, err)
	}

	return &pair, nil
}

func certificateLeafFromCredential(credential *model.Credential) (*x509.Certificate, error) {
	pair, err := certificatePairFromCredential(credential)
	if err != nil {
		return nil, err
	}

	if len(pair.Certificate) == 0 {
		return nil, fmt.Errorf("%w: %q", errCredentialWithoutCertificate, credential.Name)
	}

	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("parse X.509 certificate for credential %q: %w", credential.Name, err)
	}

	return cert, nil
}
