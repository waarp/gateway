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
	Get    serverGetCommand    `command:"get" description:"Retrieve a local agent's information"`
	Add    serverAddCommand    `command:"add" description:"Add a new local agent"`
	Delete serverDeleteCommand `command:"delete" description:"Delete a local agent"`
	List   serverListCommand   `command:"list" description:"List the known local agents"`
	Update serverUpdateCommand `command:"update" description:"Modify a local agent's information"`
}

func displayLocalAgent(w io.Writer, agent *rest.OutLocalAgent) {
	send := strings.Join(agent.AuthorizedRules.Sending, ", ")
	recv := strings.Join(agent.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, whiteBold("● Server ")+whiteBoldUL(agent.Name))
	fmt.Fprintln(w, whiteBold("  -Protocol:         ")+yellow(agent.Protocol))
	fmt.Fprintln(w, whiteBold("  -Root:             ")+white(agent.Root))
	fmt.Fprintln(w, whiteBold("  -Configuration:    ")+white(string(agent.ProtoConfig)))
	fmt.Fprintln(w, whiteBold("  -Authorized rules"))
	fmt.Fprintln(w, whiteBold("   ├─Sending:   ")+white(send))
	fmt.Fprintln(w, whiteBold("   └─Reception: ")+white(recv))
}

// ######################## GET ##########################

type serverGetCommand struct{}

//nolint:dupl
func (s *serverGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing server name")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %s", err.Error())
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + args[0]

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		server := &rest.OutLocalAgent{}
		if err := unmarshalBody(resp.Body, server); err != nil {
			return err
		}
		displayLocalAgent(w, server)
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("no server named '%s' found", args[0])
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## ADD ##########################

type serverAddCommand struct {
	Name        string `required:"true" short:"n" long:"name" description:"The server's name"`
	Protocol    string `required:"true" short:"p" long:"protocol" description:"The server's protocol"`
	Root        string `short:"r" long:"root" description:"The server's root directory"`
	ProtoConfig string `short:"c" long:"config" description:"The server's configuration in JSON" default:"{}"`
}

func (s *serverAddCommand) Execute([]string) error {
	newAgent := &rest.InLocalAgent{
		Name:        s.Name,
		Protocol:    s.Protocol,
		Root:        s.Root,
		ProtoConfig: []byte(s.ProtoConfig),
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath

	resp, err := sendRequest(conn, newAgent, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, whiteBold("The server '")+whiteBoldUL(newAgent.Name)+
			whiteBold("' was successfully added."))
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp).Error())
	}
}

// ######################## DELETE ##########################

type serverDeleteCommand struct{}

func (s *serverDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing server name")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + args[0]

	resp, err := sendRequest(conn, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Fprintln(w, whiteBold("The server '")+whiteBoldUL(args[0])+
			whiteBold("' was successfully deleted from the database."))
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("no server named '%s' found", args[0])
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## LIST ##########################

type serverListCommand struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name" choice:"protocol" default:"name"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

//nolint:dupl
func (s *serverListCommand) Execute([]string) error {
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
		body := map[string][]rest.OutLocalAgent{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}
		servers := body["localAgents"]
		if len(servers) > 0 {
			fmt.Fprintln(w, yellowBold("Servers:"))
			for _, s := range servers {
				server := s
				displayLocalAgent(w, &server)
			}
		} else {
			fmt.Fprintln(w, yellow("No servers found."))
		}
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp).Error())
	}
}

// ######################## UPDATE ##########################

type serverUpdateCommand struct {
	Name        string `short:"n" long:"name" description:"The server's name"`
	Protocol    string `short:"p" long:"protocol" description:"The server's protocol"`
	Root        string `short:"r" long:"root" description:"The server's root directory"`
	ProtoConfig string `short:"c" long:"config" description:"The server's configuration in JSON"`
}

func (s *serverUpdateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing server name")
	}

	update := rest.InLocalAgent{
		Name:        s.Name,
		Protocol:    s.Protocol,
		Root:        s.Root,
		ProtoConfig: []byte(s.ProtoConfig),
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + args[0]

	resp, err := sendRequest(conn, update, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, whiteBold("The server '")+whiteBoldUL(update.Name)+
			whiteBold("' was successfully updated."))
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return fmt.Errorf("no server named '%s' found", args[0])
	default:
		return fmt.Errorf("unexpected error: %v - %s", resp.StatusCode,
			getResponseMessage(resp).Error())
	}
}
