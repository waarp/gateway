package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
)

type serverCommand struct {
	Get       serverGet       `command:"get" description:"Retrieve a server's information"`
	Add       serverAdd       `command:"add" description:"Add a new server"`
	Delete    serverDelete    `command:"delete" description:"Delete a server"`
	List      serverList      `command:"list" description:"List the known servers"`
	Update    serverUpdate    `command:"update" description:"Modify a server's information" long-description:"Warning: the server's root & in/out/work paths cannot be changed individually, they must be updated all at once or the old values will be lost"`
	Authorize serverAuthorize `command:"authorize" description:"Give a server permission to use a rule"`
	Revoke    serverRevoke    `command:"revoke" description:"Revoke a server's permission to use a rule"`
	Cert      struct {
		Args struct {
			Server string `required:"yes" positional-arg-name:"server" description:"The server's name"`
		} `positional-args:"yes"`
		certificateCommand
	} `command:"cert" description:"Manage a server's certificates"`
}

func displayServer(w io.Writer, server *api.OutServer) {
	send := strings.Join(server.AuthorizedRules.Sending, ", ")
	recv := strings.Join(server.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, orange(bold("● Server", server.Name)))
	fmt.Fprintln(w, orange("    Protocol:      "), server.Protocol)
	fmt.Fprintln(w, orange("    Address:       "), server.Address)
	fmt.Fprintln(w, orange("    Root:          "), server.Root)
	fmt.Fprintln(w, orange("    In directory:  "), server.InDir)
	fmt.Fprintln(w, orange("    Out directory: "), server.OutDir)
	fmt.Fprintln(w, orange("    Work directory:"), server.WorkDir)
	fmt.Fprintln(w, orange("    Configuration: "), string(server.ProtoConfig))
	fmt.Fprintln(w, orange("    Authorized rules"))
	fmt.Fprintln(w, bold("    ├─Sending:  "), send)
	fmt.Fprintln(w, bold("    └─Reception:"), recv)
}

// ######################## GET ##########################

type serverGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *serverGet) Execute([]string) error {
	addr.Path = admin.APIPath + rest.ServersPath + "/" + s.Args.Name

	server := &api.OutServer{}
	if err := get(server); err != nil {
		return err
	}
	displayServer(getColorable(), server)
	return nil
}

// ######################## ADD ##########################

type serverAdd struct {
	Name        string  `required:"yes" short:"n" long:"name" description:"The server's name"`
	Protocol    string  `required:"yes" short:"p" long:"protocol" description:"The server's protocol"`
	Address     string  `required:"yes" short:"a" long:"address" description:"The server's [address:port]"`
	Root        *string `short:"r" long:"root" description:"The server's root directory"`
	InDir       *string `short:"i" long:"in" description:"The server's in directory"`
	OutDir      *string `short:"o" long:"out" description:"The server's out directory"`
	WorkDir     *string `short:"w" long:"work" description:"The server's work directory"`
	ProtoConfig string  `required:"yes" short:"c" long:"config" description:"The server's configuration in JSON"`
}

func (s *serverAdd) Execute([]string) error {
	server := &api.InServer{
		Name:        &s.Name,
		Protocol:    &s.Protocol,
		Address:     &s.Address,
		Root:        s.Root,
		InDir:       s.InDir,
		OutDir:      s.OutDir,
		WorkDir:     s.WorkDir,
		ProtoConfig: json.RawMessage(s.ProtoConfig),
	}
	addr.Path = admin.APIPath + rest.ServersPath

	if err := add(server); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The server", bold(s.Name), "was successfully added.")
	return nil
}

// ######################## DELETE ##########################

type serverDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *serverDelete) Execute([]string) error {
	path := admin.APIPath + rest.ServersPath + "/" + s.Args.Name

	if err := remove(path); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The server", bold(s.Args.Name), "was successfully deleted.")
	return nil
}

// ######################## LIST ##########################

type serverList struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+" `
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (s *serverList) Execute([]string) error {
	agentListURL(rest.ServersPath, &s.listOptions, s.SortBy, s.Protocols)

	body := map[string][]api.OutServer{}
	if err := list(&body); err != nil {
		return err
	}

	servers := body["servers"]
	w := getColorable()
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
}

// ######################## UPDATE ##########################

type serverUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
	Name        *string `short:"n" long:"name" description:"The server's name"`
	Protocol    *string `short:"p" long:"protocol" description:"The server's protocol"`
	Address     *string `short:"a" long:"address" description:"The server's [address:port]"`
	Root        *string `short:"r" long:"root" description:"The server's root directory"`
	InDir       *string `short:"i" long:"in" description:"The server's in directory"`
	OutDir      *string `short:"o" long:"out" description:"The server's out directory"`
	WorkDir     *string `short:"w" long:"work" description:"The server's work directory"`
	ProtoConfig *string `short:"c" long:"config" description:"The server's configuration in JSON"`
}

func (s *serverUpdate) Execute([]string) error {
	server := &api.InServer{
		Name:        s.Name,
		Protocol:    s.Protocol,
		Address:     s.Address,
		Root:        s.Root,
		InDir:       s.InDir,
		OutDir:      s.OutDir,
		WorkDir:     s.WorkDir,
		ProtoConfig: parseOptBytes(s.ProtoConfig),
	}
	addr.Path = admin.APIPath + rest.ServersPath + "/" + s.Args.Name

	if err := update(server); err != nil {
		return err
	}
	name := s.Args.Name
	if server.Name != nil && *server.Name != "" {
		name = *server.Name
	}
	fmt.Fprintln(getColorable(), "The server", bold(name), "was successfully updated.")
	return nil
}

// ######################## AUTHORIZE ##########################

type serverAuthorize struct {
	Args struct {
		Server    string `required:"yes" positional-arg-name:"server" description:"The server's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"  choice:"SEND" choice:"RECEIVE"`
	} `positional-args:"yes"`
}

func (s *serverAuthorize) Execute([]string) error {
	addr.Path = admin.APIPath + rest.ServersPath + "/" + s.Args.Server +
		"/authorize/" + s.Args.Rule + "/" + strings.ToLower(s.Args.Direction)

	return authorize("server", s.Args.Server, s.Args.Rule, s.Args.Direction)
}

// ######################## REVOKE ##########################

type serverRevoke struct {
	Args struct {
		Server    string `required:"yes" positional-arg-name:"server" description:"The server's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction" choice:"SEND" choice:"RECEIVE"`
	} `positional-args:"yes"`
}

func (s *serverRevoke) Execute([]string) error {
	addr.Path = admin.APIPath + rest.ServersPath + "/" + s.Args.Server +
		"/revoke/" + s.Args.Rule + "/" + strings.ToLower(s.Args.Direction)

	return revoke("server", s.Args.Server, s.Args.Rule, s.Args.Direction)
}
