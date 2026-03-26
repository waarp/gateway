package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func displayEbicsPayloadProfiles(w io.Writer, profiles []*api.OutEbicsPayloadProfile) error {
	Style0.PrintV(w, "=== EBICS payload profiles ===")
	for _, profile := range profiles {
		if err := displayEbicsPayloadProfile(w, profile); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsPayloadProfile(w io.Writer, profile *api.OutEbicsPayloadProfile) error {
	Style1.Printf(w, "EBICS payload profile %q", profile.Name)
	Style22.Option(w, "Label", profile.Label)
	Style22.PrintL(w, "Order type", profile.OrderType)
	Style22.PrintL(w, "Direction", profile.Direction)
	Style22.Option(w, "Service name", profile.ServiceName)
	Style22.Option(w, "Service option", profile.ServiceOption)
	Style22.Option(w, "Scope", profile.Scope)
	Style22.Option(w, "Message name", profile.MsgName)
	Style22.Option(w, "Container type", profile.ContainerType)
	Style22.Option(w, "Default rule", profile.DefaultRule)
	Style22.PrintL(w, "Enabled", profile.IsEnabled)

	return nil
}

func displayEbicsContractViews(w io.Writer, views []*api.OutEbicsContractView) error {
	Style0.PrintV(w, "=== EBICS contract views ===")
	for _, view := range views {
		if err := displayEbicsContractView(w, view); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsContractView(w io.Writer, view *api.OutEbicsContractView) error {
	Style1.Printf(w, "EBICS contract view #%d [%s]", view.ID, view.Status)
	Style22.PrintL(w, "Host ID", view.HostID)
	Style22.Option(w, "Partner ID", view.PartnerID)
	Style22.Option(w, "User ID", view.UserID)
	Style22.PrintL(w, "Source order", view.SourceOrderType)
	Style22.Option(w, "Version tag", view.VersionTag)
	Style22.PrintL(w, "Fetched at", view.FetchedAt)

	return nil
}

//nolint:lll // CLI tags are intentionally explicit
type EbicsPayloadProfileAdd struct {
	Name                   string             `required:"yes" long:"name" description:"The payload profile name" json:"name,omitempty"`
	Label                  string             `long:"label" description:"The payload profile label" json:"label,omitempty"`
	Description            string             `long:"description" description:"The payload profile description" json:"description,omitempty"`
	OrderType              string             `required:"yes" long:"order-type" choice:"BTU" choice:"BTD" choice:"FUL" choice:"FDL" description:"The EBICS payload order type" json:"orderType,omitempty"`
	Direction              string             `required:"yes" long:"direction" choice:"UPLOAD" choice:"DOWNLOAD" choice:"BIDIRECTIONAL" description:"The payload direction" json:"direction,omitempty"`
	ServiceName            string             `long:"service-name" description:"The EBICS service name" json:"serviceName,omitempty"`
	ServiceOption          string             `long:"service-option" description:"The EBICS service option" json:"serviceOption,omitempty"`
	Scope                  string             `long:"scope" description:"The EBICS scope" json:"scope,omitempty"`
	MsgName                string             `long:"msg-name" description:"The EBICS message name" json:"msgName,omitempty"`
	ContainerType          string             `long:"container-type" description:"The EBICS container type" json:"containerType,omitempty"`
	DefaultRule            string             `long:"default-rule" description:"The default Gateway rule" json:"defaultRule,omitempty"`
	DefaultTargetDirectory string             `long:"target-dir" description:"The default target directory" json:"defaultTargetDirectory,omitempty"`
	RequiresDeclaredAmount bool               `long:"requires-declared-amount" description:"Whether the profile requires a declared amount" json:"requiresDeclaredAmount,omitempty"`
	DefaultCurrency        string             `long:"currency" description:"The default currency" json:"defaultCurrency,omitempty"`
	FilenamePattern        string             `long:"filename-pattern" description:"The filename pattern" json:"filenamePattern,omitempty"`
	StrictContractCheck    *bool              `long:"strict-contract-check" description:"Whether strict contract checks are enabled" json:"strictContractCheck,omitempty"`
	IsEnabled              *bool              `long:"enabled" description:"Whether the profile is enabled" json:"isEnabled,omitempty"`
	AllowedExtensions      []string           `long:"allowed-extension" description:"Allowed file extensions. Can be repeated." json:"allowedExtensions,omitempty"`
	Metadata               map[string]confVal `long:"metadata" description:"Profile metadata in key:value format. Can be repeated." json:"metadata,omitempty"`
}

func (c *EbicsPayloadProfileAdd) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadProfileAdd) execute(w io.Writer) error {
	addr.Path = "/api/ebics/payload-profiles"

	if _, err := add(w, c); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS payload profile %q was successfully added.\n", c.Name)

	return nil
}

type EbicsPayloadProfileList struct {
	ListOptions
}

func (c *EbicsPayloadProfileList) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadProfileList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/payload-profiles"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsPayloadProfile
	if err := list(&body); err != nil {
		return err
	}

	if profiles := body["payloadProfiles"]; len(profiles) > 0 {
		return outputObject(w, profiles, &c.OutputFormat, displayEbicsPayloadProfiles)
	}

	fmt.Fprintln(w, "No EBICS payload profile found.")

	return nil
}

type EbicsPayloadProfileGet struct {
	OutputFormat

	Args struct {
		Profile string `required:"yes" positional-arg-name:"profile" description:"The payload profile name"`
	} `positional-args:"yes"`
}

func (c *EbicsPayloadProfileGet) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadProfileGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/payload-profiles", c.Args.Profile)

	var profile api.OutEbicsPayloadProfile
	if err := get(&profile); err != nil {
		return err
	}

	return outputObject(w, &profile, &c.OutputFormat, displayEbicsPayloadProfile)
}

//nolint:lll // CLI tags are intentionally explicit
type EbicsPayloadProfileUpdate struct {
	Args struct {
		Profile string `required:"yes" positional-arg-name:"profile" description:"The payload profile name"`
	} `positional-args:"yes" json:"-"`

	Name                   *string             `long:"name" description:"The new payload profile name" json:"name,omitempty"`
	Label                  *string             `long:"label" description:"The payload profile label" json:"label,omitempty"`
	Description            *string             `long:"description" description:"The payload profile description" json:"description,omitempty"`
	OrderType              *string             `long:"order-type" choice:"BTU" choice:"BTD" choice:"FUL" choice:"FDL" description:"The EBICS payload order type" json:"orderType,omitempty"`
	Direction              *string             `long:"direction" choice:"UPLOAD" choice:"DOWNLOAD" choice:"BIDIRECTIONAL" description:"The payload direction" json:"direction,omitempty"`
	ServiceName            *string             `long:"service-name" description:"The EBICS service name" json:"serviceName,omitempty"`
	ServiceOption          *string             `long:"service-option" description:"The EBICS service option" json:"serviceOption,omitempty"`
	Scope                  *string             `long:"scope" description:"The EBICS scope" json:"scope,omitempty"`
	MsgName                *string             `long:"msg-name" description:"The EBICS message name" json:"msgName,omitempty"`
	ContainerType          *string             `long:"container-type" description:"The EBICS container type" json:"containerType,omitempty"`
	DefaultRule            *string             `long:"default-rule" description:"The default Gateway rule" json:"defaultRule,omitempty"`
	DefaultTargetDirectory *string             `long:"target-dir" description:"The default target directory" json:"defaultTargetDirectory,omitempty"`
	RequiresDeclaredAmount *bool               `long:"requires-declared-amount" description:"Whether the profile requires a declared amount" json:"requiresDeclaredAmount,omitempty"`
	DefaultCurrency        *string             `long:"currency" description:"The default currency" json:"defaultCurrency,omitempty"`
	FilenamePattern        *string             `long:"filename-pattern" description:"The filename pattern" json:"filenamePattern,omitempty"`
	StrictContractCheck    *bool               `long:"strict-contract-check" description:"Whether strict contract checks are enabled" json:"strictContractCheck,omitempty"`
	IsEnabled              *bool               `long:"enabled" description:"Whether the profile is enabled" json:"isEnabled,omitempty"`
	AllowedExtensions      *[]string           `long:"allowed-extension" description:"Allowed file extensions. Can be repeated." json:"allowedExtensions,omitempty"`
	Metadata               *map[string]confVal `long:"metadata" description:"Profile metadata in key:value format. Can be repeated." json:"metadata,omitempty"`
}

func (c *EbicsPayloadProfileUpdate) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadProfileUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/payload-profiles", c.Args.Profile)

	if err := update(w, c); err != nil {
		return err
	}

	name := c.Args.Profile
	if c.Name != nil && *c.Name != "" {
		name = *c.Name
	}

	fmt.Fprintf(w, "The EBICS payload profile %q was successfully updated.\n", name)

	return nil
}

type EbicsContractViewList struct {
	ListOptions
}

func (c *EbicsContractViewList) Execute([]string) error { return execute(c) }
func (c *EbicsContractViewList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/contract-views"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsContractView
	if err := list(&body); err != nil {
		return err
	}

	if views := body["contractViews"]; len(views) > 0 {
		return outputObject(w, views, &c.OutputFormat, displayEbicsContractViews)
	}

	fmt.Fprintln(w, "No EBICS contract view found.")

	return nil
}

type EbicsContractViewGet struct {
	OutputFormat

	Args struct {
		View string `required:"yes" positional-arg-name:"view" description:"The contract view identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsContractViewGet) Execute([]string) error { return execute(c) }
func (c *EbicsContractViewGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/contract-views", c.Args.View)

	var body struct {
		ContractView *api.OutEbicsContractView       `json:"contractView"`
		Items        []*api.OutEbicsContractViewItem `json:"items"`
	}
	if err := get(&body); err != nil {
		return err
	}

	return outputObject(w, body.ContractView, &c.OutputFormat, displayEbicsContractView)
}
