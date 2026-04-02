package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func displayEbicsRTNEvents(w io.Writer, events []*api.OutEbicsRTNEvent) error {
	Style0.PrintV(w, "=== EBICS RTN events ===")
	for _, event := range events {
		if err := displayEbicsRTNEvent(w, event); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsRTNEvent(w io.Writer, event *api.OutEbicsRTNEvent) error {
	Style1.Printf(w, "RTN event #%d [%s]", event.ID, event.Status)
	Style22.PrintL(w, "Source", event.Source)
	Style22.Option(w, "Event ID", event.EventID)
	Style22.Option(w, "Correlation ID", event.CorrelationID)
	Style22.PrintL(w, "Idempotence key", event.IdempotenceKey)
	Style22.Option(w, "Order type hint", event.OrderTypeHint)
	Style22.Option(w, "Profile ID", event.ProfileID)
	Style22.PrintL(w, "Attempts", event.Attempts)
	Style22.PrintL(w, "Received at", event.ReceivedAt)
	if event.NextRetryAt != nil {
		Style22.PrintL(w, "Next retry at", *event.NextRetryAt)
	}
	if event.ProcessedAt != nil {
		Style22.PrintL(w, "Processed at", *event.ProcessedAt)
	}
	Style22.Option(w, "Last error", event.LastError)
	if event.AutoPullOperationID != nil {
		Style22.PrintL(w, "Auto-pull operation ID", *event.AutoPullOperationID)
	}
	if event.AutoPullTransferID != nil {
		Style22.PrintL(w, "Auto-pull transfer ID", *event.AutoPullTransferID)
	}
	Style22.Option(w, "Auto-pull order type", event.AutoPullOrderType)
	Style22.Option(w, "Auto-pull status", event.AutoPullStatus)
	Style22.Option(w, "Auto-pull outcome", event.AutoPullOutcome)
	Style22.Option(w, "Auto-pull retry", event.AutoPullRetry)
	Style22.Option(w, "Operator action", event.OperatorAction)
	Style22.Option(w, "Operator reason", event.OperatorReason)
	if len(event.OperatorMetadata) > 0 {
		Style22.Printf(w, "Operator metadata:")
		displayMap(w, Style333, event.OperatorMetadata)
	}

	return nil
}

func displayEbicsRTNProviders(w io.Writer, providers []*api.OutEbicsRTNProvider) error {
	Style0.PrintV(w, "=== EBICS RTN providers ===")
	for _, provider := range providers {
		if err := displayEbicsRTNProvider(w, provider); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsRTNProvider(w io.Writer, provider *api.OutEbicsRTNProvider) error {
	Style1.Printf(w, "RTN provider %q", provider.Name)
	Style22.PrintL(w, "Transport", provider.Transport)
	Style22.PrintL(w, "Enabled", provider.Enabled)
	Style22.PrintL(w, "Subscriber ID", provider.SubscriberID)
	if provider.ClientID != nil {
		Style22.PrintL(w, "Client ID", *provider.ClientID)
	}
	Style22.PrintL(w, "Auto-pull policy", provider.AutoPullPolicy)
	if provider.LastConnectionAt != nil {
		Style22.PrintL(w, "Last connection", *provider.LastConnectionAt)
	}
	Style22.Option(w, "Last error", provider.LastError)

	return nil
}

//nolint:lll // command tags can be long
type EbicsRTNProviderAdd struct {
	Name           string             `required:"yes" long:"name" description:"The RTN provider name" json:"name,omitempty"`
	Transport      string             `long:"transport" choice:"WSS" default:"WSS" description:"The RTN provider transport" json:"transport,omitempty"`
	Enabled        bool               `long:"enabled" description:"Whether the RTN provider is enabled" json:"enabled"`
	SubscriberID   int64              `required:"yes" long:"subscriber-id" description:"The target EBICS subscriber ID" json:"subscriberID,omitempty"`
	ClientID       *int64             `long:"client-id" description:"The Gateway client identifier" json:"clientID,omitempty"`
	AutoPullPolicy string             `long:"auto-pull-policy" choice:"MANUAL" choice:"AUTO" choice:"AUTO_FILTERED" default:"MANUAL" description:"The RTN auto-pull policy" json:"autoPullPolicy,omitempty"`
	Configuration  map[string]confVal `long:"config" description:"Provider configuration entries in key:value format. Can be repeated." json:"configuration,omitempty"`
}

func (c *EbicsRTNProviderAdd) Execute([]string) error { return execute(c) }
func (c *EbicsRTNProviderAdd) execute(w io.Writer) error {
	addr.Path = "/api/ebics/rtn/providers"

	if _, err := add(w, c); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS RTN provider %q was successfully added.\n", c.Name)

	return nil
}

type EbicsRTNProviderList struct {
	ListOptions
}

func (c *EbicsRTNProviderList) Execute([]string) error { return execute(c) }
func (c *EbicsRTNProviderList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/rtn/providers"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsRTNProvider
	if err := list(&body); err != nil {
		return err
	}

	if providers := body["providers"]; len(providers) > 0 {
		return outputObject(w, providers, &c.OutputFormat, displayEbicsRTNProviders)
	}

	fmt.Fprintln(w, "No EBICS RTN provider found.")

	return nil
}

type EbicsRTNProviderGet struct {
	OutputFormat

	Args struct {
		Provider string `required:"yes" positional-arg-name:"provider" description:"The RTN provider name"`
	} `positional-args:"yes"`
}

func (c *EbicsRTNProviderGet) Execute([]string) error { return execute(c) }
func (c *EbicsRTNProviderGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/providers", c.Args.Provider)

	var provider api.OutEbicsRTNProvider
	if err := get(&provider); err != nil {
		return err
	}

	return outputObject(w, &provider, &c.OutputFormat, displayEbicsRTNProvider)
}

//nolint:lll // command tags can be long
type EbicsRTNProviderUpdate struct {
	Args struct {
		Provider string `required:"yes" positional-arg-name:"provider" description:"The RTN provider name"`
	} `positional-args:"yes" json:"-"`

	Name           *string             `long:"name" description:"The new RTN provider name" json:"name,omitempty"`
	Transport      *string             `long:"transport" choice:"WSS" description:"The RTN provider transport" json:"transport,omitempty"`
	Enabled        *bool               `long:"enabled" description:"Whether the RTN provider is enabled" json:"enabled,omitempty"`
	SubscriberID   *int64              `long:"subscriber-id" description:"The target EBICS subscriber ID" json:"subscriberID,omitempty"`
	ClientID       *int64              `long:"client-id" description:"The Gateway client identifier" json:"clientID,omitempty"`
	AutoPullPolicy *string             `long:"auto-pull-policy" choice:"MANUAL" choice:"AUTO" choice:"AUTO_FILTERED" description:"The RTN auto-pull policy" json:"autoPullPolicy,omitempty"`
	Configuration  *map[string]confVal `long:"config" description:"Provider configuration entries in key:value format. Can be repeated." json:"configuration,omitempty"`
}

func (c *EbicsRTNProviderUpdate) Execute([]string) error { return execute(c) }
func (c *EbicsRTNProviderUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/providers", c.Args.Provider)

	if err := update(w, c); err != nil {
		return err
	}

	name := c.Args.Provider
	if c.Name != nil && *c.Name != "" {
		name = *c.Name
	}

	fmt.Fprintf(w, "The EBICS RTN provider %q was successfully updated.\n", name)

	return nil
}

type EbicsRTNProviderDelete struct {
	Args struct {
		Provider string `required:"yes" positional-arg-name:"provider" description:"The RTN provider name"`
	} `positional-args:"yes"`
}

func (c *EbicsRTNProviderDelete) Execute([]string) error { return execute(c) }
func (c *EbicsRTNProviderDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/providers", c.Args.Provider)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS RTN provider %q was successfully deleted.\n", c.Args.Provider)

	return nil
}

type EbicsRTNEventList struct {
	ListOptions
}

func (c *EbicsRTNEventList) Execute([]string) error { return execute(c) }
func (c *EbicsRTNEventList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/rtn/events"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsRTNEvent
	if err := list(&body); err != nil {
		return err
	}

	if events := body["events"]; len(events) > 0 {
		return outputObject(w, events, &c.OutputFormat, displayEbicsRTNEvents)
	}

	fmt.Fprintln(w, "No EBICS RTN event found.")

	return nil
}

type EbicsRTNEventGet struct {
	OutputFormat

	Args struct {
		Event string `required:"yes" positional-arg-name:"event" description:"The RTN event identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsRTNEventGet) Execute([]string) error { return execute(c) }
func (c *EbicsRTNEventGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/events", c.Args.Event)

	var event api.OutEbicsRTNEvent
	if err := get(&event); err != nil {
		return err
	}

	return outputObject(w, &event, &c.OutputFormat, displayEbicsRTNEvent)
}

type EbicsRTNEventRetry struct {
	Args struct {
		Event string `required:"yes" positional-arg-name:"event" description:"The RTN event identifier"`
	} `positional-args:"yes" json:"-"`

	Reason string `long:"reason" description:"Operator reason for the retry" json:"reason,omitempty"`
}

func (c *EbicsRTNEventRetry) Execute([]string) error { return execute(c) }
func (c *EbicsRTNEventRetry) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/events", c.Args.Event, "retry")

	if err := updateMethod(w, c, "PUT"); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS RTN event %q was successfully scheduled for retry.\n", c.Args.Event)

	return nil
}

type EbicsRTNEventQuarantine struct {
	Args struct {
		Event string `required:"yes" positional-arg-name:"event" description:"The RTN event identifier"`
	} `positional-args:"yes" json:"-"`

	Reason string `long:"reason" description:"Operator reason for the quarantine" json:"reason,omitempty"`
}

func (c *EbicsRTNEventQuarantine) Execute([]string) error { return execute(c) }
func (c *EbicsRTNEventQuarantine) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/events", c.Args.Event, "quarantine")

	if err := updateMethod(w, c, "PUT"); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS RTN event %q was successfully quarantined.\n", c.Args.Event)

	return nil
}
