package wg

import (
	"fmt"
	"io"
	"path"
	"time"

	"github.com/gookit/color"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	TextEnabled  = "Enabled"
	TextDisabled = "Disabled"
)

func coloredEnabled(enabled bool) string {
	return utils.If(enabled,
		color.Bold.Render(TextEnabled),
		color.Gray.Render(TextDisabled),
	)
}

func displayClient(w io.Writer, client *api.OutClient) {
	Style1.Printf(w, "Client %q [%s]", client.Name, coloredEnabled(client.Enabled))
	Style22.PrintL(w, "Protocol", client.Protocol)
	Style22.PrintL(w, "Local address", withDefault(client.LocalAddress, unspecified))
	displayProtoConfig(w, client.ProtoConfig)

	if client.NbOfAttempts != 0 {
		Style22.PrintL(w, "Transfers number of attempts", client.NbOfAttempts)
		Style22.PrintL(w, "Transfers first retry delay",
			(time.Duration(client.FirstRetryDelay) * time.Second).String())
		Style22.PrintL(w, "Transfers retry increment factor", client.RetryIncrementFactor)
	}
}

// ######################## ADD ##########################

//nolint:lll,tagliatelle // struct tags for command line arguments can be long
type ClientAdd struct {
	Name                 string             `required:"yes" short:"n" long:"name" description:"The client's name" json:"name,omitempty"`
	Protocol             string             `required:"yes" short:"p" long:"protocol" description:"The partner's protocol" json:"protocol,omitempty"`
	LocalAddress         string             `short:"a" long:"local-address" description:"The client's local address [address:port]" json:"localAddress,omitempty"`
	ProtoConfig          map[string]confVal `short:"c" long:"config" description:"The client's configuration, in key:val format. Can be repeated." json:"protoConfig,omitempty"`
	NbOfAttempts         int8               `long:"nb-of-attempts" description:"The number of times a transfer will be automatically re-tried in case of failure" json:"nbOfAttempts,omitempty"`
	FirstRetryDelay      time.Duration      `long:"first-retry-delay" description:"The delay (in seconds) between the original attempt and the first automatic retry" json:"-"`
	FirstRetryDelaySec   int32              `json:"firstRetryDelay,omitempty"`
	RetryIncrementFactor float32            `long:"retry-increment-factor" description:"The factor by which the delay will be multiplied between each attempt of a transfer" json:"retryIncrementFactor,omitempty"`
}

func (c *ClientAdd) Execute([]string) error { return execute(c) }
func (c *ClientAdd) execute(w io.Writer) error {
	addr.Path = "/api/clients"

	if c.FirstRetryDelay != 0 {
		c.FirstRetryDelaySec = int32(c.FirstRetryDelay.Seconds())
	}

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

func (c *ClientList) Execute([]string) error { return execute(c) }
func (c *ClientList) execute(w io.Writer) error {
	agentListURL("/api/clients", &c.ListOptions, c.SortBy, c.Protocols)

	var body map[string][]*api.OutClient
	if err := list(&body); err != nil {
		return err
	}

	if clients := body["clients"]; len(clients) > 0 {
		Style0.Printf(w, "=== Clients ===")

		for _, client := range clients {
			displayClient(w, client)
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

func (c *ClientGet) Execute([]string) error { return execute(c) }
func (c *ClientGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/clients", c.Args.Name)

	var client api.OutClient
	if err := get(&client); err != nil {
		return err
	}

	displayClient(w, &client)

	return nil
}

// ######################## UPDATE ##########################

//nolint:lll // struct tags for command line arguments can be long
type ClientUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The old client's name"`
	} `positional-args:"yes" json:"-"`

	Name                 *string             `short:"n" long:"name" description:"The new client's name" json:"name,omitempty"`
	Protocol             *string             `short:"p" long:"protocol" description:"The new partner's protocol" json:"protocol,omitempty"`
	LocalAddress         *string             `short:"a" long:"local-address" description:"The new client's local address [address:port]" json:"localAddress,omitempty"`
	ProtoConfig          *map[string]confVal `short:"c" long:"config" description:"The new client's configuration, in key:val format. Can be repeated." json:"protoConfig,omitempty"`
	NbOfAttempts         *int8               `long:"nb-of-attempts" description:"The number of times a transfer will be automatically re-tried in case of failure" json:"nbOfAttempts,omitempty"`
	FirstRetryDelayDur   time.Duration       `long:"first-retry-delay-dur" description:"The delay (in seconds) between the original attempt and the first automatic retry" json:"-"`
	FirstRetryDelay      int32               `json:"firstRetryDelay,omitempty"`
	RetryIncrementFactor *float32            `long:"retry-increment-factor" description:"The factor by which the delay will be multiplied between each attempt of a transfer" json:"retryIncrementFactor,omitempty"`
}

func (c *ClientUpdate) Execute([]string) error { return execute(c) }
func (c *ClientUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/clients", c.Args.Name)

	if c.FirstRetryDelayDur != 0 {
		c.FirstRetryDelay = int32(c.FirstRetryDelayDur.Seconds())
	}

	if err := update(w, c); err != nil {
		return err
	}

	name := c.Args.Name
	if c.Name != nil && *c.Name != "" {
		name = *c.Name
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

func (c *ClientDelete) Execute([]string) error { return execute(c) }
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

func (c *ClientEnable) Execute([]string) error { return execute(c) }
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

func (c *ClientDisable) Execute([]string) error { return execute(c) }
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

func (c *ClientStart) Execute([]string) error { return execute(c) }
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

func (c *ClientStop) Execute([]string) error { return execute(c) }
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

func (c *ClientRestart) Execute([]string) error { return execute(c) }
func (c *ClientRestart) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("/api/clients/%s/restart", c.Args.Name)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The client %q was successfully restarted.\n", c.Args.Name)

	return nil
}
