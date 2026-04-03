package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func displayEbicsHistoryEntries(w io.Writer, entries []*api.OutEbicsHistoryEntry) error {
	Style0.PrintV(w, "=== EBICS history ===")
	for _, entry := range entries {
		if err := displayEbicsHistoryEntry(w, entry); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsHistoryEntry(w io.Writer, entry *api.OutEbicsHistoryEntry) error {
	Style1.Printf(w, "EBICS history #%d [%s]", entry.ID, entry.Status)
	Style22.PrintL(w, "History type", entry.HistoryType)
	Style22.PrintL(w, "Operation type", entry.OperationType)
	Style22.Option(w, "Action", entry.Action)
	Style22.Option(w, "Order type", entry.OrderType)
	Style22.Option(w, "Direction", entry.Direction)
	Style22.Option(w, "Transport mode", entry.TransportMode)
	Style22.Option(w, "Host ID", entry.HostID)
	Style22.Option(w, "Partner ID", entry.PartnerID)
	Style22.Option(w, "User ID", entry.UserID)
	Style22.Option(w, "Coordination ID", entry.CoordinationID)
	Style22.Option(w, "Request ID", entry.RequestID)
	Style22.Option(w, "Correlation ID", entry.CorrelationID)
	Style22.Option(w, "Transaction ID", entry.TransactionID)
	Style22.Option(w, "Gateway outcome", entry.GatewayOutcome)
	Style22.Option(w, "Retry decision", entry.RetryDecision)
	Style22.Option(w, "Operator", entry.Operator)
	Style22.Option(w, "Reason", entry.Reason)
	if entry.ClientID != nil {
		Style22.PrintL(w, "Client ID", *entry.ClientID)
	}
	if entry.OperationID != nil {
		Style22.PrintL(w, "Operation ID", *entry.OperationID)
	}
	if entry.TransferID != nil {
		Style22.PrintL(w, "Transfer ID", *entry.TransferID)
	}
	if entry.WorkflowID != nil {
		Style22.PrintL(w, "Workflow ID", *entry.WorkflowID)
	}
	if entry.LifecycleID != nil {
		Style22.PrintL(w, "Lifecycle ID", *entry.LifecycleID)
	}
	if entry.StartedAt != nil {
		Style22.PrintL(w, "Started at", *entry.StartedAt)
	}
	if entry.FinishedAt != nil {
		Style22.PrintL(w, "Finished at", *entry.FinishedAt)
	}
	Style22.PrintL(w, "Created at", entry.CreatedAt)
	if len(entry.Evidence) > 0 {
		Style22.Printf(w, "Evidence:")
		displayMap(w, Style333, entry.Evidence)
	}
	if len(entry.Metadata) > 0 {
		Style22.Printf(w, "Metadata:")
		displayMap(w, Style333, entry.Metadata)
	}

	return nil
}

type EbicsHistoryList struct {
	ListOptions
}

func (c *EbicsHistoryList) Execute([]string) error { return execute(c) }
func (c *EbicsHistoryList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/history"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsHistoryEntry
	if err := list(&body); err != nil {
		return err
	}

	if entries := body["history"]; len(entries) > 0 {
		return outputObject(w, entries, &c.OutputFormat, displayEbicsHistoryEntries)
	}

	fmt.Fprintln(w, "No EBICS history entry found.")

	return nil
}

type EbicsHistoryGet struct {
	OutputFormat

	Args struct {
		History string `required:"yes" positional-arg-name:"history" description:"The EBICS history identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsHistoryGet) Execute([]string) error { return execute(c) }
func (c *EbicsHistoryGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/history", c.Args.History)

	var entry api.OutEbicsHistoryEntry
	if err := get(&entry); err != nil {
		return err
	}

	return outputObject(w, &entry, &c.OutputFormat, displayEbicsHistoryEntry)
}
