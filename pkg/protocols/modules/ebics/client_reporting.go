package ebics

import (
	"context"
	"errors"
	"fmt"
	"strings"

	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	errUnsupportedReportingOrder = errors.New("unsupported EBICS reporting order")
	errUnsupportedSignatureOrder = errors.New("unsupported EBICS signature order")
	errMissingReportingOrderID   = errors.New("missing EBICS reporting order identifier")
	errMissingSignatureOrderID   = errors.New("missing EBICS signature order identifier")
	errMissingReportingService   = errors.New("missing EBICS reporting service descriptor")
	errMissingSignatureService   = errors.New("missing EBICS signature service descriptor")
	errMissingHVSOrderData       = errors.New("missing HVS order data")
	errMissingHVSSignatureData   = errors.New("missing HVS signature data")
)

const (
	reportingOrderHVE = "HVE"
	signatureOrderHVS = "HVS"
)

// ReportingActionInput defines one client-side reporting/admin retrieval request.
type ReportingActionInput struct {
	EbicsSubscriberID int64
	OrderType         string
	OrderID           string
	Service           *ServiceRef
	ServiceFilters    []ServiceRef
	CompleteOrderData bool
	FetchLimit        int
	FetchOffset       int
	Metadata          map[string]any
}

// SignatureActionInput defines one client-side signature request.
type SignatureActionInput struct {
	EbicsSubscriberID int64
	OrderType         string
	OrderID           string
	Service           *ServiceRef
	OrderData         []byte
	SignatureData     []byte
	Metadata          map[string]any
}

// ServiceRef describes one EBICS service or service filter in Gateway runtime.
type ServiceRef struct {
	ServiceName   string
	ServiceOption string
	Scope         string
	MsgName       string
	ContainerType string
}

// ExecuteReportingAction runs one EBICS reporting/admin order through the
// operational Gateway client and persists the resulting EbicsOperation.
func ExecuteReportingAction(
	ctx context.Context,
	db *database.DB,
	input *ReportingActionInput,
) (*model.EbicsOperation, error) {
	service, stop, err := startOperationalClient(ctx, db)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.executeReportingAction(input)
}

// ExecuteSignatureAction runs one EBICS protocol signature order through the
// operational Gateway client and persists the resulting EbicsOperation.
func ExecuteSignatureAction(
	ctx context.Context,
	db *database.DB,
	input *SignatureActionInput,
) (*model.EbicsOperation, error) {
	service, stop, err := startOperationalClient(ctx, db)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.executeSignatureAction(input)
}

func (c *Client) executeReportingAction(input *ReportingActionInput) (*model.EbicsOperation, error) {
	if !c.state.IsRunning() {
		return nil, utils.ErrNotRunning
	}

	orderType := model.NormalizeEbicsOrderType(input.OrderType)
	if err := validateReportingActionInput(orderType, input); err != nil {
		return nil, err
	}

	execCtx, err := c.newAdminExecutionContext(input.EbicsSubscriberID)
	if err != nil {
		return nil, err
	}

	operation, err := c.insertNonPayloadOperation(execCtx, "REPORTING", orderType, "INBOUND")
	if err != nil {
		return nil, err
	}
	attachClientActionMetadata(operation, input.Metadata)

	requestCtx, cancel := context.WithTimeout(context.Background(), c.adminRequestTimeout())
	defer cancel()

	switch orderType {
	case "HVD":
		return c.executeHVD(requestCtx, execCtx, operation, input)
	case "HVU":
		return c.executeHVU(requestCtx, execCtx, operation, input)
	case "HVZ":
		return c.executeHVZ(requestCtx, execCtx, operation, input)
	case "HVT":
		return c.executeHVT(requestCtx, execCtx, operation, input)
	case "HAC":
		return c.executeHAC(requestCtx, execCtx, operation, input)
	default:
		return nil, c.failNonPayloadOperation(
			operation,
			"execute reporting order",
			fmt.Errorf("%w: %s", errUnsupportedReportingOrder, orderType),
		)
	}
}

func (c *Client) executeSignatureAction(input *SignatureActionInput) (*model.EbicsOperation, error) {
	if !c.state.IsRunning() {
		return nil, utils.ErrNotRunning
	}

	orderType := model.NormalizeEbicsOrderType(input.OrderType)
	if err := validateSignatureActionInput(orderType, input); err != nil {
		return nil, err
	}

	execCtx, err := c.newAdminExecutionContext(input.EbicsSubscriberID)
	if err != nil {
		return nil, err
	}

	direction := "INBOUND"
	if orderType == signatureOrderHVS {
		direction = "OUTBOUND"
	}

	operation, err := c.insertNonPayloadOperation(execCtx, "SIGNATURE", orderType, direction)
	if err != nil {
		return nil, err
	}
	attachClientActionMetadata(operation, input.Metadata)

	requestCtx, cancel := context.WithTimeout(context.Background(), c.adminRequestTimeout())
	defer cancel()

	switch orderType {
	case reportingOrderHVE:
		return c.executeHVE(requestCtx, execCtx, operation, input)
	case signatureOrderHVS:
		return c.executeHVS(requestCtx, execCtx, operation, input)
	default:
		return nil, c.failNonPayloadOperation(
			operation,
			"execute signature order",
			fmt.Errorf("%w: %s", errUnsupportedSignatureOrder, orderType),
		)
	}
}

func validateReportingActionInput(orderType string, input *ReportingActionInput) error {
	switch orderType {
	case "HVD", "HVT":
		if strings.TrimSpace(input.OrderID) == "" {
			return fmt.Errorf("%w for %s", errMissingReportingOrderID, orderType)
		}
		if input.Service == nil {
			return fmt.Errorf("%w for %s", errMissingReportingService, orderType)
		}
	case "HAC":
		if strings.TrimSpace(input.OrderID) == "" {
			return fmt.Errorf("%w for %s", errMissingReportingOrderID, orderType)
		}
	case reportingOrderHVE:
		return fmt.Errorf("%w: %s", errUnsupportedReportingOrder, orderType)
	case "HVU", "HVZ":
		// optional filters only
	default:
		return fmt.Errorf("%w: %s", errUnsupportedReportingOrder, orderType)
	}

	return nil
}

func validateSignatureActionInput(orderType string, input *SignatureActionInput) error {
	switch orderType {
	case reportingOrderHVE:
		if strings.TrimSpace(input.OrderID) == "" {
			return fmt.Errorf("%w for %s", errMissingSignatureOrderID, orderType)
		}
		if input.Service == nil {
			return fmt.Errorf("%w for %s", errMissingSignatureService, orderType)
		}
	case signatureOrderHVS:
		if strings.TrimSpace(input.OrderID) == "" {
			return fmt.Errorf("%w for %s", errMissingSignatureOrderID, orderType)
		}
		if input.Service == nil {
			return fmt.Errorf("%w for %s", errMissingSignatureService, orderType)
		}
		if len(input.OrderData) == 0 {
			return errMissingHVSOrderData
		}
		if len(input.SignatureData) == 0 {
			return errMissingHVSSignatureData
		}
	default:
		return fmt.Errorf("%w: %s", errUnsupportedSignatureOrder, orderType)
	}

	return nil
}

func (c *Client) executeHVD(
	ctx context.Context,
	execCtx *adminExecutionContext,
	operation *model.EbicsOperation,
	input *ReportingActionInput,
) (*model.EbicsOperation, error) {
	params := &ebicsxml.HVDOrderParams{
		HVRequest: ebicsxml.HVRequest{
			PartnerID: execCtx.subscriber.PartnerID,
			Service:   buildRestrictedService(input.Service),
			OrderID:   strings.TrimSpace(input.OrderID),
		},
	}

	doc, response, err := execCtx.libClient.DownloadHVDDocument(ctx, libebicsclient.FlowHVDRequired{
		URL:       execCtx.endpointURL,
		HostID:    execCtx.host.HostID,
		PartnerID: execCtx.subscriber.PartnerID,
		UserID:    execCtx.subscriber.UserID,
		Params:    params,
	}, libebicsclient.FlowHVDOptional{
		RequestSigner:  execCtx.requestSigner,
		ResponseSigner: execCtx.responseSigner,
		Cipher:         execCtx.downloadCipher,
	})
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "download HVD report", err)
	}

	operation.MetadataMap["reporting"] = map[string]any{
		"orderType":              "HVD",
		"orderID":                strings.TrimSpace(input.OrderID),
		"service":                serviceRefToMetadata(input.Service),
		"displayFileSize":        len(doc.DisplayFile),
		"orderDataSize":          doc.OrderDataSize,
		"orderDataAvailable":     doc.OrderDataAvailable,
		"orderDetailsAvailable":  doc.OrderDetailsAvailable,
		"bankSignatureSize":      len(doc.BankSignature),
		"signaturesAvailable":    len(doc.SignerInfo),
		"signatureReady":         len(doc.SignerInfo) > 0,
		"dataDigestVersion":      strings.TrimSpace(doc.DataDigest.SignatureVersion),
		"dataDigestEncodedValue": string(doc.DataDigest.Value),
	}

	technicalCode, businessCode := extractResponseReturnCodes(response)
	if completeErr := c.completeNonPayloadOperation(operation, technicalCode, businessCode); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func (c *Client) executeHVU(
	ctx context.Context,
	execCtx *adminExecutionContext,
	operation *model.EbicsOperation,
	input *ReportingActionInput,
) (*model.EbicsOperation, error) {
	doc, response, err := execCtx.libClient.DownloadHVUDocument(ctx, libebicsclient.FlowHVURequired{
		URL:       execCtx.endpointURL,
		HostID:    execCtx.host.HostID,
		PartnerID: execCtx.subscriber.PartnerID,
		UserID:    execCtx.subscriber.UserID,
	}, libebicsclient.FlowHVUOptional{
		Params:         buildHVUOrderParams(input.ServiceFilters),
		RequestSigner:  execCtx.requestSigner,
		ResponseSigner: execCtx.responseSigner,
		Cipher:         execCtx.downloadCipher,
	})
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "download HVU report", err)
	}

	readyCount, expectedCount, doneCount := summarizeHVUSigning(doc.OrderDetails)
	operation.MetadataMap["reporting"] = map[string]any{
		"orderType":           "HVU",
		"serviceFilters":      serviceRefsToMetadata(input.ServiceFilters),
		"orderCount":          len(doc.OrderDetails),
		"signaturesExpected":  expectedCount,
		"signaturesAvailable": doneCount,
		"readyOrderCount":     readyCount,
		"signatureState":      deriveAggregatedSignatureState(readyCount, expectedCount, doneCount),
	}

	technicalCode, businessCode := extractResponseReturnCodes(response)
	if completeErr := c.completeNonPayloadOperation(operation, technicalCode, businessCode); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func (c *Client) executeHVZ(
	ctx context.Context,
	execCtx *adminExecutionContext,
	operation *model.EbicsOperation,
	input *ReportingActionInput,
) (*model.EbicsOperation, error) {
	doc, response, err := execCtx.libClient.DownloadHVZDocument(ctx, libebicsclient.FlowHVZRequired{
		URL:       execCtx.endpointURL,
		HostID:    execCtx.host.HostID,
		PartnerID: execCtx.subscriber.PartnerID,
		UserID:    execCtx.subscriber.UserID,
	}, libebicsclient.FlowHVZOptional{
		Params:         buildHVZOrderParams(input.ServiceFilters),
		RequestSigner:  execCtx.requestSigner,
		ResponseSigner: execCtx.responseSigner,
		Cipher:         execCtx.downloadCipher,
	})
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "download HVZ report", err)
	}

	readyCount, expectedCount, doneCount, totalAmount, currencies := summarizeHVZSigning(doc.OrderDetails)
	operation.MetadataMap["reporting"] = map[string]any{
		"orderType":           "HVZ",
		"serviceFilters":      serviceRefsToMetadata(input.ServiceFilters),
		"orderCount":          len(doc.OrderDetails),
		"signaturesExpected":  expectedCount,
		"signaturesAvailable": doneCount,
		"readyOrderCount":     readyCount,
		"signatureState":      deriveAggregatedSignatureState(readyCount, expectedCount, doneCount),
		"totalAmountValues":   totalAmount,
		"currencies":          currencies,
	}

	technicalCode, businessCode := extractResponseReturnCodes(response)
	if completeErr := c.completeNonPayloadOperation(operation, technicalCode, businessCode); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func (c *Client) executeHVT(
	ctx context.Context,
	execCtx *adminExecutionContext,
	operation *model.EbicsOperation,
	input *ReportingActionInput,
) (*model.EbicsOperation, error) {
	params := &ebicsxml.HVTOrderParams{
		HVRequest: ebicsxml.HVRequest{
			PartnerID: execCtx.subscriber.PartnerID,
			Service:   buildRestrictedService(input.Service),
			OrderID:   strings.TrimSpace(input.OrderID),
		},
		OrderFlags: ebicsxml.HVTOrderFlags{
			CompleteOrderData: input.CompleteOrderData,
			FetchLimit:        input.FetchLimit,
			FetchOffset:       input.FetchOffset,
		},
	}

	if input.CompleteOrderData {
		payload, response, err := execCtx.libClient.DownloadHVTOriginal(ctx, libebicsclient.FlowHVTRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
			Params:    params,
		}, libebicsclient.FlowHVTOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		})
		if err != nil {
			return nil, c.failNonPayloadOperation(operation, "download HVT original payload", err)
		}

		operation.MetadataMap["reporting"] = map[string]any{
			"orderType":         "HVT",
			"orderID":           strings.TrimSpace(input.OrderID),
			"service":           serviceRefToMetadata(input.Service),
			"completeOrderData": true,
			"fetchLimit":        input.FetchLimit,
			"fetchOffset":       input.FetchOffset,
			"payloadSize":       len(payload),
		}

		technicalCode, businessCode := extractResponseReturnCodes(response)
		if completeErr := c.completeNonPayloadOperation(operation, technicalCode, businessCode); completeErr != nil {
			return nil, completeErr
		}

		return operation, nil
	}

	doc, response, err := execCtx.libClient.DownloadHVTDetails(ctx, libebicsclient.FlowHVTRequired{
		URL:       execCtx.endpointURL,
		HostID:    execCtx.host.HostID,
		PartnerID: execCtx.subscriber.PartnerID,
		UserID:    execCtx.subscriber.UserID,
		Params:    params,
	}, libebicsclient.FlowHVTOptional{
		RequestSigner:  execCtx.requestSigner,
		ResponseSigner: execCtx.responseSigner,
		Cipher:         execCtx.downloadCipher,
	})
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "download HVT details", err)
	}

	operation.MetadataMap["reporting"] = map[string]any{
		"orderType":          "HVT",
		"orderID":            strings.TrimSpace(input.OrderID),
		"service":            serviceRefToMetadata(input.Service),
		"completeOrderData":  false,
		"fetchLimit":         input.FetchLimit,
		"fetchOffset":        input.FetchOffset,
		"numOrderInfos":      doc.NumOrderInfos,
		"returnedOrderInfos": len(doc.OrderInfo),
	}

	technicalCode, businessCode := extractResponseReturnCodes(response)
	if completeErr := c.completeNonPayloadOperation(operation, technicalCode, businessCode); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func (c *Client) executeHAC(
	ctx context.Context,
	execCtx *adminExecutionContext,
	operation *model.EbicsOperation,
	input *ReportingActionInput,
) (*model.EbicsOperation, error) {
	doc, response, err := execCtx.libClient.DownloadHACDocument(ctx, libebicsclient.FlowHACRequired{
		URL:       execCtx.endpointURL,
		HostID:    execCtx.host.HostID,
		PartnerID: execCtx.subscriber.PartnerID,
		UserID:    execCtx.subscriber.UserID,
		OrderID:   strings.TrimSpace(input.OrderID),
	}, libebicsclient.FlowHACOptional{
		RequestSigner:  execCtx.requestSigner,
		ResponseSigner: execCtx.responseSigner,
		Cipher:         execCtx.downloadCipher,
	})
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "download HAC report", err)
	}

	actions, reasons := summarizeHAC(doc)
	operation.MetadataMap["reporting"] = map[string]any{
		"orderType":    "HAC",
		"orderID":      strings.TrimSpace(input.OrderID),
		"messageID":    strings.TrimSpace(doc.CustomerReport.GroupHeader.MessageID),
		"createdAt":    strings.TrimSpace(doc.CustomerReport.GroupHeader.CreatedAt),
		"stepCount":    len(doc.CustomerReport.OriginalPaymentInfos),
		"actionTypes":  actions,
		"reasonCodes":  reasons,
		"originalMsg":  strings.TrimSpace(doc.CustomerReport.OriginalGroupInfo.OriginalMessageID),
		"originalName": strings.TrimSpace(doc.CustomerReport.OriginalGroupInfo.OriginalMessageName),
	}

	technicalCode, businessCode := extractResponseReturnCodes(response)
	if completeErr := c.completeNonPayloadOperation(operation, technicalCode, businessCode); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func (c *Client) executeHVE(
	ctx context.Context,
	execCtx *adminExecutionContext,
	operation *model.EbicsOperation,
	input *SignatureActionInput,
) (*model.EbicsOperation, error) {
	params := &ebicsxml.HVEOrderParams{
		HVRequest: ebicsxml.HVRequest{
			PartnerID: execCtx.subscriber.PartnerID,
			Service:   buildRestrictedService(input.Service),
			OrderID:   strings.TrimSpace(input.OrderID),
		},
	}

	payload, response, err := execCtx.libClient.DownloadHVE(ctx, libebicsclient.FlowHVERequired{
		URL:       execCtx.endpointURL,
		HostID:    execCtx.host.HostID,
		PartnerID: execCtx.subscriber.PartnerID,
		UserID:    execCtx.subscriber.UserID,
		Params:    params,
	}, libebicsclient.FlowHVEOptional{
		RequestSigner:  execCtx.requestSigner,
		ResponseSigner: execCtx.responseSigner,
		Cipher:         execCtx.downloadCipher,
	})
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "download HVE signatures", err)
	}

	operation.MetadataMap["signature"] = map[string]any{
		"orderType":      "HVE",
		"orderID":        strings.TrimSpace(input.OrderID),
		"service":        serviceRefToMetadata(input.Service),
		"payloadSize":    len(payload),
		"signatureReady": len(payload) > 0,
		"signatureState": ebicsruntime.SignatureStateReady,
	}

	technicalCode, businessCode := extractResponseReturnCodes(response)
	if completeErr := c.completeNonPayloadOperation(operation, technicalCode, businessCode); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func (c *Client) executeHVS(
	ctx context.Context,
	execCtx *adminExecutionContext,
	operation *model.EbicsOperation,
	input *SignatureActionInput,
) (*model.EbicsOperation, error) {
	params := &ebicsxml.HVSOrderParams{
		HVRequest: ebicsxml.HVRequest{
			PartnerID: execCtx.subscriber.PartnerID,
			Service:   buildRestrictedService(input.Service),
			OrderID:   strings.TrimSpace(input.OrderID),
		},
	}

	err := execCtx.libClient.UploadHVS(ctx, libebicsclient.FlowHVSRequired{
		URL:           execCtx.endpointURL,
		HostID:        execCtx.host.HostID,
		PartnerID:     execCtx.subscriber.PartnerID,
		UserID:        execCtx.subscriber.UserID,
		Params:        params,
		OrderData:     input.OrderData,
		SignatureData: input.SignatureData,
	}, libebicsclient.FlowHVSOptional{
		ResponseSigner: execCtx.responseSigner,
	})
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "upload HVS signature", err)
	}

	operation.MetadataMap["signature"] = map[string]any{
		"orderType":         "HVS",
		"orderID":           strings.TrimSpace(input.OrderID),
		"service":           serviceRefToMetadata(input.Service),
		"orderDataSize":     len(input.OrderData),
		"signatureDataSize": len(input.SignatureData),
		"signatureState":    ebicsruntime.SignatureStateAdded,
	}

	if completeErr := c.completeNonPayloadOperation(operation, "", ""); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func attachClientActionMetadata(operation *model.EbicsOperation, metadata map[string]any) {
	if operation == nil || len(metadata) == 0 {
		return
	}

	if operation.MetadataMap == nil {
		operation.MetadataMap = map[string]any{}
	}

	operation.MetadataMap["requestMetadata"] = metadata
}

func buildRestrictedService(service *ServiceRef) ebicsxml.RestrictedService {
	restricted := ebicsxml.RestrictedService{
		ServiceName:   strings.TrimSpace(service.ServiceName),
		Scope:         strings.TrimSpace(service.Scope),
		ServiceOption: strings.TrimSpace(service.ServiceOption),
		MsgName: ebicsxml.MessageType{
			Value: strings.TrimSpace(service.MsgName),
		},
	}

	if containerType := strings.TrimSpace(service.ContainerType); containerType != "" {
		restricted.Container = &ebicsxml.ContainerFlag{ContainerType: containerType}
	}

	return restricted
}

func buildServiceFilter(service *ServiceRef) ebicsxml.ServiceFilter {
	filter := ebicsxml.ServiceFilter{
		ServiceName:   strings.TrimSpace(service.ServiceName),
		Scope:         strings.TrimSpace(service.Scope),
		ServiceOption: strings.TrimSpace(service.ServiceOption),
	}

	if msgName := strings.TrimSpace(service.MsgName); msgName != "" {
		filter.MsgName = &ebicsxml.MessageType{Value: msgName}
	}
	if containerType := strings.TrimSpace(service.ContainerType); containerType != "" {
		filter.Container = &ebicsxml.ContainerFlag{ContainerType: containerType}
	}

	return filter
}

func buildHVUOrderParams(filters []ServiceRef) *ebicsxml.HVUOrderParams {
	if len(filters) == 0 {
		return nil
	}

	params := &ebicsxml.HVUOrderParams{
		ServiceFilter: make([]ebicsxml.ServiceFilter, 0, len(filters)),
	}
	for idx := range filters {
		params.ServiceFilter = append(params.ServiceFilter, buildServiceFilter(&filters[idx]))
	}

	return params
}

func buildHVZOrderParams(filters []ServiceRef) *ebicsxml.HVZOrderParams {
	if len(filters) == 0 {
		return nil
	}

	params := &ebicsxml.HVZOrderParams{
		ServiceFilter: make([]ebicsxml.ServiceFilter, 0, len(filters)),
	}
	for idx := range filters {
		params.ServiceFilter = append(params.ServiceFilter, buildServiceFilter(&filters[idx]))
	}

	return params
}

func serviceRefToMetadata(service *ServiceRef) map[string]any {
	return map[string]any{
		"serviceName":   strings.TrimSpace(service.ServiceName),
		"serviceOption": strings.TrimSpace(service.ServiceOption),
		"scope":         strings.TrimSpace(service.Scope),
		"msgName":       strings.TrimSpace(service.MsgName),
		"containerType": strings.TrimSpace(service.ContainerType),
	}
}

func serviceRefsToMetadata(services []ServiceRef) []map[string]any {
	if len(services) == 0 {
		return nil
	}

	items := make([]map[string]any, 0, len(services))
	for idx := range services {
		items = append(items, serviceRefToMetadata(&services[idx]))
	}

	return items
}

func summarizeHVUSigning(details []ebicsxml.HVUOrderDetails) (readyCount, expectedCount, doneCount int) {
	for idx := range details {
		detail := &details[idx]
		if detail.SigningInfo.ReadyToBeSigned {
			readyCount++
		}
		expectedCount += detail.SigningInfo.NumSigRequired
		doneCount += detail.SigningInfo.NumSigDone
	}

	return readyCount, expectedCount, doneCount
}

func summarizeHVZSigning(
	details []ebicsxml.HVZOrderDetails,
) (readyCount, expectedCount, doneCount int, totalAmounts, currencies []string) {
	currencySet := map[string]struct{}{}

	for idx := range details {
		detail := &details[idx]
		if detail.SigningInfo.ReadyToBeSigned {
			readyCount++
		}
		expectedCount += detail.SigningInfo.NumSigRequired
		doneCount += detail.SigningInfo.NumSigDone

		if detail.TotalAmount != nil && strings.TrimSpace(detail.TotalAmount.Value) != "" {
			totalAmounts = append(totalAmounts, strings.TrimSpace(detail.TotalAmount.Value))
		}
		if currency := strings.TrimSpace(detail.Currency); currency != "" {
			if _, exists := currencySet[currency]; !exists {
				currencySet[currency] = struct{}{}
				currencies = append(currencies, currency)
			}
		}
	}

	return readyCount, expectedCount, doneCount, totalAmounts, currencies
}

func deriveAggregatedSignatureState(readyCount, expectedCount, doneCount int) string {
	switch {
	case doneCount > 0 && expectedCount > 0 && doneCount >= expectedCount:
		return ebicsruntime.SignatureStateReady
	case doneCount > 0:
		return ebicsruntime.SignatureStatePartiallyAvailable
	case readyCount > 0 || expectedCount > 0:
		return ebicsruntime.SignatureStateWaiting
	default:
		return ebicsruntime.SignatureStateUnknown
	}
}

func summarizeHAC(doc *ebicsxml.HACDocument) (actionTypes, reasonCodes []string) {
	if doc == nil {
		return nil, nil
	}

	actionTypes = make([]string, 0, len(doc.CustomerReport.OriginalPaymentInfos))
	reasonCodes = make([]string, 0, len(doc.CustomerReport.OriginalPaymentInfos))
	for _, step := range doc.CustomerReport.OriginalPaymentInfos {
		actionTypes = append(actionTypes, strings.TrimSpace(step.ActionType))
		for _, reason := range step.StatusReasonInfos {
			if code := strings.TrimSpace(reason.Reason.Code); code != "" {
				reasonCodes = append(reasonCodes, code)
			}
		}
	}

	return actionTypes, reasonCodes
}
