package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/jessevdk/go-flags"
)

var (
	in          = os.Stdin
	out         = os.Stdout
	commandLine options
	addr        *url.URL
	insecure    bool
)

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
}

func main() {
	parser := flags.NewNamedParser("waarp-gateway", flags.Default)
	_, err := parser.AddGroup("Commands", "", &commandLine)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}

	_, err = parser.AddCommand("version", "Print version and exit",
		"Print version and exit", &versionCommand{})
	if err != nil {
		panic(err.Error())
	}

	_, err = parser.Parse()

	if err != nil && !flags.WroteHelp(err) {
		os.Exit(1)
	}
}
