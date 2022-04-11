// Package wgd contains the code for the gateway service executable and all its
// sub-commands.
package wgd

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

//nolint:lll // tags can be long for flags
type options struct {
	Server  serverCommand  `command:"server" description:"Start/Create the gateway"`
	Import  importCommand  `command:"import" description:"Imports the data of source file into the gateway database"`
	Export  exportCommand  `command:"export" description:"Exports the data of the gateway database to the destination file"`
	Version versionCommand `command:"version" description:"Print version and exit"`
	Migrate migrateCommand `command:"migrate" description:"Migrate the gateway database to a different version"`
}

// InitParser initializes the given parser with the waarp-gatewayd options and
// subcommands.
func InitParser(parser *flags.Parser) {
	_, err := parser.AddGroup("Commands", "", &options{})
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// Main parses & executes the waarp-gatewayd command using the given parser.
func Main(parser *flags.Parser) {
	if _, err := parser.Parse(); err != nil && !flags.WroteHelp(err) {
		os.Exit(1)
	}
}
