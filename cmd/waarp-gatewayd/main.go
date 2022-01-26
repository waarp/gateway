package main

import (
	"errors"
	"fmt"
	"os"

	flags "github.com/jessevdk/go-flags"
)

//nolint:lll // tags can be long for flags
type options struct {
	Server  serverCommand  `command:"server" description:"Start/Create the gateway"`
	Import  importCommand  `command:"import" description:"Imports the data of source file into the gateway database"`
	Export  exportCommand  `command:"export" description:"Exports the data of the gateway database to the destination file"`
	Version versionCommand `command:"version" description:"Print version and exit"`
	Migrate migrateCommand `command:"migrate" description:"Migrate the gateway database to a different version"`
}

func main() {
	opts := options{}
	parser := flags.NewParser(&opts, flags.Default)

	if _, err := parser.Parse(); err != nil {
		var err2 *flags.Error
		if ok := errors.As(err, &err2); ok && !flags.WroteHelp(err2) {
			fmt.Fprintln(os.Stderr)
			parser.WriteHelp(os.Stderr)
		}

		os.Exit(1)
	}
}
