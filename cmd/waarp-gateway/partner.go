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
	Get    partnerGetCommand    `command:"get" description:"Retrieve a remote agent's information"`
	Add    partnerAddCommand    `command:"add" description:"Add a new remote agent"`
	List   partnerListCommand   `command:"list" description:"List the known remote agents"`
	Delete partnerDeleteCommand `command:"delete" description:"Delete a remote agent"`
	Update partnerUpdateCommand `command:"update" description:"Update an existing remote agent"`
}

func displayRemoteAgent(w io.Writer, agent *rest.OutRemoteAgent) {
	send := strings.Join(agent.AuthorizedRules.Sending, ", ")
	recv := strings.Join(agent.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, whiteBold("● Partner ")+whiteBoldUL(agent.Name))
	fmt.Fprintln(w, whiteBold("  -Protocol:         ")+yellow(agent.Protocol))
	fmt.Fprintln(w, whiteBold("  -Configuration:    ")+white(string(agent.ProtoConfig)))
	fmt.Fprintln(w, whiteBold("  -Authorized rules"))
	fmt.Fprintln(w, whiteBold("   ├─Sending:   ")+white(send))
	fmt.Fprintln(w, whiteBold("   └─Reception: ")+white(recv))
}

// ######################## ADD ##########################

type partnerAddCommand struct {
	Name        string `required:"true" short:"n" long:"name" description:"The partner's name"`
	Protocol    string `required:"true" short:"p" long:"protocol" description:"The partner's protocol"`
	ProtoConfig string `short:"c" long:"config" description:"The partner's configuration in JSON" default:"{}"`
}

func (p *partnerAddCommand) Execute([]string) error {
	newAgent := rest.InRemoteAgent{
		Name:        p.Name,
		Protocol:    p.Protocol,
		ProtoConfig: json.RawMessage(p.ProtoConfig),
	}

	conn, err := url.Parse(auth.DSN)
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
		fmt.Fprintln(w, whiteBold("The partner '")+whiteBoldUL(newAgent.Name)+
			whiteBold("' was successfully added."))
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp).Error())
	}
}

// ######################## LIST ##########################

type partnerListCommand struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name" choice:"protocol" default:"name"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

//nolint:dupl
func (p *partnerListCommand) Execute([]string) error {
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
		body := map[string][]rest.OutRemoteAgent{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}
		partners := body["remoteAgents"]
		if len(partners) > 0 {
			fmt.Fprintln(w, yellowBold("Partners:"))
			for _, p := range partners {
				partner := p
				displayRemoteAgent(w, &partner)
			}
		} else {
			fmt.Fprintln(w, yellow("No partners found."))
		}
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp).Error())
	}
}

// ######################## GET ##########################

type partnerGetCommand struct{}

//nolint:dupl
func (p *partnerGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing partner name")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %s", err.Error())
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + args[0]

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		partner := &rest.OutRemoteAgent{}
		if err := unmarshalBody(resp.Body, partner); err != nil {
			return err
		}
		displayRemoteAgent(w, partner)
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("no partner named '%s' found", args[0])
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## DELETE ##########################

type partnerDeleteCommand struct{}

func (p *partnerDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing partner ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + args[0]

	resp, err := sendRequest(conn, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Fprintln(w, whiteBold("The partner '")+whiteBoldUL(args[0])+
			whiteBold("' was successfully deleted from the database."))
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("no partner named '%s' found", args[0])
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## UPDATE ##########################

type partnerUpdateCommand struct {
	Name        string `short:"n" long:"name" description:"The partner's name"`
	Protocol    string `short:"p" long:"protocol" description:"The partner's protocol'"`
	ProtoConfig string `short:"c" long:"config" description:"The partner's configuration in JSON"`
}

func (p *partnerUpdateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing partner ID")
	}

	update := rest.InRemoteAgent{
		Name:        p.Name,
		Protocol:    p.Protocol,
		ProtoConfig: json.RawMessage(p.ProtoConfig),
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + args[0]

	resp, err := sendRequest(conn, update, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, whiteBold("The partner '")+whiteBoldUL(update.Name)+
			whiteBold("' was successfully updated."))
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return fmt.Errorf("no partner named '%s' found", args[0])
	default:
		return fmt.Errorf("unexpected error: %v - %s", resp.StatusCode,
			getResponseMessage(resp).Error())
	}
}
