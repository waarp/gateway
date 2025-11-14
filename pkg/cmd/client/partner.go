package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

//nolint:gochecknoglobals //a global var is required here
var Partner string

type PartnerArg struct{}

func (*PartnerArg) UnmarshalFlag(value string) error {
	Partner = value

	return nil
}

func displayPartners(w io.Writer, partners []*api.OutPartner) error {
	Style0.Printf(w, "=== Partners ===")
	for _, partner := range partners {
		if err := displayPartner(w, partner); err != nil {
			return err
		}
	}

	return nil
}

func displayPartner(w io.Writer, partner *api.OutPartner) error {
	Style1.Printf(w, "Partner %q", partner.Name)
	Style22.PrintL(w, "Protocol", partner.Protocol)
	Style22.PrintL(w, "Address", partner.Address)
	Style22.PrintL(w, "Credentials",
		withDefault(join(partner.Credentials), none))

	displayProtoConfig(w, partner.ProtoConfig)
	displayAuthorizedRules(w, partner.AuthorizedRules)

	return nil
}

// ######################## ADD ##########################

//nolint:lll // struct tags for command line arguments can be long
type PartnerAdd struct {
	Name        string             `required:"yes" short:"n" long:"name" description:"The partner's name" json:"name,omitempty"`
	Protocol    string             `required:"yes" short:"p" long:"protocol" description:"The partner's protocol" json:"protocol,omitempty"`
	Address     string             `required:"yes" short:"a" long:"address" description:"The partner's [address:port]" json:"address,omitempty"`
	ProtoConfig map[string]confVal `short:"c" long:"config" description:"The partner's configuration, in key:val format. Can be repeated." json:"protoConfig,omitempty"`
}

func (p *PartnerAdd) Execute([]string) error { return execute(p) }
func (p *PartnerAdd) execute(w io.Writer) error {
	addr.Path = "/api/partners"

	if _, err := add(w, p); err != nil {
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

func (p *PartnerList) Execute([]string) error { return execute(p) }

//nolint:dupl //duplicate is for a different type, best keep separate
func (p *PartnerList) execute(w io.Writer) error {
	agentListURL("/api/partners", &p.ListOptions, p.SortBy, p.Protocols)

	body := map[string][]*api.OutPartner{}
	if err := list(&body); err != nil {
		return err
	}

	if partners := body["partners"]; len(partners) > 0 {
		return outputObject(w, partners, &p.OutputFormat, displayPartners)
	}

	fmt.Fprintln(w, "No partners found.")

	return nil
}

// ######################## GET ##########################

type PartnerGet struct {
	OutputFormat

	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
}

func (p *PartnerGet) Execute([]string) error { return execute(p) }
func (p *PartnerGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/partners", p.Args.Name)

	partner := &api.OutPartner{}
	if err := get(partner); err != nil {
		return err
	}

	return outputObject(w, partner, &p.OutputFormat, displayPartner)
}

// ######################## DELETE ##########################

type PartnerDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
}

func (p *PartnerDelete) Execute([]string) error { return execute(p) }
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
	} `positional-args:"yes" json:"-"`

	Name        *string             `short:"n" long:"name" description:"The partner's name" json:"name,omitempty"`
	Protocol    *string             `short:"p" long:"protocol" description:"The partner's protocol'" json:"protocol,omitempty"`
	Address     *string             `short:"a" long:"address" description:"The partner's [address:port]" json:"address,omitempty"`
	ProtoConfig *map[string]confVal `short:"c" long:"config" description:"The partner's configuration, in key:val format. Can be repeated." json:"protoConfig,omitempty"`
}

func (p *PartnerUpdate) Execute([]string) error { return execute(p) }
func (p *PartnerUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/partners", p.Args.Name)

	if err := update(w, p); err != nil {
		return err
	}

	name := p.Args.Name
	if p.Name != nil && *p.Name != "" {
		name = *p.Name
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

func (p *PartnerAuthorize) Execute([]string) error { return execute(p) }
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

func (p *PartnerRevoke) Execute([]string) error { return execute(p) }
func (p *PartnerRevoke) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/revoke/%s/%s", p.Args.Partner,
		p.Args.Rule, p.Args.Direction)

	return revoke(w, "partner", p.Args.Partner, p.Args.Rule, p.Args.Direction)
}
