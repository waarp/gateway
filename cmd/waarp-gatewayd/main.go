package main

import (
	"github.com/jessevdk/go-flags"

	wgd "code.waarp.fr/apps/gateway/gateway/pkg/cmd/server"
)

//nolint:lll // tags can be long for flags
type commands struct {
	Server  wgd.ServerCommand  `command:"server" description:"Start/Create the gateway"`
	Import  wgd.ImportCommand  `command:"import" description:"Imports the data of source file into the gateway database"`
	Export  wgd.ExportCommand  `command:"export" description:"Exports the data of the gateway database to the destination file"`
	Version wgd.VersionCommand `command:"version" description:"Print version and exit"`
	Migrate wgd.MigrateCommand `command:"migrate" description:"Migrate the gateway database to a different version"`
}

func main() {
	parser := flags.NewNamedParser("waarp-gatewayd", flags.Default)

	wgd.InitParser(parser, &commands{})
	wgd.Main(parser)
}
