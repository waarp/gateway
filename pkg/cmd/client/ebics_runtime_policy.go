package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func displayEbicsRuntimePolicy(w io.Writer, policy *api.OutEbicsRuntimePolicy) error {
	Style1.PrintV(w, "EBICS runtime policy")
	Style22.PrintL(w, "Enabled", policy.Enabled)
	Style22.PrintL(w, "Maintenance interval (s)", policy.MaintenanceIntervalSeconds)
	Style22.PrintL(w, "Transaction retention (s)", policy.TransactionRetentionSeconds)
	Style22.PrintL(w, "RTN event retention (s)", policy.RTNEventRetentionSeconds)

	return nil
}

type EbicsRuntimePolicyGet struct {
	OutputFormat
}

func (c *EbicsRuntimePolicyGet) Execute([]string) error { return execute(c) }
func (c *EbicsRuntimePolicyGet) execute(w io.Writer) error {
	addr.Path = "/api/ebics/runtime-policy"

	var policy api.OutEbicsRuntimePolicy
	if err := get(&policy); err != nil {
		return err
	}

	return outputObject(w, &policy, &c.OutputFormat, displayEbicsRuntimePolicy)
}

type EbicsRuntimePolicyUpdate struct {
	Enabled                     *string `long:"enabled" choice:"true" choice:"false" description:"Enable policy"`
	MaintenanceIntervalSeconds  *int64  `long:"maintenance-interval-seconds" description:"Interval in seconds"`
	TransactionRetentionSeconds *int64  `long:"transaction-retention-seconds" description:"Tx retention in seconds"`
	RTNEventRetentionSeconds    *int64  `long:"rtn-event-retention-seconds" description:"RTN retention in seconds"`
}

func (c *EbicsRuntimePolicyUpdate) Execute([]string) error { return execute(c) }
func (c *EbicsRuntimePolicyUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/runtime-policy")

	body := map[string]any{}
	if c.Enabled != nil {
		body["enabled"] = *c.Enabled == "true"
	}
	if c.MaintenanceIntervalSeconds != nil {
		body["maintenanceIntervalSeconds"] = *c.MaintenanceIntervalSeconds
	}
	if c.TransactionRetentionSeconds != nil {
		body["transactionRetentionSeconds"] = *c.TransactionRetentionSeconds
	}
	if c.RTNEventRetentionSeconds != nil {
		body["rtnEventRetentionSeconds"] = *c.RTNEventRetentionSeconds
	}

	if err := updateMethod(w, body, "PUT"); err != nil {
		return err
	}

	fmt.Fprintln(w, "The EBICS runtime policy was successfully updated.")

	return nil
}
