package wg

import (
	"errors"
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

var errMissingEbicsServerContractSetPayload = errors.New("missing EBICS server contract set payload in response")

type outEbicsServerContractSetDetail struct {
	ServerContractSet *api.OutEbicsServerContractSet    `json:"serverContractSet"`
	Items             []*api.OutEbicsServerContractItem `json:"items"`
}

func displayEbicsServerContractSets(w io.Writer, sets []*api.OutEbicsServerContractSet) error {
	Style0.PrintV(w, "=== EBICS server contract sets ===")
	for _, set := range sets {
		if err := displayEbicsServerContractSet(w, set); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsServerContractSet(w io.Writer, set *api.OutEbicsServerContractSet) error {
	Style1.Printf(w, "EBICS server contract set %q [%s]", set.Name, set.Status)
	Style22.PrintL(w, "Host ID", set.HostID)
	Style22.PrintL(w, "Scope", set.Scope)
	Style22.Option(w, "Partner ID", set.PartnerID)
	Style22.Option(w, "User ID", set.UserID)
	Style22.PrintL(w, "Source order", set.SourceOrderType)
	Style22.Option(w, "Version tag", set.VersionTag)
	Style22.Option(w, "Published at", set.PublishedAt)
	Style22.Option(w, "Description", set.Description)

	return nil
}

func displayEbicsServerContractSetItems(w io.Writer, items []*api.OutEbicsServerContractItem) error {
	if len(items) == 0 {
		Style22.PrintL(w, "Items", none)

		return nil
	}

	Style22.Printf(w, "Items:")
	for _, item := range items {
		label := item.ItemType
		if item.OrderType != "" {
			label += " / " + item.OrderType
		} else if item.AdminOrderType != "" {
			label += " / " + item.AdminOrderType
		}

		Style333.Printf(
			w,
			"%s [%s] service=%s/%s scope=%s msg=%s account=%s auth=%s max=%s %s enabled=%t payload=%s",
			item.ItemKey,
			label,
			withDefault(item.ServiceName, none),
			withDefault(item.ServiceOption, none),
			withDefault(item.Scope, none),
			withDefault(item.MsgName, none),
			withDefault(item.AccountID, none),
			withDefault(item.AuthorisationLevel, none),
			withDefault(item.MaxAmountValue, none),
			withDefault(item.MaxAmountCurrency, none),
			item.IsEnabled,
			withDefault(item.Payload, none),
		)
	}

	return nil
}

func displayEbicsServerContractSetDetail(w io.Writer, body *outEbicsServerContractSetDetail) error {
	if body == nil || body.ServerContractSet == nil {
		return errMissingEbicsServerContractSetPayload
	}

	if err := displayEbicsServerContractSet(w, body.ServerContractSet); err != nil {
		return err
	}

	return displayEbicsServerContractSetItems(w, body.Items)
}

type EbicsServerContractSetList struct {
	ListOptions
}

func (c *EbicsServerContractSetList) Execute([]string) error { return execute(c) }
func (c *EbicsServerContractSetList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/server-contract-sets"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsServerContractSet
	if err := list(&body); err != nil {
		return err
	}

	if sets := body["serverContractSets"]; len(sets) > 0 {
		return outputObject(w, sets, &c.OutputFormat, displayEbicsServerContractSets)
	}

	fmt.Fprintln(w, "No EBICS server contract set found.")

	return nil
}

type EbicsServerContractSetGet struct {
	OutputFormat

	Args struct {
		Set string `required:"yes" positional-arg-name:"set" description:"The server contract set identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsServerContractSetGet) Execute([]string) error { return execute(c) }
func (c *EbicsServerContractSetGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/server-contract-sets", c.Args.Set)

	var body outEbicsServerContractSetDetail
	if err := get(&body); err != nil {
		return err
	}

	return outputObject(w, &body, &c.OutputFormat, displayEbicsServerContractSetDetail)
}
