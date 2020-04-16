package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type serverCommand struct {
	Get       serverGet       `command:"get" description:"Retrieve a server's information"`
	Add       serverAdd       `command:"add" description:"Add a new server"`
	Delete    serverDelete    `command:"delete" description:"Delete a server"`
	List      serverList      `command:"list" description:"List the known servers"`
	Update    serverUpdate    `command:"update" description:"Modify a server's information"`
	Authorize serverAuthorize `command:"authorize" description:"Give a server permission to use a rule"`
	Revoke    serverRevoke    `command:"revoke" description:"Revoke a server's permission to use a rule"`
}

func displayServer(w io.Writer, server *rest.OutServer) {
	send := strings.Join(server.AuthorizedRules.Sending, ", ")
	recv := strings.Join(server.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, bold("● Server", server.Name))
	fmt.Fprintln(w, bold("        Protocol:"), server.Protocol)
	fmt.Fprintln(w, bold("            Root:"), server.Root)
	fmt.Fprintln(w, bold("   Configuration:"), string(server.ProtoConfig))
	fmt.Fprintln(w, bold("   Authorized rules"))
	fmt.Fprintln(w, bold("   ├─  Sending:"), send)
	fmt.Fprintln(w, bold("   └─Reception:"), recv)
}

// ######################## GET ##########################

type serverGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

//nolint:dupl
func (s *serverGet) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %s", err.Error())
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + s.Args.Name

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		server := &rest.OutServer{}
		if err := unmarshalBody(resp.Body, server); err != nil {
			return err
		}
		displayServer(w, server)
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## ADD ##########################

type serverAdd struct {
	Name        string `required:"yes" short:"n" long:"name" description:"The server's name"`
	Protocol    string `required:"yes" short:"p" long:"protocol" description:"The server's protocol"`
	Root        string `short:"r" long:"root" description:"The server's root directory"`
	ProtoConfig string `short:"c" long:"config" description:"The server's configuration in JSON" default:"{}" default-mask:"-"`
}

func (s *serverAdd) Execute([]string) error {
	server := &rest.InLocalAgent{
		Name:        s.Name,
		Protocol:    s.Protocol,
		Root:        s.Root,
		ProtoConfig: []byte(s.ProtoConfig),
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath

	resp, err := sendRequest(conn, server, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, "The server", bold(server.Name), "was successfully added.")
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## DELETE ##########################

type serverDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *serverDelete) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + s.Args.Name

	resp, err := sendRequest(conn, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Fprintln(w, "The server", bold(s.Args.Name), "was successfully deleted.")
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## LIST ##########################

type serverList struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+" `
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

//nolint:dupl
func (s *serverList) Execute([]string) error {
	conn, err := agentListURL(rest.LocalAgentsPath, &s.listOptions, s.SortBy, s.Protocols)
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
		body := map[string][]rest.OutServer{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}
		servers := body["localAgents"]
		if len(servers) > 0 {
			fmt.Fprintln(w, bold("Servers:"))
			for _, s := range servers {
				server := s
				displayServer(w, &server)
			}
		} else {
			fmt.Fprintln(w, "No servers found.")
		}
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## UPDATE ##########################

type serverUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
	Name        string `short:"n" long:"name" description:"The server's name"`
	Protocol    string `short:"p" long:"protocol" description:"The server's protocol"`
	Root        string `short:"r" long:"root" description:"The server's root directory"`
	ProtoConfig string `short:"c" long:"config" description:"The server's configuration in JSON"`
}

func (s *serverUpdate) Execute([]string) error {
	update := rest.InLocalAgent{
		Name:        s.Name,
		Protocol:    s.Protocol,
		Root:        s.Root,
		ProtoConfig: []byte(s.ProtoConfig),
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + s.Args.Name

	resp, err := sendRequest(conn, update, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, "The server", bold(update.Name), "was successfully updated.")
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

type serverAuthorize struct {
	Args struct {
		Server string `required:"yes" positional-arg-name:"server" description:"The server's name"`
		Rule   string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (s *serverAuthorize) Execute([]string) error {
	path := admin.APIPath + rest.LocalAgentsPath + "/" + s.Args.Server +
		"/authorize/" + s.Args.Rule

	return authorize(path, "server", s.Args.Server, s.Args.Rule)
}

// ######################## REVOKE ##########################

type serverRevoke struct {
	Args struct {
		Server string `required:"yes" positional-arg-name:"server" description:"The server's name"`
		Rule   string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (s *serverRevoke) Execute([]string) error {
	path := admin.APIPath + rest.LocalAgentsPath + "/" + s.Args.Server +
		"/revoke/" + s.Args.Rule

	return revoke(path, "server", s.Args.Server, s.Args.Rule)
}
