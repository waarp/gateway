package wg

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

//nolint:gochecknoglobals //a global var is required here
var Partner string

type PartnerArg struct{}

func (*PartnerArg) UnmarshalFlag(value string) error {
	Partner = value

	return nil
}

func displayPartner(w io.Writer, partner *api.OutPartner) {
	send := strings.Join(partner.AuthorizedRules.Sending, ", ")
	recv := strings.Join(partner.AuthorizedRules.Reception, ", ")

	protoConfig, err := json.Marshal(partner.ProtoConfig)
	if err != nil {
		protoConfig = []byte(red("<error while serializing the configuration>"))
	}

	fmt.Fprintln(w, boldOrange("● Partner %q", partner.Name))
	fmt.Fprintln(w, orange("    Protocol:     "), partner.Protocol)
	fmt.Fprintln(w, orange("    Address:      "), partner.Address)
	fmt.Fprintln(w, orange("    Configuration:"), string(protoConfig))
	fmt.Fprintln(w, orange("    Authorized rules"))
	fmt.Fprintln(w, bold("    ├─Sending:  "), send)
	fmt.Fprintln(w, bold("    └─Reception:"), recv)
}

// ######################## ADD ##########################

//nolint:lll // struct tags for command line arguments can be long
type PartnerAdd struct {
	Name        string             `required:"yes" short:"n" long:"name" description:"The partner's name"`
	Protocol    string             `required:"yes" short:"p" long:"protocol" description:"The partner's protocol"`
	Address     string             `required:"yes" short:"a" long:"address" description:"The partner's [address:port]"`
	ProtoConfig map[string]confVal `short:"c" long:"config" description:"The partner's configuration, in key:val format. Can be repeated."`
}

func (p *PartnerAdd) Execute([]string) error {
	partner := api.InPartner{
		Name:     &p.Name,
		Protocol: &p.Protocol,
		Address:  &p.Address,
	}

	if err := utils.JSONConvert(p.ProtoConfig, &partner.ProtoConfig); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	addr.Path = "/api/partners"

	if err := add(partner); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The partner", bold(p.Name), "was successfully added.")

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type PartnerList struct {
	ListOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (p *PartnerList) Execute([]string) error {
	agentListURL("/api/partners", &p.ListOptions, p.SortBy, p.Protocols)

	body := map[string][]api.OutPartner{}
	if err := list(&body); err != nil {
		return err
	}

	w := getColorable() //nolint:ifshort // decrease readability

	if partners := body["partners"]; len(partners) > 0 {
		fmt.Fprintln(w, bold("Partners:"))

		for i := range partners {
			displayPartner(w, &partners[i])
		}
	} else {
		fmt.Fprintln(w, "No partners found.")
	}

	return nil
}

// ######################## GET ##########################

type PartnerGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
}

func (p *PartnerGet) Execute([]string) error {
	addr.Path = path.Join("/api/partners", p.Args.Name)

	partner := &api.OutPartner{}
	if err := get(partner); err != nil {
		return err
	}

	displayPartner(getColorable(), partner)

	return nil
}

// ######################## DELETE ##########################

type PartnerDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
}

func (p *PartnerDelete) Execute([]string) error {
	addr.Path = path.Join("/api/partners", p.Args.Name)

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The partner", bold(p.Args.Name), "was successfully deleted.")

	return nil
}

// ######################## UPDATE ##########################

//nolint:lll // struct tags for command line arguments can be long
type PartnerUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
	Name        *string            `short:"n" long:"name" description:"The partner's name"`
	Protocol    *string            `short:"p" long:"protocol" description:"The partner's protocol'"`
	Address     *string            `short:"a" long:"address" description:"The partner's [address:port]"`
	ProtoConfig map[string]confVal `short:"c" long:"config" description:"The partner's configuration, in key:val format. Can be repeated."`
}

func (p *PartnerUpdate) Execute([]string) error {
	partner := &api.InPartner{
		Name:     p.Name,
		Protocol: p.Protocol,
		Address:  p.Address,
	}

	if err := utils.JSONConvert(p.ProtoConfig, &partner.ProtoConfig); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	addr.Path = path.Join("/api/partners", p.Args.Name)

	if err := update(partner); err != nil {
		return err
	}

	name := p.Args.Name
	if partner.Name != nil && *partner.Name != "" {
		name = *partner.Name
	}

	fmt.Fprintln(getColorable(), "The partner", bold(name), "was successfully updated.")

	return nil
}

// ######################## AUTHORIZE ##########################

//nolint:lll // struct tags for command line arguments can be long
type PartnerAuthorize struct {
	Args struct {
		Partner   string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction" choice:"send" choice:"receive"`
	} `positional-args:"yes"`
}

func (p *PartnerAuthorize) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/authorize/%s/%s", p.Args.Partner,
		p.Args.Rule, p.Args.Direction)

	return authorize(out, "partner", p.Args.Partner, p.Args.Rule, p.Args.Direction)
}

// ######################## REVOKE ##########################

type PartnerRevoke struct {
	Args struct {
		Partner   string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (p *PartnerRevoke) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/revoke/%s/%s", p.Args.Partner,
		p.Args.Rule, p.Args.Direction)

	return revoke(out, "partner", p.Args.Partner, p.Args.Rule, p.Args.Direction)
}
