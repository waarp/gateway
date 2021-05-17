package main

import (
	"fmt"
	"os"

	flags "github.com/jessevdk/go-flags"
)

type options struct {
	Server  serverCommand  `command:"server" description:"Start/Create the gateway"`
	Import  importCommand  `command:"import" description:"Imports the data of source file into the gateway database"`
	Export  exportCommand  `command:"export" description:"Exports the data of the gateway database to the destination file"`
	Version versionCommand `command:"version" description:"Print version and exit"`
}

func main() {
	opts := options{}
	parser := flags.NewParser(&opts, flags.Default)

	if _, err := parser.Parse(); err != nil {
		if _, ok := err.(*flags.Error); ok && !flags.WroteHelp(err) {
			fmt.Fprintln(os.Stderr)
			parser.WriteHelp(os.Stderr)
		}

		os.Exit(1)
	}
}
