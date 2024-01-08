package wg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var ErrMissingServerName = errors.New("the 'server' name argument is missing")

//nolint:gochecknoglobals //a global var is required here
var Server string

type ServerArg struct{}

func (*ServerArg) UnmarshalFlag(value string) error {
	Server = value

	return nil
}

func DisplayServer(w io.Writer, server *api.OutServer) {
	f := NewFormatter(w)
	defer f.Render()

	displayServer(f, server)
}

func displayServer(f *Formatter, server *api.OutServer) {
	f.Title("Server %q [%s]", server.Name, enabledStatus(server.Enabled))
	f.Indent()

	defer f.UnIndent()

	f.Value("Protocol", server.Protocol)
	f.Value("Address", server.Address)
	f.ValueCond("Root directory", server.RootDir)
	f.ValueCond("Receive directory", server.ReceiveDir)
	f.ValueCond("Send directory", server.SendDir)
	f.ValueCond("Temp receive directory", server.TmpReceiveDir)
	displayProtoConfig(f, server.ProtoConfig)
	displayAuthorizedRules(f, server.AuthorizedRules)
}

func warnServerRootDeprecated(w io.Writer) {
	fmt.Fprintln(w, "[WARNING] The '-r' ('--root') option is deprecated. "+
		"Use '--root-dir' instead.")
}

func warnServerInDeprecated(w io.Writer) {
	fmt.Fprintln(w, "[WARNING] The '-i' ('--in') option is deprecated. "+
		"Use '--receive-dir' instead.")
}

func warnServerOutDeprecated(w io.Writer) {
	fmt.Fprintln(w, "[WARNING] The '-o' ('--out') option is deprecated. "+
		"Use '--send-dir' instead.")
}

func warnServerWorkDeprecated(w io.Writer) {
	fmt.Fprintln(w, "[WARNING] The '-w' ('--work') option is deprecated. "+
		"Use '--tmp-dir' instead.")
}

// ######################## GET ##########################

type ServerGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *ServerGet) Execute([]string) error { return s.execute(stdOutput) }
func (s *ServerGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/servers", s.Args.Name)

	server := &api.OutServer{}
	if err := get(server); err != nil {
		return err
	}

	DisplayServer(w, server)

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

func (s *ServerAdd) Execute([]string) error { return s.execute(stdOutput) }
func (s *ServerAdd) execute(w io.Writer) error {
	server := &api.InServer{
		Name:          &s.Name,
		Protocol:      &s.Protocol,
		Address:       &s.Address,
		RootDir:       s.RootDir,
		ReceiveDir:    s.ReceiveDir,
		SendDir:       s.SendDir,
		TmpReceiveDir: s.TempRcvDir,
	}

	if err := utils.JSONConvert(s.ProtoConfig, &server.ProtoConfig); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if s.Root != nil {
		warnServerRootDeprecated(w)

		server.Root = s.Root
	}

	if s.InDir != nil {
		warnServerInDeprecated(w)

		server.InDir = s.InDir
	}

	if s.OutDir != nil {
		warnServerOutDeprecated(w)

		server.OutDir = s.OutDir
	}

	if s.WorkDir != nil {
		warnServerWorkDeprecated(w)

		server.WorkDir = s.WorkDir
	}

	addr.Path = "/api/servers"

	if _, err := add(w, server); err != nil {
		return err
	}

	fmt.Fprintf(w, "The server %q was successfully added.\n", s.Name)

	return nil
}

// ######################## DELETE ##########################

type ServerDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *ServerDelete) Execute([]string) error { return s.execute(stdOutput) }
func (s *ServerDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/servers", s.Args.Name)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The server %q was successfully deleted.\n", s.Args.Name)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags can be long for command line args
type ServerList struct {
	ListOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+" `
	Protocols []string `short:"p" long:"protocol" description:"Filter the agents based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (s *ServerList) Execute([]string) error { return s.execute(stdOutput) }

//nolint:dupl //duplicate is for a different type, best keep separate
func (s *ServerList) execute(w io.Writer) error {
	agentListURL("/api/servers", &s.ListOptions, s.SortBy, s.Protocols)

	body := map[string][]*api.OutServer{}
	if err := list(&body); err != nil {
		return err
	}

	if servers := body["servers"]; len(servers) > 0 {
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("Servers:")

		for _, server := range servers {
			displayServer(f, server)
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

func (s *ServerUpdate) Execute([]string) error { return s.execute(stdOutput) }
func (s *ServerUpdate) execute(w io.Writer) error {
	server := &api.InServer{
		Name:          s.Name,
		Protocol:      s.Protocol,
		Address:       s.Address,
		RootDir:       s.RootDir,
		ReceiveDir:    s.ReceiveDir,
		SendDir:       s.SendDir,
		TmpReceiveDir: s.TempRcvDir,
	}

	if err := utils.JSONConvert(s.ProtoConfig, &server.ProtoConfig); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	if s.Root != nil {
		warnServerRootDeprecated(w)

		server.Root = s.Root
	}

	if s.InDir != nil {
		warnServerInDeprecated(w)

		server.InDir = s.InDir
	}

	if s.OutDir != nil {
		warnServerOutDeprecated(w)

		server.OutDir = s.OutDir
	}

	if s.WorkDir != nil {
		warnServerWorkDeprecated(w)

		server.WorkDir = s.WorkDir
	}

	addr.Path = path.Join("/api/servers", s.Args.Name)

	if err := update(w, server); err != nil {
		return err
	}

	name := s.Args.Name
	if server.Name != nil && *server.Name != "" {
		name = *server.Name
	}

	fmt.Fprintf(w, "The server %q was successfully updated.\n", name)

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

func (s *ServerAuthorize) Execute([]string) error { return s.execute(stdOutput) }
func (s *ServerAuthorize) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/authorize/%s/%s", s.Args.Server,
		s.Args.Rule, s.Args.Direction)

	return authorize(w, "server", s.Args.Server, s.Args.Rule, s.Args.Direction)
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

func (s *ServerRevoke) Execute([]string) error { return s.execute(stdOutput) }
func (s *ServerRevoke) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/revoke/%s/%s", s.Args.Server,
		s.Args.Rule, s.Args.Direction)

	return revoke(w, "server", s.Args.Server, s.Args.Rule, s.Args.Direction)
}

// ######################## ENABLE/DISABLE ##########################

type serverEnableDisable struct {
	Args struct {
		Server string `required:"yes" positional-arg-name:"server" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *serverEnableDisable) run(w io.Writer, isEnable bool) error {
	server := s.Args.Server
	if server == "" {
		return ErrMissingServerName
	}

	handlerPath, status := rest.ServerPathEnable, "enabled"
	if !isEnable {
		handlerPath, status = rest.ServerPathDisable, "disabled"
	}

	addr.Path = strings.ReplaceAll(handlerPath, "{server}", server)

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // response should not have a body

	switch resp.StatusCode {
	case http.StatusAccepted:
		fmt.Fprintf(w, "The server %q was successfully %s.\n", server, status)

		return nil
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

type (
	ServerEnable  struct{ serverEnableDisable }
	ServerDisable struct{ serverEnableDisable }
)

func (s *ServerEnable) Execute([]string) error     { return s.execute(stdOutput) }
func (s *ServerEnable) execute(w io.Writer) error  { return s.run(w, true) }
func (s *ServerDisable) Execute([]string) error    { return s.execute(stdOutput) }
func (s *ServerDisable) execute(w io.Writer) error { return s.run(w, false) }

// ######################## START/STOP ############################

type ServerStart struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *ServerStart) Execute([]string) error { return s.execute(stdOutput) }
func (s *ServerStart) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("/api/servers/%s/start", s.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The server %q was successfully started.\n", s.Args.Name)

	return nil
}

type ServerStop struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *ServerStop) Execute([]string) error { return s.execute(stdOutput) }
func (s *ServerStop) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("/api/servers/%s/stop", s.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The server %q was successfully stopped.\n", s.Args.Name)

	return nil
}

type ServerRestart struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
}

func (s *ServerRestart) Execute([]string) error { return s.execute(stdOutput) }
func (s *ServerRestart) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("/api/servers/%s/restart", s.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The server %q was successfully restarted.\n", s.Args.Name)

	return nil
}
