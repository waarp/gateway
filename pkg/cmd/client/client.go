package wg

import (
	"fmt"
	"io"
	"os"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func DisplayClient(w io.Writer, client *api.OutClient) {
	f := NewFormatter(w)
	defer f.Render()

	displayClient(f, client)
}

func displayClient(f *Formatter, client *api.OutClient) {
	f.Title("Client %q", client.Name)
	f.Indent()

	defer f.UnIndent()

	f.Value("Protocol:", client.Protocol)

	if client.LocalAddress == "" {
		f.Empty("Local address:", "<unspecified>")
	} else {
		f.Value("Local address:", client.LocalAddress)
	}

	displayProtoConfig(f, client.ProtoConfig)
}

// ######################## ADD ##########################

//nolint:lll // struct tags for command line arguments can be long
type ClientAdd struct {
	Name         string             `required:"yes" short:"n" long:"name" description:"The client's name"`
	Protocol     string             `required:"yes" short:"p" long:"protocol" description:"The partner's protocol"`
	LocalAddress string             `short:"a" long:"local-address" description:"The client's local address [address:port]"`
	ProtoConfig  map[string]confVal `short:"c" long:"config" description:"The client's configuration, in key:val format. Can be repeated."`
}

func (c *ClientAdd) Execute([]string) error { return c.execute(os.Stdout) }
func (c *ClientAdd) execute(w io.Writer) error {
	client := map[string]any{
		"name":        c.Name,
		"protocol":    c.Protocol,
		"protoConfig": c.ProtoConfig,
	}
	optionalProperty(client, "localAddress", c.LocalAddress)

	addr.Path = "/api/clients"

	if err := add(client); err != nil {
		return err
	}

	w = makeColorable(w)
	fmt.Fprintln(w, "The client", bold(c.Name), "was successfully added.")

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type ClientList struct {
	ListOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"protocol+" choice:"protocol-" default:"name+"`
	Protocols []string `short:"p" long:"protocol" description:"Filter the clients based on the protocol they use. Can be repeated multiple times to filter multiple protocols."`
}

func (c *ClientList) Execute([]string) error { return c.execute(os.Stdout) }
func (c *ClientList) execute(w io.Writer) error {
	agentListURL("/api/clients", &c.ListOptions, c.SortBy, c.Protocols)

	var body map[string][]*api.OutClient
	if err := list(&body); err != nil {
		return err
	}

	f := NewFormatter(w)
	defer f.Render()

	if clients := body["clients"]; len(clients) > 0 {
		f.MainTitle("Clients:")

		for _, client := range clients {
			displayClient(f, client)
		}
	} else {
		f.Empty("No clients found.", nil)
	}

	return nil
}

// ######################## GET ##########################

type ClientGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientGet) Execute([]string) error { return c.execute(os.Stdout) }
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
	} `positional-args:"yes"`

	Name         string             `short:"n" long:"name" description:"The new client's name"`
	Protocol     string             `short:"p" long:"protocol" description:"The new partner's protocol"`
	LocalAddress string             `short:"a" long:"local-address" description:"The new client's local address [address:port]"`
	ProtoConfig  map[string]confVal `short:"c" long:"config" description:"The new client's configuration, in key:val format. Can be repeated."`
}

func (c *ClientUpdate) Execute([]string) error { return c.execute(os.Stdout) }
func (c *ClientUpdate) execute(w io.Writer) error {
	client := map[string]any{}
	optionalProperty(client, "name", c.Name)
	optionalProperty(client, "protocol", c.Protocol)
	optionalProperty(client, "localAddress", c.LocalAddress)
	optionalProperty(client, "protoConfig", c.ProtoConfig)

	addr.Path = path.Join("/api/clients", c.Args.Name)

	if err := update(client); err != nil {
		return err
	}

	name := c.Args.Name
	if c.Name != "" {
		name = c.Name
	}

	fmt.Fprintln(makeColorable(w), "The client", bold(name),
		"was successfully updated.")

	return nil
}

// ######################## DELETE ##########################

type ClientDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientDelete) Execute([]string) error { return c.execute(os.Stdout) }
func (c *ClientDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/clients", c.Args.Name)

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(makeColorable(w), "The client", bold(c.Args.Name),
		"was successfully deleted.")

	return nil
}

// ######################## START/STOP ############################

type ClientEnable struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientEnable) Execute([]string) error { return c.execute(os.Stdout) }
func (c *ClientEnable) execute(w io.Writer) error {
	if err := exec(fmt.Sprintf("/api/clients/%s/enable", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintln(makeColorable(w), "The client", bold(c.Args.Name), "was successfully enabled.")

	return nil
}

type ClientDisable struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientDisable) Execute([]string) error { return c.execute(os.Stdout) }
func (c *ClientDisable) execute(w io.Writer) error {
	if err := exec(fmt.Sprintf("/api/clients/%s/disable", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintln(makeColorable(w), "The client", bold(c.Args.Name), "was successfully disabled.")

	return nil
}

// ######################## START/STOP ############################

type ClientStart struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientStart) Execute([]string) error { return c.execute(os.Stdout) }
func (c *ClientStart) execute(w io.Writer) error {
	if err := exec(fmt.Sprintf("/api/clients/%s/start", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintln(makeColorable(w), "The client", bold(c.Args.Name), "was successfully started.")

	return nil
}

type ClientStop struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientStop) Execute([]string) error { return c.execute(os.Stdout) }
func (c *ClientStop) execute(w io.Writer) error {
	if err := exec(fmt.Sprintf("/api/clients/%s/stop", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintln(makeColorable(w), "The client", bold(c.Args.Name), "was successfully stopped.")

	return nil
}

type ClientRestart struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The client's name"`
	} `positional-args:"yes"`
}

func (c *ClientRestart) Execute([]string) error { return c.execute(os.Stdout) }
func (c *ClientRestart) execute(w io.Writer) error {
	if err := exec(fmt.Sprintf("/api/clients/%s/restart", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintln(makeColorable(w), "The client", bold(c.Args.Name), "was successfully restarted.")

	return nil
}
