package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

func displayLocalAgent(w io.Writer, agent rest.OutLocalAgent) {
	var config bytes.Buffer
	_ = json.Indent(&config, agent.ProtoConfig, "    ", "  ")
	fmt.Fprintf(w, "\033[97;1m● %s\033[0m (ID %v)\n", agent.Name, agent.ID)
	fmt.Fprintf(w, "  \033[97m-Protocol     :\033[0m \033[33m%s\033[0m\n", agent.Protocol)
	fmt.Fprintf(w, "  \033[97m-Root         :\033[0m \033[33m%s\033[0m\n", agent.Root)
	fmt.Fprintf(w, "  \033[97m-Configuration:\033[0m \033[37m%s\033[0m\n", config.String())
}

// ######################## GET ##########################

type serverGetCommand struct{}

func (s *serverGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing server ID")
	}

	res := rest.OutLocalAgent{}
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + args[0]

	if err := getCommand(&res, conn); err != nil {
		return err
	}

	displayLocalAgent(getColorable(), res)

	return nil
}

// ######################## ADD ##########################

type serverAddCommand struct {
	Name        string `required:"true" short:"n" long:"name" description:"The server's name"`
	Protocol    string `required:"true" short:"p" long:"protocol" description:"The server's protocol" choice:"sftp"`
	Root        string `required:"true" short:"r" long:"root" description:"The server's root directory"`
	ProtoConfig string `long:"config" description:"The server's configuration in JSON"`
}

func (s *serverAddCommand) Execute([]string) error {
	if s.ProtoConfig == "" {
		s.ProtoConfig = "{}"
	}
	newAgent := rest.InLocalAgent{
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

func (s *serverListCommand) Execute([]string) error {
	conn, err := agentListURL(rest.LocalAgentsPath, &s.listOptions, s.SortBy, s.Protocols)
	if err != nil {
		return err
	}

	res := map[string][]rest.OutLocalAgent{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	agents := res["localAgents"]
	if len(agents) > 0 {
		fmt.Fprintf(w, "\033[33;1mLocal agents:\033[0m\n")
		for _, server := range agents {
			displayLocalAgent(w, server)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo local agents found\033[0m")
	}

	return nil
}

// ######################## UPDATE ##########################

type serverUpdateCommand struct {
	Name        string `short:"n" long:"name" description:"The server's name"`
	Protocol    string `short:"p" long:"protocol" description:"The server's protocol" choice:"sftp"`
	Root        string `short:"r" long:"root" description:"The server's root directory"`
	ProtoConfig string `long:"config" description:"The server's configuration in JSON"`
}

func (s *serverUpdateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing server ID")
	}

	newAgent := rest.InLocalAgent{
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

	_, err = updateCommand(newAgent, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The server n°\033[33m%s\033[0m was successfully updated\n", args[0])

	return nil
}
