package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

var (
	in  = os.Stdin
	out = os.Stdout
	// Deprecated
	auth        connectionOptions
	commandLine options
)

type options struct {
	Args struct {
		Address string `required:"yes" positional-arg-name:"address" description:"The address of the gateway"`
	} `positional-args:"yes"`
	Status  statusCommand  `command:"status" description:"Show the status of the gateway services"`
	Server  serverCommand  `command:"server" description:"Manage the local servers"`
	Partner partnerCommand `command:"partner" description:"Manage the remote partners"`
}

// Deprecated
type commands struct {
	Status      statusCommand      `command:"status" description:"Show the status of the gateway services"`
	User        userCommand        `command:"user" description:"Manage the gateway's users"`
	Server      serverCommand      `command:"server" description:"Manage the gateway's local servers"`
	Partner     partnerCommand     `command:"partner" description:"Manage the gateway's remote partners"`
	Access      accessCommand      `command:"access" description:"Manage the gateway's local accounts"`
	Account     accountCommand     `command:"account" description:"Manage the gateway's remote accounts"`
	Certificate certificateCommand `command:"certificate" description:"Manage the gateway's certificates"`
	Transfer    transferCommand    `command:"transfer" description:"Manage the gateway's planned transfers"`
	History     historyCommand     `command:"history" description:"Manage the gateway's transfer history"`
	Rule        ruleCommand        `command:"rule" description:"Manage the gateway's transfer rules"`
}

// Deprecated
type connectionOptions struct {
	DSN string `short:"a" long:"address" required:"true" description:"The connection parameters of the gateway interface. Must have the following form: user@address:port"`
}

func main() {
	cmd := flags.NewNamedParser("waarp-gateway", flags.Default)
	_, err := cmd.AddGroup("Connection Options", "", &auth)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
	if _, err := cmd.AddGroup("Database Commands", "", &commands{}); err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(2)
	}

	_, err = cmd.Parse()

	if err != nil && !flags.WroteHelp(err) {
		os.Exit(3)
	}
}
