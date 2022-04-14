package wg

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

type partnerCommand struct {
	Get       partnerGet       `command:"get" description:"Retrieve a partner's information"`
	Add       partnerAdd       `command:"add" description:"Add a new partner"`
	List      partnerList      `command:"list" description:"List the known partners"`
	Delete    partnerDelete    `command:"delete" description:"Delete a partner"`
	Update    partnerUpdate    `command:"update" description:"Update an existing partner"`
	Authorize partnerAuthorize `command:"authorize" description:"Give a partner permission to use a rule"`
	Revoke    partnerRevoke    `command:"revoke" description:"Revoke a partner's permission to use a rule"`
	Cert      struct {
		Args struct {
			Partner string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
		} `positional-args:"yes"`
		certificateCommand
	} `command:"cert" description:"Manage an partner's certificates"`
}

func displayPartner(w io.Writer, partner *api.OutPartner) {
	send := strings.Join(partner.AuthorizedRules.Sending, ", ")
	recv := strings.Join(partner.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, orange(bold("● Partner", partner.Name)))
	fmt.Fprintln(w, orange("    Protocol:     "), partner.Protocol)
	fmt.Fprintln(w, orange("    Address:      "), partner.Address)
	fmt.Fprintln(w, orange("    Configuration:"), string(partner.ProtoConfig))
	fmt.Fprintln(w, orange("    Authorized rules"))
	fmt.Fprintln(w, bold("    ├─Sending:  "), send)
	fmt.Fprintln(w, bold("    └─Reception:"), recv)
}

// ######################## ADD ##########################

//nolint:lll // struct tags for command line arguments can be long
type partnerAdd struct {
	Name        string `required:"yes" short:"n" long:"name" description:"The partner's name"`
	Protocol    string `required:"yes" short:"p" long:"protocol" description:"The partner's protocol"`
	Address     string `required:"yes" short:"a" long:"address" description:"The partner's [address:port]"`
	ProtoConfig string `required:"yes" short:"c" long:"config" description:"The partner's configuration in JSON" default:"{}"`
}

func (p *partnerAdd) Execute([]string) error {
	partner := api.InPartner{
		Name:        &p.Name,
		Protocol:    &p.Protocol,
		Address:     &p.Address,
		ProtoConfig: json.RawMessage(p.ProtoConfig),
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
type partnerList struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (p *partnerList) Execute([]string) error {
	agentListURL("/api/partners", &p.listOptions, p.SortBy, p.Protocols)

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

type partnerGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
}

func (p *partnerGet) Execute([]string) error {
	addr.Path = path.Join("/api/partners", p.Args.Name)

	partner := &api.OutPartner{}
	if err := get(partner); err != nil {
		return err
	}

	displayPartner(getColorable(), partner)

	return nil
}

// ######################## DELETE ##########################

type partnerDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
}

func (p *partnerDelete) Execute([]string) error {
	addr.Path = path.Join("/api/partners", p.Args.Name)

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The partner", bold(p.Args.Name), "was successfully deleted.")

	return nil
}

// ######################## UPDATE ##########################

type partnerUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
	Name        *string            `short:"n" long:"name" description:"The partner's name"`
	Protocol    *string            `short:"p" long:"protocol" description:"The partner's protocol'"`
	Address     *string            `short:"a" long:"address" description:"The partner's [address:port]"`
	ProtoConfig map[string]confVal `short:"c" long:"config" description:"The partner's configuration in JSON"`
}

func (p *partnerUpdate) Execute([]string) error {
	conf, err := json.Marshal(p.ProtoConfig)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	partner := &api.InPartner{
		Name:        p.Name,
		Protocol:    p.Protocol,
		Address:     p.Address,
		ProtoConfig: conf,
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
type partnerAuthorize struct {
	Args struct {
		Partner   string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction" choice:"send" choice:"receive"`
	} `positional-args:"yes"`
}

func (p *partnerAuthorize) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/authorize/%s/%s", p.Args.Partner,
		p.Args.Rule, p.Args.Direction)

	return authorize("partner", p.Args.Partner, p.Args.Rule, p.Args.Direction)
}

// ######################## REVOKE ##########################

type partnerRevoke struct {
	Args struct {
		Partner   string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (p *partnerRevoke) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/revoke/%s/%s", p.Args.Partner,
		p.Args.Rule, p.Args.Direction)

	return revoke("partner", p.Args.Partner, p.Args.Rule, p.Args.Direction)
}
