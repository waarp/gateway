package main

import (
	"io"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mattn/go-colorable"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	in, out *os.File
	auth    connectionOptions
)

type commands struct {
	Status      statusCommand      `command:"status" description:"Show the status of the gateway services"`
	Server      serverCommand      `command:"server" description:"Manage the gateway's local servers"`
	Partner     partnerCommand     `command:"partner" description:"Manage the gateway's remote partners"`
	Access      accessCommand      `command:"access" description:"Manage the gateway's local accounts"`
	Account     accountCommand     `command:"account" description:"Manage the gateway's remote accounts"`
	Certificate certificateCommand `command:"certificate" description:"Manage the gateway's certificates"`
	Transfer    transferCommand    `command:"transfer" description:"Manage the gateway's transfers"`
}

type connectionOptions struct {
	DSN string `short:"a" long:"address" required:"true" description:"The connection parameters of the gateway interface. Must have the following form: user@address:port"`
}

func main() {
	in = os.Stdin
	out = os.Stdout

	cmd := flags.NewNamedParser("waarp-gateway", flags.Default)
	_, err := cmd.AddGroup("Connection Options", "", &auth)
	if err != nil {
		os.Exit(1)
	}
	if _, err := cmd.AddGroup("Database Commands", "", &commands{}); err != nil {
		os.Exit(1)
	}

	_, err = cmd.Parse()

	if err != nil && !flags.WroteHelp(err) {
		os.Exit(2)
	}
}

func getColorable() io.Writer {
	if terminal.IsTerminal(int(out.Fd())) {
		return colorable.NewColorable(out)
	}
	return colorable.NewNonColorable(out)
}
