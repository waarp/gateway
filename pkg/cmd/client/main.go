// Package wg contains the code for the administration command line interface
// executable and all its sub-commands.
package wg

import (
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/jessevdk/go-flags"
)

//nolint:gochecknoglobals // global var is used by design
var (
	in          io.Reader = os.Stdin
	out         io.Writer = os.Stdout
	commandLine options
	addr        *url.URL
)

//nolint:gochecknoinits // init is required here
func init() {
	addr = &url.URL{}
}

type options struct {
	addrOpt
	Status  statusCommand  `command:"status" description:"Show the status of the gateway services"`
	Server  serverCommand  `command:"server" description:"Manage the local servers"`
	Partner partnerCommand `command:"partner" description:"Manage the remote partners"`
	Account struct {
		Local  localAccountCommand  `command:"local" description:"Manage a server's accounts"`
		Remote remoteAccountCommand `command:"remote" description:"Manage a partner's accounts"`
	} `command:"account" description:"Manage the accounts"`
	History  historyCommand  `command:"history" description:"Manage the transfer history"`
	Transfer transferCommand `command:"transfer" description:"Manage the running transfers"`
	Rule     ruleCommand     `command:"rule" description:"Manage the transfer rules"`
	User     userCommand     `command:"user" description:"Manage the gateway users"`
	Override overrideCommand `command:"override" description:"Manage the node's setting overrides"`
}

// Main parses & executes the waarp-gateway command using the given parser.
func Main(parser *flags.Parser) {
	_, err := parser.AddGroup("Commands", "", &commandLine)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	_, err = parser.AddCommand("version", "Print version and exit",
		"Print version and exit", versionCommand{})
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	_, err = parser.Parse()

	if err != nil && !flags.WroteHelp(err) {
		os.Exit(1)
	}
}
