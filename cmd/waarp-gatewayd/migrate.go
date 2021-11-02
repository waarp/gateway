package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/jessevdk/go-flags"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

//nolint:lll // tags can be long for flags
type migrateCommand struct {
	ConfigFile    string         `required:"yes" short:"c" long:"config" description:"The configuration file to use"`
	DryRun        bool           `short:"d" long:"dry-run" description:"Simulate the migration but does not commit the changes"`
	ExtractToFile flags.Filename `short:"f" long:"file" description:"Writes the migration commands into a file instead of sending them to the database"`
	Args          struct {
		Version string `default:"latest" positional-arg-name:"version" description:"The version to which the database should be migrated"`
	} `positional-args:"yes"`

	// Can be used in testing to specify the index from which the migration should
	// start (useful for testing untagged versions).
	FromIndex int `hidden:"yes" long:"from-index" default:"-1"`
}

func (cmd *migrateCommand) Execute([]string) error {
	config, err := conf.LoadServerConfig(cmd.ConfigFile)
	if err != nil {
		return fmt.Errorf("cannot load server config: %w", err)
	}

	var out io.Writer
	if cmd.DryRun {
		out = ioutil.Discard
	}

	if cmd.ExtractToFile != "" {
		file, err := os.OpenFile(cmd.ConfigFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("cannot open destination file: %w", err)
		}

		defer func() { _ = file.Close() }() //nolint:errcheck // cannot handle the error

		out = file
	}

	if err := migrations.Execute(&config.Database, cmd.Args.Version, cmd.FromIndex, out); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
