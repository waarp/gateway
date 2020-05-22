package main

import (
	"fmt"
	"io"
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
	Cert      struct {
		Args struct {
			Server string `required:"yes" positional-arg-name:"server" description:"The server's name"`
		} `positional-args:"yes"`
		certificateCommand
	} `command:"cert" description:"Manage a server's certificates"`
}

func displayServer(w io.Writer, server *rest.OutServer) {
	send := strings.Join(server.AuthorizedRules.Sending, ", ")
	recv := strings.Join(server.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, orange(bold("● Server", server.Name)))
	fmt.Fprintln(w, orange("    Protocol:     "), server.Protocol)
	fmt.Fprintln(w, orange("    Root:         "), server.Root)
	fmt.Fprintln(w, orange("    Configuration:"), string(server.ProtoConfig))
	fmt.Fprintln(w, orange("    Authorized rules"))
	fmt.Fprintln(w, bold("    ├─  Sending:"), send)
	fmt.Fprintln(w, bold("    └─Reception:"), recv)
}

// ######################## GET ##########################

type serverGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *serverGet) Execute([]string) error {
	path := admin.APIPath + rest.ServersPath + "/" + s.Args.Name

	server := &rest.OutServer{}
	if err := get(path, server); err != nil {
		return err
	}
	displayServer(getColorable(), server)
	return nil
}

// ######################## ADD ##########################

type serverAdd struct {
	Name        string `required:"yes" short:"n" long:"name" description:"The server's name"`
	Protocol    string `required:"yes" short:"p" long:"protocol" description:"The server's protocol"`
	Root        string `short:"r" long:"root" description:"The server's root directory"`
	ProtoConfig string `short:"c" long:"config" description:"The server's configuration in JSON" default:"{}" default-mask:"-"`
}

func (s *serverAdd) Execute([]string) error {
	server := &rest.InServer{
		Name:        s.Name,
		Protocol:    s.Protocol,
		Root:        s.Root,
		ProtoConfig: []byte(s.ProtoConfig),
	}
	path := admin.APIPath + rest.ServersPath

	if err := add(path, server); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The server", bold(server.Name), "was successfully added.")
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
	addr, err := agentListURL(rest.ServersPath, &s.listOptions, s.SortBy, s.Protocols)
	if err != nil {
		return err
	}

	body := map[string][]rest.OutServer{}
	if err := list(addr, &body); err != nil {
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
	Name        string `short:"n" long:"name" description:"The server's name"`
	Protocol    string `short:"p" long:"protocol" description:"The server's protocol"`
	Root        string `short:"r" long:"root" description:"The server's root directory"`
	ProtoConfig string `short:"c" long:"config" description:"The server's configuration in JSON"`
}

func (s *serverUpdate) Execute([]string) error {
	server := rest.InServer{
		Name:        s.Name,
		Protocol:    s.Protocol,
		Root:        s.Root,
		ProtoConfig: []byte(s.ProtoConfig),
	}
	path := admin.APIPath + rest.ServersPath + "/" + s.Args.Name

	if err := update(path, server); err != nil {
		return err
	}
	name := s.Args.Name
	if server.Name != "" {
		name = server.Name
	}
	fmt.Fprintln(getColorable(), "The server", bold(name), "was successfully updated.")
	return nil
}

// ######################## AUTHORIZE ##########################

type serverAuthorize struct {
	Args struct {
		Server string `required:"yes" positional-arg-name:"server" description:"The server's name"`
		Rule   string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (s *serverAuthorize) Execute([]string) error {
	path := admin.APIPath + rest.ServersPath + "/" + s.Args.Server +
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
	path := admin.APIPath + rest.ServersPath + "/" + s.Args.Server +
		"/revoke/" + s.Args.Rule

	return revoke(path, "server", s.Args.Server, s.Args.Rule)
}
