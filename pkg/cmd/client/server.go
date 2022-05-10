package wg

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

//nolint:gochecknoglobals //a global var is required here
var Server string

type ServerArg struct{}

func (*ServerArg) UnmarshalFlag(value string) error {
	Server = value

	return nil
}

func displayServer(w io.Writer, server *api.OutServer) {
	send := strings.Join(server.AuthorizedRules.Sending, ", ")
	recv := strings.Join(server.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, orange(bold("● Server", server.Name)))
	fmt.Fprintln(w, orange("    Protocol:              "), server.Protocol)
	fmt.Fprintln(w, orange("    Address:               "), server.Address)
	fmt.Fprintln(w, orange("    Root directory:        "), server.RootDir)
	fmt.Fprintln(w, orange("    Receive directory:     "), server.ReceiveDir)
	fmt.Fprintln(w, orange("    Send directory:        "), server.SendDir)
	fmt.Fprintln(w, orange("    Temp receive directory:"), server.TmpReceiveDir)
	fmt.Fprintln(w, orange("    Configuration:         "), string(server.ProtoConfig))
	fmt.Fprintln(w, orange("    Authorized rules"))
	fmt.Fprintln(w, bold("    ├─Sending:  "), send)
	fmt.Fprintln(w, bold("    └─Reception:"), recv)
}

// ######################## GET ##########################

type ServerGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *ServerGet) Execute([]string) error {
	addr.Path = path.Join("/api/servers", s.Args.Name)

	server := &api.OutServer{}
	if err := get(server); err != nil {
		return err
	}

	displayServer(getColorable(), server)

	return nil
}

// ######################## ADD ##########################

//nolint:lll // struct tags can be long for command line args
type ServerAdd struct {
	Name        string             `required:"yes" short:"n" long:"name" description:"The server's name"`
	Protocol    string             `required:"yes" short:"p" long:"protocol" description:"The server's protocol"`
	Address     string             `required:"yes" short:"a" long:"address" description:"The server's [address:port]"`
	RootDir     *string            `long:"root-dir" description:"The server's local root directory"`
	ReceiveDir  *string            `long:"receive-dir" description:"The server's local directory for received files"`
	SendDir     *string            `long:"send-dir" description:"The server's local directory for files to send"`
	TempRcvDir  *string            `long:"tmp-dir" description:"The server's local temporary directory for incoming files"`
	ProtoConfig map[string]confVal `short:"c" long:"config" description:"The server's configuration, in key:val format. Can be repeated."`

	// Deprecated options
	Root    *string `short:"r" long:"root" description:"[DEPRECATED] The server's root directory"`     // Deprecated: replaced by RootDir
	InDir   *string `short:"i" long:"in" description:"[DEPRECATED] The server's local in directory"`   // Deprecated: replaced by ReceiveDir
	OutDir  *string `short:"o" long:"out" description:"[DEPRECATED] The server's local out directory"` // Deprecated: replaced by SendDir
	WorkDir *string `short:"w" long:"work" description:"[DEPRECATED] The server's work directory"`     // Deprecated: replaced by TempRcvDir
}

func (s *ServerAdd) Execute([]string) error {
	conf, err := json.Marshal(s.ProtoConfig)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	server := &api.InServer{
		Name:          &s.Name,
		Protocol:      &s.Protocol,
		Address:       &s.Address,
		RootDir:       s.RootDir,
		ReceiveDir:    s.ReceiveDir,
		SendDir:       s.SendDir,
		TmpReceiveDir: s.TempRcvDir,
		ProtoConfig:   conf,
	}

	if s.Root != nil {
		fmt.Fprintln(out, "[WARNING] The '-r' ('--root') option is deprecated. "+
			"Use '--root-dir' instead.")

		server.Root = s.Root
	}

	if s.InDir != nil {
		fmt.Fprintln(out, "[WARNING] The '-i' ('--in') option is deprecated. "+
			"Use '--receive-dir' instead.")

		server.InDir = s.InDir
	}

	if s.OutDir != nil {
		fmt.Fprintln(out, "[WARNING] The '-o' ('--out') option is deprecated. "+
			"Use '--send-dir' instead.")

		server.OutDir = s.OutDir
	}

	if s.WorkDir != nil {
		fmt.Fprintln(out, "[WARNING] The '-w' ('--work') option is deprecated. "+
			"Use '--tmp-dir' instead.")

		server.WorkDir = s.WorkDir
	}

	addr.Path = "/api/servers"

	if err := add(server); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The server", bold(s.Name), "was successfully added.")

	return nil
}

// ######################## DELETE ##########################

type ServerDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *ServerDelete) Execute([]string) error {
	addr.Path = path.Join("/api/servers", s.Args.Name)

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The server", bold(s.Args.Name), "was successfully deleted.")

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags can be long for command line args
type ServerList struct {
	ListOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+" `
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

//nolint:dupl // hard to factorize
func (s *ServerList) Execute([]string) error {
	agentListURL("/api/servers", &s.ListOptions, s.SortBy, s.Protocols)

	body := map[string][]api.OutServer{}
	if err := list(&body); err != nil {
		return err
	}

	w := getColorable() //nolint:ifshort // decrease readability

	if servers := body["servers"]; len(servers) > 0 {
		fmt.Fprintln(w, bold("Servers:"))

		for i := range servers {
			server := servers[i]
			displayServer(w, &server)
		}
	} else {
		fmt.Fprintln(w, "No servers found.")
	}

	return nil
}

// ######################## UPDATE ##########################

//nolint:lll // struct tags can be long for command line args
type ServerUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
	Name        *string            `short:"n" long:"name" description:"The server's name"`
	Protocol    *string            `short:"p" long:"protocol" description:"The server's protocol"`
	Address     *string            `short:"a" long:"address" description:"The server's [address:port]"`
	RootDir     *string            `long:"root-dir" description:"The server's local root directory"`
	ReceiveDir  *string            `long:"receive-dir" description:"The server's local directory for received files"`
	SendDir     *string            `long:"send-dir" description:"The server's local directory for files to send"`
	TempRcvDir  *string            `long:"tmp-dir" description:"The server's local temporary directory for incoming files"`
	ProtoConfig map[string]confVal `short:"c" long:"config" description:"The server's configuration in JSON"`

	// Deprecated options
	Root    *string `short:"r" long:"root" description:"[DEPRECATED] The server's root directory"`     // Deprecated: replaced by RootDir
	InDir   *string `short:"i" long:"in" description:"[DEPRECATED] The server's local in directory"`   // Deprecated: replaced by ReceiveDir
	OutDir  *string `short:"o" long:"out" description:"[DEPRECATED] The server's local out directory"` // Deprecated: replaced by SendDir
	WorkDir *string `short:"w" long:"work" description:"[DEPRECATED] The server's work directory"`     // Deprecated: replaced by TempRcvDir
}

func (s *ServerUpdate) Execute([]string) error {
	conf, err := json.Marshal(s.ProtoConfig)
	if err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	server := &api.InServer{
		Name:          s.Name,
		Protocol:      s.Protocol,
		Address:       s.Address,
		RootDir:       s.RootDir,
		ReceiveDir:    s.ReceiveDir,
		SendDir:       s.SendDir,
		TmpReceiveDir: s.TempRcvDir,
		ProtoConfig:   conf,
	}

	if s.Root != nil {
		fmt.Fprintln(out, "[WARNING] The '-r' ('--root') option is deprecated. "+
			"Use '--root-dir' instead.")

		server.Root = s.Root
	}

	if s.InDir != nil {
		fmt.Fprintln(out, "[WARNING] The '-i' ('--in') option is deprecated. "+
			"Use '--receive-dir' instead.")

		server.InDir = s.InDir
	}

	if s.OutDir != nil {
		fmt.Fprintln(out, "[WARNING] The '-o' ('--out') option is deprecated. "+
			"Use '--send-dir' instead.")

		server.OutDir = s.OutDir
	}

	if s.WorkDir != nil {
		fmt.Fprintln(out, "[WARNING] The '-w' ('--work') option is deprecated. "+
			"Use '--tmp-dir' instead.")

		server.WorkDir = s.WorkDir
	}

	addr.Path = path.Join("/api/servers", s.Args.Name)

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

//nolint:lll // struct tags can be long for command line args
type ServerAuthorize struct {
	Args struct {
		Server    string `required:"yes" positional-arg-name:"server" description:"The server's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"  choice:"send" choice:"receive"`
	} `positional-args:"yes"`
}

func (s *ServerAuthorize) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/authorize/%s/%s", s.Args.Server,
		s.Args.Rule, s.Args.Direction)

	return authorize("server", s.Args.Server, s.Args.Rule, s.Args.Direction)
}

// ######################## REVOKE ##########################

//nolint:lll // struct tags can be long for command line args
type ServerRevoke struct {
	Args struct {
		Server    string `required:"yes" positional-arg-name:"server" description:"The server's name"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction" choice:"send" choice:"receive"`
	} `positional-args:"yes"`
}

func (s *ServerRevoke) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/revoke/%s/%s", s.Args.Server,
		s.Args.Rule, s.Args.Direction)

	return revoke("server", s.Args.Server, s.Args.Rule, s.Args.Direction)
}
