package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func displayEbicsRTNOutboundProviders(w io.Writer, providers []*api.OutEbicsRTNOutboundProvider) error {
	Style0.PrintV(w, "=== EBICS outbound RTN providers ===")
	for _, provider := range providers {
		if err := displayEbicsRTNOutboundProvider(w, provider); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsRTNOutboundProvider(w io.Writer, provider *api.OutEbicsRTNOutboundProvider) error {
	Style1.Printf(w, "Outbound RTN provider %q", provider.Name)
	Style22.PrintL(w, "Transport", provider.Transport)
	Style22.PrintL(w, "Enabled", provider.Enabled)
	Style22.PrintL(w, "Subscriber ID", provider.SubscriberID)
	Style22.Option(w, "Activation status", provider.ActivationStatus)
	Style22.Option(w, "Activation reason", provider.ActivationReason)
	if provider.LastConnectionAt != nil {
		Style22.PrintL(w, "Last connection", *provider.LastConnectionAt)
	}
	Style22.Option(w, "Last error", provider.LastError)

	return nil
}

func displayEbicsRTNOutboundNotifications(
	w io.Writer,
	notifications []*api.OutEbicsRTNOutboundNotification,
) error {
	Style0.PrintV(w, "=== EBICS outbound RTN notifications ===")
	for _, notification := range notifications {
		if err := displayEbicsRTNOutboundNotification(w, notification); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsRTNOutboundNotification(w io.Writer, notification *api.OutEbicsRTNOutboundNotification) error {
	Style1.Printf(w, "Outbound RTN notification #%d [%s]", notification.ID, notification.Status)
	Style22.PrintL(w, "Provider ID", notification.ProviderID)
	Style22.PrintL(w, "Event type", notification.EventType)
	Style22.PrintL(w, "Source order type", notification.SourceOrderType)
	Style22.Option(w, "Correlation ID", notification.CorrelationID)
	Style22.PrintL(w, "Subscriber ID", notification.SubscriberID)
	if notification.ServerReportingSetID != nil {
		Style22.PrintL(w, "Server reporting set ID", *notification.ServerReportingSetID)
	}
	Style22.Option(w, "Server reporting item key", notification.ServerReportingItemKey)
	Style22.PrintL(w, "Attempts", notification.Attempts)
	if notification.NextRetryAt != nil {
		Style22.PrintL(w, "Next retry at", *notification.NextRetryAt)
	}
	if notification.SentAt != nil {
		Style22.PrintL(w, "Sent at", *notification.SentAt)
	}
	Style22.Option(w, "Last error", notification.LastError)
	if len(notification.Payload) > 0 {
		Style22.Printf(w, "Payload:")
		displayMap(w, Style333, notification.Payload)
	}

	return nil
}

//nolint:lll // command tags are intentionally explicit
type EbicsRTNOutboundProviderAdd struct {
	Name          string             `required:"yes" long:"name" description:"The outbound RTN provider name" json:"name,omitempty"`
	Transport     string             `long:"transport" choice:"WSS" default:"WSS" description:"The outbound RTN provider transport" json:"transport,omitempty"`
	Enabled       bool               `long:"enabled" description:"Whether the outbound RTN provider is enabled" json:"enabled"`
	SubscriberID  int64              `required:"yes" long:"subscriber-id" description:"The target EBICS subscriber ID" json:"subscriberID,omitempty"`
	Configuration map[string]confVal `long:"config" description:"Provider configuration entries in key:value format. Can be repeated." json:"configuration,omitempty"`
}

func (c *EbicsRTNOutboundProviderAdd) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundProviderAdd) execute(w io.Writer) error {
	addr.Path = "/api/ebics/rtn/outbound/providers"
	if _, err := add(w, c); err != nil {
		return err
	}

	fmt.Fprintf(w, "The outbound RTN provider %q was successfully added.\n", c.Name)

	return nil
}

type EbicsRTNOutboundProviderList struct {
	ListOptions
}

func (c *EbicsRTNOutboundProviderList) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundProviderList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/rtn/outbound/providers"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsRTNOutboundProvider
	if err := list(&body); err != nil {
		return err
	}
	if providers := body["providers"]; len(providers) > 0 {
		return outputObject(w, providers, &c.OutputFormat, displayEbicsRTNOutboundProviders)
	}

	fmt.Fprintln(w, "No outbound RTN provider found.")

	return nil
}

type EbicsRTNOutboundProviderGet struct {
	OutputFormat

	Args struct {
		Provider string `required:"yes" positional-arg-name:"provider" description:"The outbound RTN provider name"`
	} `positional-args:"yes"`
}

func (c *EbicsRTNOutboundProviderGet) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundProviderGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/outbound/providers", c.Args.Provider)

	var provider api.OutEbicsRTNOutboundProvider
	if err := get(&provider); err != nil {
		return err
	}

	return outputObject(w, &provider, &c.OutputFormat, displayEbicsRTNOutboundProvider)
}

//nolint:lll // command tags are intentionally explicit
type EbicsRTNOutboundProviderUpdate struct {
	Args struct {
		Provider string `required:"yes" positional-arg-name:"provider" description:"The outbound RTN provider name"`
	} `positional-args:"yes" json:"-"`

	Name          *string             `long:"name" description:"The new outbound RTN provider name" json:"name,omitempty"`
	Transport     *string             `long:"transport" choice:"WSS" description:"The outbound RTN provider transport" json:"transport,omitempty"`
	Enabled       *bool               `long:"enabled" description:"Whether the outbound RTN provider is enabled" json:"enabled,omitempty"`
	SubscriberID  *int64              `long:"subscriber-id" description:"The target EBICS subscriber ID" json:"subscriberID,omitempty"`
	Configuration *map[string]confVal `long:"config" description:"Provider configuration entries in key:value format. Can be repeated." json:"configuration,omitempty"`
}

func (c *EbicsRTNOutboundProviderUpdate) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundProviderUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/outbound/providers", c.Args.Provider)
	if err := update(w, c); err != nil {
		return err
	}

	name := c.Args.Provider
	if c.Name != nil && *c.Name != "" {
		name = *c.Name
	}

	fmt.Fprintf(w, "The outbound RTN provider %q was successfully updated.\n", name)

	return nil
}

type EbicsRTNOutboundProviderDelete struct {
	Args struct {
		Provider string `required:"yes" positional-arg-name:"provider" description:"The outbound RTN provider name"`
	} `positional-args:"yes"`
}

func (c *EbicsRTNOutboundProviderDelete) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundProviderDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/outbound/providers", c.Args.Provider)
	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The outbound RTN provider %q was successfully deleted.\n", c.Args.Provider)

	return nil
}

//nolint:lll // command tags are intentionally explicit
type EbicsRTNOutboundNotificationAdd struct {
	ProviderID           int64  `required:"yes" long:"provider-id" description:"The outbound RTN provider identifier" json:"providerID,omitempty"`
	ServerReportingSetID int64  `required:"yes" long:"server-reporting-set-id" description:"The EBICS server reporting set identifier" json:"serverReportingSetID,omitempty"`
	ItemKey              string `required:"yes" long:"item-key" description:"The reporting item key to notify" json:"itemKey,omitempty"`
}

func (c *EbicsRTNOutboundNotificationAdd) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundNotificationAdd) execute(w io.Writer) error {
	addr.Path = "/api/ebics/rtn/outbound/notifications"
	if _, err := add(w, c); err != nil {
		return err
	}

	fmt.Fprintln(w, "The outbound RTN notification was successfully queued.")

	return nil
}

type EbicsRTNOutboundNotificationList struct {
	ListOptions
}

func (c *EbicsRTNOutboundNotificationList) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundNotificationList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/rtn/outbound/notifications"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsRTNOutboundNotification
	if err := list(&body); err != nil {
		return err
	}
	if notifications := body["notifications"]; len(notifications) > 0 {
		return outputObject(w, notifications, &c.OutputFormat, displayEbicsRTNOutboundNotifications)
	}

	fmt.Fprintln(w, "No outbound RTN notification found.")

	return nil
}

//nolint:lll // command tags are intentionally explicit
type EbicsRTNOutboundNotificationGet struct {
	OutputFormat

	Args struct {
		Notification string `required:"yes" positional-arg-name:"notification" description:"The outbound RTN notification identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsRTNOutboundNotificationGet) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundNotificationGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/outbound/notifications", c.Args.Notification)

	var notification api.OutEbicsRTNOutboundNotification
	if err := get(&notification); err != nil {
		return err
	}

	return outputObject(w, &notification, &c.OutputFormat, displayEbicsRTNOutboundNotification)
}

//nolint:lll // command tags are intentionally explicit
type EbicsRTNOutboundNotificationRetry struct {
	Args struct {
		Notification string `required:"yes" positional-arg-name:"notification" description:"The outbound RTN notification identifier"`
	} `positional-args:"yes" json:"-"`

	Action string `json:"action,omitempty"`
	Reason string `long:"reason" description:"Operator reason for the retry" json:"reason,omitempty"`
}

func (c *EbicsRTNOutboundNotificationRetry) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundNotificationRetry) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/outbound/notifications", c.Args.Notification, "actions")
	c.Action = "RETRY"
	if err := updateMethod(w, c, "PUT"); err != nil {
		return err
	}

	fmt.Fprintf(w, "The outbound RTN notification %q was successfully scheduled for retry.\n", c.Args.Notification)

	return nil
}

//nolint:lll // command tags are intentionally explicit
type EbicsRTNOutboundNotificationQuarantine struct {
	Args struct {
		Notification string `required:"yes" positional-arg-name:"notification" description:"The outbound RTN notification identifier"`
	} `positional-args:"yes" json:"-"`

	Action string `json:"action,omitempty"`
	Reason string `long:"reason" description:"Operator reason for the quarantine" json:"reason,omitempty"`
}

func (c *EbicsRTNOutboundNotificationQuarantine) Execute([]string) error { return execute(c) }
func (c *EbicsRTNOutboundNotificationQuarantine) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/rtn/outbound/notifications", c.Args.Notification, "actions")
	c.Action = "QUARANTINE"
	if err := updateMethod(w, c, "PUT"); err != nil {
		return err
	}

	fmt.Fprintf(w, "The outbound RTN notification %q was successfully quarantined.\n", c.Args.Notification)

	return nil
}
