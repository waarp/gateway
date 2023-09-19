package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //a global var is required here
var Partner string

type PartnerArg struct{}

func (*PartnerArg) UnmarshalFlag(value string) error {
	Partner = value

	return nil
}

func DisplayPartner(w io.Writer, partner *api.OutPartner) {
	f := NewFormatter(w)
	defer f.Render()

	displayPartner(f, partner)
}

func displayPartner(f *Formatter, partner *api.OutPartner) {
	f.Title("Partner %q", partner.Name)
	f.Indent()

	defer f.UnIndent()

	f.Value("Protocol", partner.Protocol)
	f.Value("Address", partner.Address)

	displayProtoConfig(f, partner.ProtoConfig)
	displayAuthorizedRules(f, partner.AuthorizedRules)
}

// ######################## ADD ##########################

//nolint:lll // struct tags for command line arguments can be long
type PartnerAdd struct {
	Name        string             `required:"yes" short:"n" long:"name" description:"The partner's name"`
	Protocol    string             `required:"yes" short:"p" long:"protocol" description:"The partner's protocol"`
	Address     string             `required:"yes" short:"a" long:"address" description:"The partner's [address:port]"`
	ProtoConfig map[string]confVal `short:"c" long:"config" description:"The partner's configuration, in key:val format. Can be repeated."`
}

func (p *PartnerAdd) Execute([]string) error { return p.execute(stdOutput) }
func (p *PartnerAdd) execute(w io.Writer) error {
	partner := api.InPartner{
		Name:     &p.Name,
		Protocol: &p.Protocol,
		Address:  &p.Address,
	}

	if err := utils.JSONConvert(p.ProtoConfig, &partner.ProtoConfig); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	addr.Path = "/api/partners"

	if _, err := add(w, partner); err != nil {
		return err
	}

	fmt.Fprintf(w, "The partner %q was successfully added.\n", p.Name)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type PartnerList struct {
	ListOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (p *PartnerList) Execute([]string) error { return p.execute(stdOutput) }

//nolint:dupl //duplicate is for a different type, best keep separate
func (p *PartnerList) execute(w io.Writer) error {
	agentListURL("/api/partners", &p.ListOptions, p.SortBy, p.Protocols)

	body := map[string][]*api.OutPartner{}
	if err := list(&body); err != nil {
		return err
	}

	if partners := body["partners"]; len(partners) > 0 {
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("Partners:")

		for _, partner := range partners {
			displayPartner(f, partner)
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

func (p *PartnerGet) Execute([]string) error { return p.execute(stdOutput) }
func (p *PartnerGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/partners", p.Args.Name)

	partner := &api.OutPartner{}
	if err := get(partner); err != nil {
		return err
	}

	DisplayPartner(w, partner)

	return nil
}

// ######################## DELETE ##########################

type PartnerDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
}

func (p *PartnerDelete) Execute([]string) error { return p.execute(stdOutput) }
func (p *PartnerDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/partners", p.Args.Name)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The partner %q was successfully deleted.\n", p.Args.Name)

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

func (p *PartnerUpdate) Execute([]string) error { return p.execute(stdOutput) }
func (p *PartnerUpdate) execute(w io.Writer) error {
	partner := &api.InPartner{
		Name:     p.Name,
		Protocol: p.Protocol,
		Address:  p.Address,
	}

	if err := utils.JSONConvert(p.ProtoConfig, &partner.ProtoConfig); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	addr.Path = path.Join("/api/partners", p.Args.Name)

	if err := update(w, partner); err != nil {
		return err
	}

	name := p.Args.Name
	if partner.Name != nil && *partner.Name != "" {
		name = *partner.Name
	}

	fmt.Fprintf(w, "The partner %q was successfully updated.\n", name)

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

func (p *PartnerAuthorize) Execute([]string) error { return p.execute(stdOutput) }
func (p *PartnerAuthorize) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/authorize/%s/%s", p.Args.Partner,
		p.Args.Rule, p.Args.Direction)

	return authorize(w, "partner", p.Args.Partner, p.Args.Rule, p.Args.Direction)
}

// ######################## REVOKE ##########################

type PartnerRevoke struct {
	Args struct {
		Partner   string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (p *PartnerRevoke) Execute([]string) error { return p.execute(stdOutput) }
func (p *PartnerRevoke) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/revoke/%s/%s", p.Args.Partner,
		p.Args.Rule, p.Args.Direction)

	return revoke(w, "partner", p.Args.Partner, p.Args.Rule, p.Args.Direction)
}
