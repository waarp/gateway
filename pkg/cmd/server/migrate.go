package wgd

import (
	"fmt"
	"io"
	"os"

	"code.waarp.fr/lib/log"
	"github.com/jessevdk/go-flags"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/migrations"
)

//nolint:lll // tags can be long for flags
type MigrateCommand struct {
	ConfigFile    flags.Filename `required:"yes" short:"c" long:"config" description:"The configuration file to use"`
	DryRun        bool           `short:"d" long:"dry-run" description:"Simulate the migration but does not commit the changes"`
	ExtractToFile flags.Filename `short:"f" long:"file" description:"Writes the migration commands into a file instead of sending them to the database"`
	Verbose       []bool         `short:"v" long:"verbose" description:"Show verbose debug information. Can be repeated to increase verbosity"`
	Args          struct {
		Version string `default:"latest" positional-arg-name:"version" description:"The version to which the database should be migrated"`
	} `positional-args:"yes"`

	// Can be used in testing to specify the index from which the migration should
	// start (useful for testing untagged versions).
	FromIndex int `hidden:"yes" long:"from-index" default:"-1"`
}

func (cmd *MigrateCommand) Execute([]string) error {
	config, logger, err := initMigration(string(cmd.ConfigFile), cmd.Verbose)
	if err != nil {
		return err
	}

	var out io.Writer
	if cmd.DryRun {
		out = io.Discard
	}

	if cmd.ExtractToFile != "" {
		file, err := os.OpenFile(string(cmd.ExtractToFile), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("cannot open destination file: %w", err)
		}

		//nolint:gosec //close must be deferred here
		defer func() {
			if err := file.Close(); err != nil {
				logger.Warning("Error while closing the script output file: %s", err)
			}
		}()

		out = file
	}

	if err := migrations.Execute(&config.Database, logger, cmd.Args.Version,
		cmd.FromIndex, out); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func initMigration(configFile string, verbose []bool) (*conf.ServerConfig, *log.Logger, error) {
	config, err := conf.ParseServerConfig(configFile)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot load server config: %w", err)
	}

	config.Log = MakeLogConf(verbose)

	back, err2 := conf.NewLogBackend(config.Log.Level, config.Log.LogTo,
		config.Log.SyslogFacility, "waarp-gateway")
	if err2 != nil {
		return nil, nil, fmt.Errorf("cannot initialize log backend: %w", err2)
	}

	conf.GlobalConfig = *config

	return config, back.NewLogger("Migration"), nil
}
