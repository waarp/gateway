package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

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

func displayServer(agent rest.OutAgent) {
	w := getColorable()

	var config bytes.Buffer
	_ = json.Indent(&config, agent.ProtoConfig, "  ", "  ")
	fmt.Fprintf(w, "\033[37;1;4mLocal agent n°%v:\033[0m\n", agent.ID)
	fmt.Fprintf(w, "          \033[37mName:\033[0m \033[37;1m%s\033[0m\n", agent.Name)
	fmt.Fprintf(w, "      \033[37mProtocol:\033[0m \033[37;1m%s\033[0m\n", agent.Protocol)
	fmt.Fprintf(w, " \033[37mConfiguration:\033[0m \033[37m%s\033[0m\n", config.String())
}

// ######################## GET ##########################

type serverGetCommand struct{}

func (s *serverGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing server ID")
	}

	res := rest.OutAgent{}
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + args[0]

	if err := getCommand(&res, conn); err != nil {
		return err
	}

	displayServer(res)

	return nil
}

// ######################## ADD ##########################

type serverAddCommand struct {
	Name        string `required:"true" short:"n" long:"name" description:"The server's name"`
	Protocol    string `required:"true" short:"p" long:"protocol" description:"The server's protocol'" choice:"sftp"`
	ProtoConfig string `long:"config" description:"The server's configuration in JSON"`
}

func (s *serverAddCommand) Execute(_ []string) error {
	if s.ProtoConfig == "" {
		s.ProtoConfig = "{}"
	}
	newAgent := rest.InAgent{
		Name:        s.Name,
		Protocol:    s.Protocol,
		ProtoConfig: []byte(s.ProtoConfig),
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath

	loc, err := addCommand(newAgent, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The server \033[33m'%s'\033[0m was successfully added. "+
		"It can be consulted at the address: \033[37m%s\033[0m\n", newAgent.Name, loc)

	return nil
}

// ######################## DELETE ##########################

type serverDeleteCommand struct{}

func (s *serverDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing server ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + args[0]

	if err := deleteCommand(conn); err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The server n°\033[33m%s\033[0m was successfully deleted from "+
		"the database\n", args[0])

	return nil
}

// ######################## LIST ##########################

type serverListCommand struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name" choice:"protocol" default:"name"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (s *serverListCommand) Execute(_ []string) error {
	conn, err := agentListURL(rest.LocalAgentsPath, &s.listOptions, s.SortBy, s.Protocols)
	if err != nil {
		return err
	}

	res := map[string][]rest.OutAgent{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	agents := res["localAgents"]
	if len(agents) > 0 {
		fmt.Fprintf(w, "\033[33mLocal agents:\033[0m\n")
		for _, server := range agents {
			displayServer(server)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo local agents found\033[0m")
	}

	return nil
}

// ######################## UPDATE ##########################

type serverUpdateCommand struct {
	Name        string `short:"n" long:"name" description:"The server's name"`
	Protocol    string `short:"p" long:"protocol" description:"The server's protocol'" choice:"sftp"`
	ProtoConfig string `long:"config" description:"The server's configuration in JSON"`
}

func (s *serverUpdateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing server ID")
	}

	newAgent := rest.InAgent{
		Name:        s.Name,
		Protocol:    s.Protocol,
		ProtoConfig: []byte(s.ProtoConfig),
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + args[0]

	_, err = updateCommand(newAgent, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The server n°\033[33m%s\033[0m was successfully updated\n", args[0])

	return nil
}
