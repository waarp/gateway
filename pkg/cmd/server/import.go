package wgd

import (
	"context"
	"fmt"
	"os"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

func MakeLogConf(verbose []bool) conf.LogConfig {
	logConf := conf.LogConfig{LogTo: "/dev/null", Level: "ERROR"}

	switch len(verbose) {
	case 0:
	case 1:
		logConf = conf.LogConfig{LogTo: "stderr", Level: "WARNING"}
	case 2: //nolint:gomnd // useless here
		logConf = conf.LogConfig{LogTo: "stderr", Level: "INFO"}
	default:
		logConf = conf.LogConfig{LogTo: "stderr", Level: "DEBUG"}
	}

	return logConf
}

func initImportExport(configFile string, verbose []bool) (*database.DB, *log.Logger, error) {
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
	db := &database.DB{}

	err = db.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("cannot start database: %w", err)
	}

	return db, back.NewLogger("Import/Export"), nil
}

//nolint:lll // tags can be long for flags
type ImportCommand struct {
	ConfigFile string   `short:"c" long:"config" description:"The configuration file to use"`
	File       string   `short:"s" long:"source" description:"The data file to import"`
	Target     []string `short:"t" long:"target" default:"all" choice:"rules" choice:"servers" choice:"partners" choice:"all" description:"Limit the import to a subset of data. Can be repeated to import multiple subsets."`
	Dry        bool     `short:"d" long:"dry-run" description:"Do not make any changes, but simulate the import of the file"`
	Verbose    []bool   `short:"v" long:"verbose" description:"Show verbose debug information. Can be repeated to increase verbosity"`
}

func (i *ImportCommand) Execute([]string) error {
	db, logger, err := initImportExport(i.ConfigFile, i.Verbose)
	if err != nil {
		return fmt.Errorf("error at init: %w", err)
	}

	defer func() { _ = db.Stop(context.Background()) }() //nolint:errcheck // cannot handle the error

	return i.Run(db, logger)
}

func (i *ImportCommand) Run(db *database.DB, logger *log.Logger) error {
	f := os.Stdin

	if i.File != "" {
		var err error

		f, err = os.Open(i.File)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}

		//nolint:gosec //close must be deferred here
		defer func() {
			if err := f.Close(); err != nil {
				logger.Warning("Error while closing the source file: %s", err)
			}
		}()
	}

	if err := backup.ImportData(db, f, i.Target, i.Dry); err != nil {
		return fmt.Errorf("error at import: %w", err)
	}

	logger.Info("Import successful.")

	return nil
}
