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

type partnerCommand struct {
	Get    partnerGetCommand    `command:"get" description:"Retrieve a remote agent's information"`
	Add    partnerAddCommand    `command:"add" description:"Add a new remote agent"`
	List   partnerListCommand   `command:"list" description:"List the known remote agents"`
	Delete partnerDeleteCommand `command:"delete" description:"Delete a remote agent"`
	Update partnerUpdateCommand `command:"update" description:"Update an existing remote agent"`
}

func displayRemoteAgent(w io.Writer, agent rest.OutRemoteAgent) {
	var config bytes.Buffer
	_ = json.Indent(&config, agent.ProtoConfig, "    ", "  ")
	fmt.Fprintf(w, "\033[97;1m● %s\033[0m (ID %v)\n", agent.Name, agent.ID)
	fmt.Fprintf(w, "  \033[97m-Protocol     :\033[0m \033[33m%s\033[0m\n", agent.Protocol)
	fmt.Fprintf(w, "  \033[97m-Configuration:\033[0m \033[37m%s\033[0m\n", config.String())
}

// ######################## ADD ##########################

type partnerAddCommand struct {
	Name        string `required:"true" short:"n" long:"name" description:"The partner's name"`
	Protocol    string `required:"true" short:"p" long:"protocol" description:"The partner's protocol" choice:"sftp" choice:"r66"`
	ProtoConfig string `long:"config" description:"The partner's configuration in JSON"`
}

func (p *partnerAddCommand) Execute([]string) error {
	if p.ProtoConfig == "" {
		p.ProtoConfig = "{}"
	}
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

	loc, err := addCommand(newAgent, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The partner \033[33m'%s'\033[0m was successfully added. "+
		"It can be consulted at the address: \033[37m%s\033[0m\n", newAgent.Name, loc)

	return nil
}

// ######################## LIST ##########################

type partnerListCommand struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name" choice:"protocol" default:"name"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (p *partnerListCommand) Execute([]string) error {
	conn, err := agentListURL(rest.RemoteAgentsPath, &p.listOptions, p.SortBy, p.Protocols)
	if err != nil {
		return err
	}

	res := map[string][]rest.OutRemoteAgent{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	agents := res["remoteAgents"]
	if len(agents) > 0 {
		fmt.Fprintf(w, "\033[33mRemote agents:\033[0m\n")
		for _, partner := range agents {
			displayRemoteAgent(w, partner)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo remote agents found\033[0m")
	}

	return nil
}

// ######################## GET ##########################

type partnerGetCommand struct{}

func (p *partnerGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing partner ID")
	}

	res := rest.OutRemoteAgent{}
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + args[0]

	if err := getCommand(&res, conn); err != nil {
		return err
	}

	displayRemoteAgent(getColorable(), res)

	return nil
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

	if err := deleteCommand(conn); err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The partner n°\033[33m%s\033[0m was successfully deleted from "+
		"the database\n", args[0])

	return nil
}

// ######################## UPDATE ##########################

type partnerUpdateCommand struct {
	Name        string `short:"n" long:"name" description:"The partner's name"`
	Protocol    string `short:"p" long:"protocol" description:"The partner's protocol'" choice:"sftp"`
	ProtoConfig string `long:"config" description:"The partner's configuration in JSON"`
}

func (p *partnerUpdateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing partner ID")
	}

	newAgent := rest.InRemoteAgent{
		Name:        p.Name,
		Protocol:    p.Protocol,
		ProtoConfig: json.RawMessage(p.ProtoConfig),
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + args[0]

	_, err = updateCommand(newAgent, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The partner n°\033[33m%s\033[0m was successfully updated\n", args[0])

	return nil
}
