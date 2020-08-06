package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
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

func displayPartner(w io.Writer, partner *rest.OutPartner) {
	send := strings.Join(partner.AuthorizedRules.Sending, ", ")
	recv := strings.Join(partner.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, orange(bold("● Partner", partner.Name)))
	fmt.Fprintln(w, orange("    Protocol:     "), partner.Protocol)
	fmt.Fprintln(w, orange("    Configuration:"), string(partner.ProtoConfig))
	fmt.Fprintln(w, orange("    Authorized rules"))
	fmt.Fprintln(w, bold("    ├─Sending:  "), send)
	fmt.Fprintln(w, bold("    └─Reception:"), recv)
}

// ######################## ADD ##########################

type partnerAdd struct {
	Name        string `required:"yes" short:"n" long:"name" description:"The partner's name"`
	Protocol    string `required:"yes" short:"p" long:"protocol" description:"The partner's protocol"`
	ProtoConfig string `short:"c" long:"config" description:"The partner's configuration in JSON" default:"{}" default-mask:"-"`
}

func (p *partnerAdd) Execute([]string) error {
	partner := rest.InPartner{
		Name:        p.Name,
		Protocol:    p.Protocol,
		ProtoConfig: json.RawMessage(p.ProtoConfig),
	}
	path := admin.APIPath + rest.PartnersPath

	if err := add(path, partner); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The partner", bold(partner.Name), "was successfully added.")
	return nil
}

// ######################## LIST ##########################

type partnerList struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (p *partnerList) Execute([]string) error {
	addr, err := agentListURL(rest.PartnersPath, &p.listOptions, p.SortBy, p.Protocols)
	if err != nil {
		return err
	}

	body := map[string][]rest.OutPartner{}
	if err := list(addr, &body); err != nil {
		return err
	}

	partners := body["partners"]
	w := getColorable()
	if len(partners) > 0 {
		fmt.Fprintln(w, bold("Partners:"))
		for _, p := range partners {
			partner := p
			displayPartner(w, &partner)
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
	path := admin.APIPath + rest.PartnersPath + "/" + p.Args.Name

	partner := &rest.OutPartner{}
	if err := get(path, partner); err != nil {
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
	path := admin.APIPath + rest.PartnersPath + "/" + p.Args.Name

	if err := remove(path); err != nil {
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
	Name        string `short:"n" long:"name" description:"The partner's name"`
	Protocol    string `short:"p" long:"protocol" description:"The partner's protocol'"`
	ProtoConfig string `short:"c" long:"config" description:"The partner's configuration in JSON"`
}

func (p *partnerUpdate) Execute([]string) error {
	partner := rest.InPartner{
		Name:        p.Name,
		Protocol:    p.Protocol,
		ProtoConfig: json.RawMessage(p.ProtoConfig),
	}
	path := admin.APIPath + rest.PartnersPath + "/" + p.Args.Name

	if err := update(path, partner); err != nil {
		return err
	}
	name := p.Args.Name
	if partner.Name != "" {
		name = partner.Name
	}
	fmt.Fprintln(getColorable(), "The partner", bold(name), "was successfully updated.")
	return nil
}

// ######################## AUTHORIZE ##########################

type partnerAuthorize struct {
	Args struct {
		Partner   string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (p *partnerAuthorize) Execute([]string) error {
	path := admin.APIPath + rest.PartnersPath + "/" + p.Args.Partner +
		"/authorize/" + p.Args.Rule + "/" + p.Args.Direction

	return authorize(path, "partner", p.Args.Partner, p.Args.Rule, p.Args.Direction)
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
	path := admin.APIPath + rest.PartnersPath + "/" + p.Args.Partner +
		"/revoke/" + p.Args.Rule + "/" + p.Args.Direction

	return revoke(path, "partner", p.Args.Partner, p.Args.Rule, p.Args.Direction)
}
