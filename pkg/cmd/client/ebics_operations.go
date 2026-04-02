package wg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

var (
	errMissingEbicsTransactionPayload = errors.New("missing EBICS transaction payload in response")
	errMissingEbicsOperationPayload   = errors.New("missing EBICS operation payload in response")
)

func displayEbicsOperations(w io.Writer, operations []*api.OutEbicsOperation) error {
	Style0.PrintV(w, "=== EBICS operations ===")
	for _, operation := range operations {
		if err := displayEbicsOperation(w, operation); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsOperation(w io.Writer, operation *api.OutEbicsOperation) error {
	Style1.Printf(w, "EBICS operation #%d [%s]", operation.ID, operation.Status)
	Style22.PrintL(w, "Operation type", operation.OperationType)
	Style22.PrintL(w, "Order type", operation.OrderType)
	Style22.Option(w, "Signature state", operation.SignatureState)
	Style22.PrintL(w, "Direction", operation.Direction)
	Style22.PrintL(w, "Transport mode", operation.TransportMode)
	Style22.PrintL(w, "Severity", operation.Severity)
	Style22.Option(w, "Transaction ID", operation.TransactionID)
	Style22.Option(w, "Request ID", operation.RequestID)
	Style22.Option(w, "Correlation ID", operation.CorrelationID)
	Style22.Option(w, "Technical return code", operation.TechnicalReturnCode)
	Style22.Option(w, "Technical return message", operation.TechnicalReturnMessage)
	Style22.Option(w, "Business return code", operation.BusinessReturnCode)
	Style22.Option(w, "Business return message", operation.BusinessReturnMessage)
	Style22.Option(w, "Gateway outcome", operation.GatewayOutcome)
	Style22.Option(w, "Retry decision", operation.RetryDecision)
	Style22.PrintL(w, "Manual action required", operation.ManualActionRequired)
	if operation.TransferID != nil {
		Style22.PrintL(w, "Transfer ID", *operation.TransferID)
	}

	return nil
}

func displayEbicsOperationDetail(w io.Writer, detail *api.OutEbicsOperationDetail) error {
	if detail == nil || detail.Operation == nil {
		return errMissingEbicsOperationPayload
	}

	if err := displayEbicsOperation(w, detail.Operation); err != nil {
		return err
	}

	Style22.PrintL(w, "Host ID", detail.HostID)
	Style22.Option(w, "Partner ID", detail.PartnerID)
	Style22.Option(w, "User ID", detail.UserID)
	if detail.StartedAt != nil {
		Style22.PrintL(w, "Started at", *detail.StartedAt)
	}
	if detail.FinishedAt != nil {
		Style22.PrintL(w, "Finished at", *detail.FinishedAt)
	}
	if detail.Links != nil {
		if detail.Links.TransferID != nil {
			Style22.PrintL(w, "Transfer ID", *detail.Links.TransferID)
		}
		if detail.Links.ContractViewID != nil {
			Style22.PrintL(w, "Contract view ID", *detail.Links.ContractViewID)
		}
		if detail.Links.RTNEventID != nil {
			Style22.PrintL(w, "RTN event ID", *detail.Links.RTNEventID)
		}
	}
	if detail.Transaction != nil {
		if err := displayEbicsTransaction(w, detail.Transaction); err != nil {
			return err
		}
	}

	return displayEbicsTransactionSegments(w, detail.Segments)
}

func displayEbicsTransactions(w io.Writer, transactions []*api.OutEbicsTransaction) error {
	Style0.PrintV(w, "=== EBICS transactions ===")
	for _, transaction := range transactions {
		if err := displayEbicsTransaction(w, transaction); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsTransaction(w io.Writer, transaction *api.OutEbicsTransaction) error {
	Style1.Printf(w, "EBICS transaction #%d [%s]", transaction.ID, transaction.Status)
	Style22.PrintL(w, "Transaction ID", transaction.TransactionID)
	Style22.PrintL(w, "Order type", transaction.OrderType)
	Style22.PrintL(w, "Direction", transaction.Direction)
	Style22.PrintL(w, "Segments", fmt.Sprintf("%d/%d", transaction.CurrentSegment, transaction.SegmentCount))
	Style22.PrintL(w, "Total size", transaction.TotalSize)
	if transaction.TransferID != nil {
		Style22.PrintL(w, "Transfer ID", *transaction.TransferID)
	}

	return nil
}

func displayEbicsTransactionSegments(w io.Writer, segments []*api.OutEbicsTransactionSegment) error {
	if len(segments) == 0 {
		Style22.PrintL(w, "Segments", none)

		return nil
	}

	Style22.Printf(w, "Segments:")
	for _, segment := range segments {
		Style333.Printf(
			w,
			"#%d [%s] size=%d checksum=%s ref=%s",
			segment.SegmentNumber,
			segment.SegmentStatus,
			segment.PayloadSize,
			withDefault(segment.Checksum, none),
			withDefault(segment.StoredPayloadRef, none),
		)
	}

	return nil
}

func displayEbicsTransactionSegment(w io.Writer, segment *api.OutEbicsTransactionSegment) error {
	Style1.Printf(w, "EBICS transaction segment #%d [%s]", segment.SegmentNumber, segment.SegmentStatus)
	Style22.PrintL(w, "Payload size", segment.PayloadSize)
	Style22.Option(w, "Checksum", segment.Checksum)
	Style22.Option(w, "Stored payload ref", segment.StoredPayloadRef)

	return nil
}

type EbicsOperationList struct {
	ListOptions
}

func (c *EbicsOperationList) Execute([]string) error { return execute(c) }
func (c *EbicsOperationList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/operations"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsOperation
	if err := list(&body); err != nil {
		return err
	}

	if operations := body["operations"]; len(operations) > 0 {
		return outputObject(w, operations, &c.OutputFormat, displayEbicsOperations)
	}

	fmt.Fprintln(w, "No EBICS operation found.")

	return nil
}

type EbicsOperationGet struct {
	OutputFormat

	Args struct {
		Operation string `required:"yes" positional-arg-name:"operation" description:"The EBICS operation identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsOperationGet) Execute([]string) error { return execute(c) }
func (c *EbicsOperationGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/operations", c.Args.Operation)

	var detail api.OutEbicsOperationDetail
	if err := get(&detail); err != nil {
		return err
	}

	return outputObject(w, &detail, &c.OutputFormat, displayEbicsOperationDetail)
}

type EbicsTransactionList struct {
	ListOptions
}

func (c *EbicsTransactionList) Execute([]string) error { return execute(c) }
func (c *EbicsTransactionList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/transactions"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsTransaction
	if err := list(&body); err != nil {
		return err
	}

	if transactions := body["transactions"]; len(transactions) > 0 {
		return outputObject(w, transactions, &c.OutputFormat, displayEbicsTransactions)
	}

	fmt.Fprintln(w, "No EBICS transaction found.")

	return nil
}

type EbicsTransactionGet struct {
	OutputFormat

	Args struct {
		Transaction string `required:"yes" positional-arg-name:"transaction" description:"The EBICS transaction identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsTransactionGet) Execute([]string) error { return execute(c) }
func (c *EbicsTransactionGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/transactions", c.Args.Transaction)

	var detail api.OutEbicsTransactionDetail
	if err := get(&detail); err != nil {
		return err
	}

	if detail.Transaction == nil {
		return errMissingEbicsTransactionPayload
	}

	if err := outputObject(w, detail.Transaction, &c.OutputFormat, displayEbicsTransaction); err != nil {
		return err
	}

	if c.Format == "json" || c.Format == "yaml" {
		return nil
	}

	Style22.PrintL(w, "Host ID", detail.HostID)
	Style22.Option(w, "Partner ID", detail.PartnerID)
	Style22.Option(w, "User ID", detail.UserID)
	Style22.Option(w, "Request ID", detail.RequestID)
	Style22.Option(w, "Correlation ID", detail.CorrelationID)

	return displayEbicsTransactionSegments(w, detail.Segments)
}

type EbicsTransactionSegments struct {
	OutputFormat

	Args struct {
		Transaction string `required:"yes" positional-arg-name:"transaction" description:"The EBICS transaction identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsTransactionSegments) Execute([]string) error { return execute(c) }
func (c *EbicsTransactionSegments) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/transactions", c.Args.Transaction, "segments")

	var body map[string][]*api.OutEbicsTransactionSegment
	if err := get(&body); err != nil {
		return err
	}

	if segments := body["segments"]; len(segments) > 0 {
		return outputObject(w, segments, &c.OutputFormat, displayEbicsTransactionSegments)
	}

	fmt.Fprintln(w, "No EBICS transaction segment found.")

	return nil
}

type EbicsTransactionSegmentGet struct {
	OutputFormat

	Args struct {
		Transaction string `required:"yes" positional-arg-name:"transaction" description:"The EBICS transaction identifier"`
		Segment     int    `required:"yes" positional-arg-name:"segment" description:"The EBICS segment number"`
	} `positional-args:"yes"`
}

func (c *EbicsTransactionSegmentGet) Execute([]string) error { return execute(c) }
func (c *EbicsTransactionSegmentGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/transactions", c.Args.Transaction, "segments", fmt.Sprint(c.Args.Segment))

	var segment api.OutEbicsTransactionSegment
	if err := get(&segment); err != nil {
		return err
	}

	return outputObject(w, &segment, &c.OutputFormat, displayEbicsTransactionSegment)
}

//nolint:lll // CLI tags are intentionally explicit
type EbicsOperationReporting struct {
	OutputFormat

	ClientID          int64              `required:"yes" long:"client-id" description:"The Gateway EBICS client identifier"`
	EbicsSubscriberID int64              `required:"yes" long:"subscriber" description:"The EBICS subscriber identifier"`
	OrderType         string             `required:"yes" long:"order-type" choice:"HVD" choice:"HVU" choice:"HVZ" choice:"HVT" choice:"HAC" description:"The reporting order type"`
	OrderID           string             `long:"order-id" description:"The target EBICS order identifier"`
	ServiceName       string             `long:"service-name" description:"The EBICS service name"`
	ServiceOption     string             `long:"service-option" description:"The EBICS service option"`
	Scope             string             `long:"scope" description:"The EBICS service scope"`
	MsgName           string             `long:"msg-name" description:"The EBICS message name"`
	ContainerType     string             `long:"container-type" description:"The EBICS container type"`
	CompleteOrderData bool               `long:"complete-order-data" description:"Request the original order payload for HVT"`
	FetchLimit        int                `long:"fetch-limit" description:"The HVT fetch limit"`
	FetchOffset       int                `long:"fetch-offset" description:"The HVT fetch offset"`
	Metadata          map[string]confVal `long:"metadata" description:"Structured metadata in key:value format. Can be repeated."`
}

func (c *EbicsOperationReporting) Execute([]string) error { return execute(c) }
func (c *EbicsOperationReporting) execute(w io.Writer) error {
	addr.Path = "/api/ebics/operations/actions/reporting"

	req := api.InEbicsReportingAction{
		ClientID:          c.ClientID,
		EbicsSubscriberID: c.EbicsSubscriberID,
		OrderType:         c.OrderType,
		OrderID:           c.OrderID,
		CompleteOrderData: c.CompleteOrderData,
		FetchLimit:        c.FetchLimit,
		FetchOffset:       c.FetchOffset,
		Metadata:          normalizeConfMap(c.Metadata),
	}
	if service := cliServiceRef(c.ServiceName, c.ServiceOption, c.Scope, c.MsgName, c.ContainerType); service != nil {
		switch c.OrderType {
		case "HVU", "HVZ":
			req.ServiceFilters = []*api.InEbicsServiceRef{service}
		default:
			req.Service = service
		}
	}

	var operation api.OutEbicsOperation
	if err := postEbicsOperationAction(req, &operation); err != nil {
		return err
	}

	return outputObject(w, &operation, &c.OutputFormat, displayEbicsOperation)
}

//nolint:lll // CLI tags are intentionally explicit
type EbicsOperationSignature struct {
	OutputFormat

	ClientID          int64              `required:"yes" long:"client-id" description:"The Gateway EBICS client identifier"`
	EbicsSubscriberID int64              `required:"yes" long:"subscriber" description:"The EBICS subscriber identifier"`
	OrderType         string             `required:"yes" long:"order-type" choice:"HVE" choice:"HVS" description:"The signature order type"`
	OrderID           string             `required:"yes" long:"order-id" description:"The target EBICS order identifier"`
	ServiceName       string             `long:"service-name" description:"The EBICS service name"`
	ServiceOption     string             `long:"service-option" description:"The EBICS service option"`
	Scope             string             `long:"scope" description:"The EBICS service scope"`
	MsgName           string             `long:"msg-name" description:"The EBICS message name"`
	ContainerType     string             `long:"container-type" description:"The EBICS container type"`
	OrderDataPath     string             `long:"order-data" description:"Path to the HVS order data file"`
	SignatureDataPath string             `long:"signature-data" description:"Path to the HVS signature data file"`
	Metadata          map[string]confVal `long:"metadata" description:"Structured metadata in key:value format. Can be repeated."`
}

func (c *EbicsOperationSignature) Execute([]string) error { return execute(c) }
func (c *EbicsOperationSignature) execute(w io.Writer) error {
	addr.Path = "/api/ebics/operations/actions/signature"

	req := api.InEbicsSignatureAction{
		ClientID:          c.ClientID,
		EbicsSubscriberID: c.EbicsSubscriberID,
		OrderType:         c.OrderType,
		OrderID:           c.OrderID,
		Metadata:          normalizeConfMap(c.Metadata),
	}
	if service := cliServiceRef(c.ServiceName, c.ServiceOption, c.Scope, c.MsgName, c.ContainerType); service != nil {
		req.Service = service
	}

	if c.OrderDataPath != "" {
		data, err := os.ReadFile(c.OrderDataPath)
		if err != nil {
			return fmt.Errorf("read HVS order data %q: %w", c.OrderDataPath, err)
		}
		req.OrderData = data
	}
	if c.SignatureDataPath != "" {
		data, err := os.ReadFile(c.SignatureDataPath)
		if err != nil {
			return fmt.Errorf("read HVS signature data %q: %w", c.SignatureDataPath, err)
		}
		req.SignatureData = data
	}

	var operation api.OutEbicsOperation
	if err := postEbicsOperationAction(req, &operation); err != nil {
		return err
	}

	return outputObject(w, &operation, &c.OutputFormat, displayEbicsOperation)
}

func cliServiceRef(serviceName, serviceOption, scope, msgName, containerType string) *api.InEbicsServiceRef {
	if serviceName == "" && serviceOption == "" && scope == "" && msgName == "" && containerType == "" {
		return nil
	}

	return &api.InEbicsServiceRef{
		ServiceName:   serviceName,
		ServiceOption: serviceOption,
		Scope:         scope,
		MsgName:       msgName,
		ContainerType: containerType,
	}
}

func postEbicsOperationAction(request, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, reqErr := sendRequest(ctx, request, http.MethodPost)
	if reqErr != nil {
		return reqErr
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle

	switch resp.StatusCode {
	case http.StatusCreated:
		return unmarshalBody(resp.Body, out)
	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status, getResponseErrorMessage(resp))
	}
}

func normalizeConfMap(input map[string]confVal) map[string]any {
	if len(input) == 0 {
		return nil
	}

	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = string(value)
	}

	return out
}
