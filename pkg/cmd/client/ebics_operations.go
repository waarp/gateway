package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
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
	Style22.Option(w, "Business return code", operation.BusinessReturnCode)
	Style22.Option(w, "Gateway outcome", operation.GatewayOutcome)
	Style22.Option(w, "Retry decision", operation.RetryDecision)
	Style22.PrintL(w, "Manual action required", operation.ManualActionRequired)
	if operation.TransferID != nil {
		Style22.PrintL(w, "Transfer ID", *operation.TransferID)
	}

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

	var operation api.OutEbicsOperation
	if err := get(&operation); err != nil {
		return err
	}

	return outputObject(w, &operation, &c.OutputFormat, displayEbicsOperation)
}
