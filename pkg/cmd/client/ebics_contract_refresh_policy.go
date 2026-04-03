package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

const cliBoolTrue = "true"

func displayEbicsContractRefreshPolicies(w io.Writer, policies []*api.OutEbicsContractRefreshPolicy) error {
	Style0.PrintV(w, "=== EBICS contract refresh policies ===")
	for _, policy := range policies {
		if err := displayEbicsContractRefreshPolicy(w, policy); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsContractRefreshPolicy(w io.Writer, policy *api.OutEbicsContractRefreshPolicy) error {
	Style1.Printf(w, "EBICS contract refresh policy %q [%s]", policy.Name, policy.Status)
	Style22.PrintL(w, "Enabled", policy.Enabled)
	Style22.PrintL(w, "Client ID", policy.ClientID)
	Style22.Option(w, "Client name", policy.ClientName)
	Style22.PrintL(w, "Subscriber ID", policy.SubscriberID)
	Style22.Option(w, "Host ID", policy.HostID)
	Style22.Option(w, "Partner ID", policy.PartnerID)
	Style22.Option(w, "User ID", policy.UserID)
	Style22.PrintL(w, "Include HEV", policy.IncludeHEV)
	Style22.PrintL(w, "Interval (s)", policy.IntervalSeconds)
	Style22.Option(w, "Next run at", policy.NextRunAt)
	Style22.Option(w, "Last attempt at", policy.LastAttemptAt)
	Style22.Option(w, "Last success at", policy.LastSuccessAt)
	Style22.Option(w, "Last error", policy.LastError)
	Style22.Option(w, "Activation status", policy.ActivationStatus)
	Style22.Option(w, "Activation reason", policy.ActivationReason)

	return nil
}

type EbicsContractRefreshPolicyAdd struct {
	Name            string  `required:"yes" long:"name" description:"The contract refresh policy name"`
	ClientID        int64   `required:"yes" long:"client-id" description:"The Gateway EBICS client identifier"`
	SubscriberID    int64   `required:"yes" long:"subscriber" description:"The EBICS subscriber identifier"`
	Enabled         *string `long:"enabled" choice:"true" choice:"false" description:"Whether the policy is enabled"`
	IncludeHEV      *string `long:"include-hev" choice:"true" choice:"false" description:"Include HEV before refresh"`
	IntervalSeconds int64   `required:"yes" long:"interval-seconds" description:"The refresh interval in seconds"`
}

func (c *EbicsContractRefreshPolicyAdd) Execute([]string) error { return execute(c) }
func (c *EbicsContractRefreshPolicyAdd) execute(w io.Writer) error {
	addr.Path = "/api/ebics/contract-refresh-policies"

	body := api.InEbicsContractRefreshPolicy{
		Name:            c.Name,
		ClientID:        c.ClientID,
		SubscriberID:    c.SubscriberID,
		IntervalSeconds: c.IntervalSeconds,
	}
	if c.Enabled != nil {
		value := *c.Enabled == cliBoolTrue
		body.Enabled = &value
	}
	if c.IncludeHEV != nil {
		value := *c.IncludeHEV == cliBoolTrue
		body.IncludeHEV = &value
	}
	if _, err := add(w, &body); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS contract refresh policy %q was successfully added.\n", c.Name)

	return nil
}

type EbicsContractRefreshPolicyList struct {
	ListOptions
}

func (c *EbicsContractRefreshPolicyList) Execute([]string) error { return execute(c) }
func (c *EbicsContractRefreshPolicyList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/contract-refresh-policies"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsContractRefreshPolicy
	if err := list(&body); err != nil {
		return err
	}

	if policies := body["contractRefreshPolicies"]; len(policies) > 0 {
		return outputObject(w, policies, &c.OutputFormat, displayEbicsContractRefreshPolicies)
	}

	fmt.Fprintln(w, "No EBICS contract refresh policy found.")

	return nil
}

type EbicsContractRefreshPolicyGet struct {
	OutputFormat

	Args struct {
		Policy string `required:"yes" positional-arg-name:"policy" description:"The contract refresh policy name"`
	} `positional-args:"yes"`
}

func (c *EbicsContractRefreshPolicyGet) Execute([]string) error { return execute(c) }
func (c *EbicsContractRefreshPolicyGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/contract-refresh-policies", c.Args.Policy)

	var policy api.OutEbicsContractRefreshPolicy
	if err := get(&policy); err != nil {
		return err
	}

	return outputObject(w, &policy, &c.OutputFormat, displayEbicsContractRefreshPolicy)
}

type EbicsContractRefreshPolicyUpdate struct {
	Args struct {
		Policy string `required:"yes" positional-arg-name:"policy" description:"The contract refresh policy name"`
	} `positional-args:"yes"`

	Name            *string `long:"name" description:"The new contract refresh policy name"`
	ClientID        *int64  `long:"client-id" description:"The Gateway EBICS client identifier"`
	SubscriberID    *int64  `long:"subscriber" description:"The EBICS subscriber identifier"`
	Enabled         *string `long:"enabled" choice:"true" choice:"false" description:"Whether the policy is enabled"`
	IncludeHEV      *string `long:"include-hev" choice:"true" choice:"false" description:"Include HEV before refresh"`
	IntervalSeconds *int64  `long:"interval-seconds" description:"The refresh interval in seconds"`
}

func (c *EbicsContractRefreshPolicyUpdate) Execute([]string) error { return execute(c) }
func (c *EbicsContractRefreshPolicyUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/contract-refresh-policies", c.Args.Policy)

	body := map[string]any{}
	if c.Name != nil {
		body["name"] = *c.Name
	}
	if c.ClientID != nil {
		body["clientID"] = *c.ClientID
	}
	if c.SubscriberID != nil {
		body["subscriberID"] = *c.SubscriberID
	}
	if c.Enabled != nil {
		body["enabled"] = *c.Enabled == cliBoolTrue
	}
	if c.IncludeHEV != nil {
		body["includeHEV"] = *c.IncludeHEV == cliBoolTrue
	}
	if c.IntervalSeconds != nil {
		body["intervalSeconds"] = *c.IntervalSeconds
	}

	if err := updateMethod(w, body, "PATCH"); err != nil {
		return err
	}

	name := c.Args.Policy
	if c.Name != nil && *c.Name != "" {
		name = *c.Name
	}

	fmt.Fprintf(w, "The EBICS contract refresh policy %q was successfully updated.\n", name)

	return nil
}

type EbicsContractRefreshPolicyDelete struct {
	Args struct {
		Policy string `required:"yes" positional-arg-name:"policy" description:"The contract refresh policy name"`
	} `positional-args:"yes"`
}

func (c *EbicsContractRefreshPolicyDelete) Execute([]string) error { return execute(c) }
func (c *EbicsContractRefreshPolicyDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/contract-refresh-policies", c.Args.Policy)
	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS contract refresh policy %q was successfully deleted.\n", c.Args.Policy)

	return nil
}
