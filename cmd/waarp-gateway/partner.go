package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
}

func displayPartner(w io.Writer, partner *rest.OutPartner) {
	send := strings.Join(partner.AuthorizedRules.Sending, ", ")
	recv := strings.Join(partner.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, bold("● Partner", partner.Name))
	fmt.Fprintln(w, orange("        Protocol:"), partner.Protocol)
	fmt.Fprintln(w, orange("   Configuration:"), string(partner.ProtoConfig))
	fmt.Fprintln(w, orange("   Authorized rules"))
	fmt.Fprintln(w, orange("   ├─Sending:  "), send)
	fmt.Fprintln(w, orange("   └─Reception:"), recv)
}

// ######################## ADD ##########################

type partnerAdd struct {
	Name        string `required:"yes" short:"n" long:"name" description:"The partner's name"`
	Protocol    string `required:"yes" short:"p" long:"protocol" description:"The partner's protocol"`
	ProtoConfig string `short:"c" long:"config" description:"The partner's configuration in JSON" default:"{}" default-mask:"-"`
}

func (p *partnerAdd) Execute([]string) error {
	newAgent := rest.InRemoteAgent{
		Name:        p.Name,
		Protocol:    p.Protocol,
		ProtoConfig: json.RawMessage(p.ProtoConfig),
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath

	resp, err := sendRequest(conn, newAgent, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, "The partner", bold(newAgent.Name), "was successfully added.")
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## LIST ##########################

type partnerList struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

//nolint:dupl
func (p *partnerList) Execute([]string) error {
	conn, err := agentListURL(rest.RemoteAgentsPath, &p.listOptions, p.SortBy, p.Protocols)
	if err != nil {
		return err
	}

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		body := map[string][]rest.OutPartner{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}
		partners := body["partners"]
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
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## GET ##########################

type partnerGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
}

//nolint:dupl
func (p *partnerGet) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %s", err.Error())
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + p.Args.Name

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		partner := &rest.OutPartner{}
		if err := unmarshalBody(resp.Body, partner); err != nil {
			return err
		}
		displayPartner(w, partner)
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## DELETE ##########################

type partnerDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The partner's name"`
	} `positional-args:"yes"`
}

func (p *partnerDelete) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + p.Args.Name

	resp, err := sendRequest(conn, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Fprintln(w, "The partner", bold(p.Args.Name), "was successfully deleted.")
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
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
	update := rest.InRemoteAgent{
		Name:        p.Name,
		Protocol:    p.Protocol,
		ProtoConfig: json.RawMessage(p.ProtoConfig),
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + p.Args.Name

	resp, err := sendRequest(conn, update, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, "The partner", bold(update.Name), "was successfully updated.")
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %v - %s", resp.StatusCode,
			getResponseMessage(resp).Error())
	}
}

// ######################## AUTHORIZE ##########################

type partnerAuthorize struct {
	Args struct {
		Partner string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
		Rule    string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (p *partnerAuthorize) Execute([]string) error {
	path := admin.APIPath + rest.RemoteAgentsPath + "/" + p.Args.Partner +
		"/authorize/" + p.Args.Rule

	return authorize(path, "partner", p.Args.Partner, p.Args.Rule)
}

// ######################## REVOKE ##########################

type partnerRevoke struct {
	Args struct {
		Partner string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
		Rule    string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (p *partnerRevoke) Execute([]string) error {
	path := admin.APIPath + rest.RemoteAgentsPath + "/" + p.Args.Partner +
		"/revoke/" + p.Args.Rule

	return revoke(path, "partner", p.Args.Partner, p.Args.Rule)
}
