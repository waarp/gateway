package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func enabledStatus(enabled bool) string {
	return utils.If(enabled, "Enabled", "Disabled")
}

func DisplayClient(w io.Writer, client *api.OutClient) {
	f := NewFormatter(w)
	defer f.Render()

	displayClient(f, client)
}

func displayClient(f *Formatter, client *api.OutClient) {
	f.Title("Client %q [%s]", client.Name, enabledStatus(client.Enabled))
	f.Indent()

	defer f.UnIndent()

	f.Value("Protocol", client.Protocol)

	if client.LocalAddress == "" {
		f.Empty("Local address", "<unspecified>")
	} else {
		f.Value("Local address", client.LocalAddress)
	}

	displayProtoConfig(f, client.ProtoConfig)
}

// ######################## ADD ##########################

//nolint:lll,tagliatelle // struct tags for command line arguments can be long
type ClientAdd struct {
	Name         string             `required:"yes" short:"n" long:"name" description:"The client's name" json:"name,omitempty"`
	Protocol     string             `required:"yes" short:"p" long:"protocol" description:"The partner's protocol" json:"protocol,omitempty"`
	LocalAddress string             `short:"a" long:"local-address" description:"The client's local address [address:port]" json:"localAddress,omitempty"`
	ProtoConfig  map[string]confVal `short:"c" long:"config" description:"The client's configuration, in key:val format. Can be repeated." json:"config,omitempty"`
}

func (c *ClientAdd) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientAdd) execute(w io.Writer) error {
	addr.Path = "/api/clients"

	if _, err := add(w, c); err != nil {
		return err
	}

	fmt.Fprintf(w, "The client %q was successfully added.\n", c.Name)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type ClientList struct {
	ListOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the clients based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (c *ClientList) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientList) execute(w io.Writer) error {
	agentListURL("/api/clients", &c.ListOptions, c.SortBy, c.Protocols)

	var body map[string][]*api.OutClient
	if err := list(&body); err != nil {
		return err
	}

	if clients := body["clients"]; len(clients) > 0 {
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("Clients:")

		for _, client := range clients {
			displayClient(f, client)
		}
	} else {
		fmt.Fprintln(w, "No clients found.")
	}

	return nil
}

// ######################## GET ##########################

type ClientGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientGet) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/clients", c.Args.Name)

	var client api.OutClient
	if err := get(&client); err != nil {
		return err
	}

	DisplayClient(w, &client)

	return nil
}

// ######################## UPDATE ##########################

//nolint:lll // struct tags for command line arguments can be long
type ClientUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The old client's name"`
	} `positional-args:"yes" json:"-"`

	Name         string             `short:"n" long:"name" description:"The new client's name" json:"name,omitempty"`
	Protocol     string             `short:"p" long:"protocol" description:"The new partner's protocol" json:"protocol,omitempty"`
	LocalAddress string             `short:"a" long:"local-address" description:"The new client's local address [address:port]" json:"localAddress,omitempty"`
	ProtoConfig  map[string]confVal `short:"c" long:"config" description:"The new client's configuration, in key:val format. Can be repeated." json:"protoConfig,omitempty"`
}

func (c *ClientUpdate) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/clients", c.Args.Name)

	if err := update(w, c); err != nil {
		return err
	}

	name := c.Args.Name
	if c.Name != "" {
		name = c.Name
	}

	fmt.Fprintf(w, "The client %q was successfully updated.\n", name)

	return nil
}

// ######################## DELETE ##########################

type ClientDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientDelete) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/clients", c.Args.Name)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The client %q was successfully deleted.\n", c.Args.Name)

	return nil
}

// ######################## START/STOP ############################

type ClientEnable struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientEnable) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientEnable) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("/api/clients/%s/enable", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The client %q was successfully enabled.\n", c.Args.Name)

	return nil
}

type ClientDisable struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientDisable) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientDisable) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("/api/clients/%s/disable", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The client %q was successfully disabled.\n", c.Args.Name)

	return nil
}

// ######################## START/STOP ############################

type ClientStart struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientStart) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientStart) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("/api/clients/%s/start", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The client %q was successfully started.\n", c.Args.Name)

	return nil
}

type ClientStop struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientStop) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientStop) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("/api/clients/%s/stop", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The client %q was successfully stopped.\n", c.Args.Name)

	return nil
}

type ClientRestart struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientRestart) Execute([]string) error { return c.execute(stdOutput) }
func (c *ClientRestart) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("/api/clients/%s/restart", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The client %q was successfully restarted.\n", c.Args.Name)

	return nil
}
