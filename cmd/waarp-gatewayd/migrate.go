package main

import (
	"io"
	"io/ioutil"
	"os"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
	"github.com/jessevdk/go-flags"
)

type migrateCommand struct {
	ConfigFile    string         `required:"yes" short:"c" long:"config" description:"The configuration file to use"`
	DryRun        bool           `short:"d" long:"dry-run" description:"Simulate the migration but does not commit the changes"`
	ExtractToFile flags.Filename `short:"f" long:"file" description:"Writes the migration commands into a file instead of sending them to the database"`
	Args          struct {
		Version string `default:"latest" positional-arg-name:"version" description:"The version to which the database should be migrated"`
	} `positional-args:"yes"`
}

func (cmd *migrateCommand) Execute([]string) error {
	config, err := conf.LoadServerConfig(cmd.ConfigFile)
	if err != nil {
		return err
	}

	var out io.Writer
	if cmd.DryRun {
		out = ioutil.Discard
	}
	if cmd.ExtractToFile != "" {
		file, err := os.OpenFile(cmd.ConfigFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer func() { _ = file.Close() }()
		out = file
	}

	return migrations.Execute(&config.Database, cmd.Args.Version, out)
}
