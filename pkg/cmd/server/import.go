package wgd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
)

var ErrResetPipe = errors.New("cannot use -r without -s")

func MakeLogConf(verbose []bool) conf.LogConfig {
	logConf := conf.LogConfig{LogTo: "/dev/null", Level: "ERROR"}

	switch len(verbose) {
	case 0:
	case 1:
		logConf = conf.LogConfig{LogTo: "stderr", Level: "WARNING"}
	case 2: //nolint:mnd // useless here
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

	back, err2 := logging.NewLogBackend(config.Log.Level, config.Log.LogTo,
		config.Log.SyslogFacility, "waarp-gateway")
	if err2 != nil {
		return nil, nil, fmt.Errorf("cannot initialize log backend: %w", err2)
	}

	back.SetFormatter(func(record *log.Record) string {
		return fmt.Sprintf("[%-8s] %s", record.Level, record.Message)
	})

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
	ConfigFile string   `short:"c" long:"config" description:"The configuration file to use."`
	File       string   `short:"s" long:"source" description:"The data file to import. If none is given, the content will be read from the standard output."`
	Target     []string `short:"t" long:"target" default:"all" choice:"rules" choice:"servers" choice:"partners" choice:"clients" choice:"users" choice:"clouds" choice:"snmp" choice:"authorities" choice:"keys" choice:"all" description:"Limit the import to a subset of data. Can be repeated to import multiple subsets."`
	Dry        bool     `short:"d" long:"dry-run" description:"Do not make any changes, but simulate the import of the file."`
	Verbose    []bool   `short:"v" long:"verbose" description:"Show verbose debug information. Can be repeated to increase verbosity."`
	Reset      bool     `short:"r" long:"reset-before-import" description:"Empty the database tables before importing the elements from the file. Cannot be used without the -s option."`
	ForceReset bool     `long:"force-reset-before-import" description:"Empty the database tables before importing the elements from the file without confirmation prompt."`
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
	file := os.Stdin

	if i.File != "" {
		var opErr error

		file, opErr = os.Open(i.File)
		if opErr != nil {
			return fmt.Errorf("failed to open file: %w", opErr)
		}

		//nolint:gosec //close must be deferred here
		defer func() {
			if err := file.Close(); err != nil {
				logger.Warningf("Error while closing the source file: %v", err)
			}
		}()
	}

	if i.Reset && !i.ForceReset {
		if i.File == "" {
			return ErrResetPipe
		}

		var yes string

		fmt.Fprintln(os.Stdout, "You are about to reset the database prior to the import.")
		fmt.Fprintln(os.Stdout, "This operation cannot be undone. Do you wish to proceed anyway ?")
		fmt.Fprintln(os.Stdout)
		fmt.Fprint(os.Stdout, "(Type 'YES' in all caps to proceed): ")
		fmt.Fscanf(os.Stdin, "%s", &yes) //nolint:errcheck //error is handled bellow

		if yes != "YES" { //nolint:goconst //the other instances are for different commands
			fmt.Fprintln(os.Stderr, "Import aborted.")

			return nil
		}
	}

	reset := i.Reset || i.ForceReset

	if err := backup.ImportData(db, file, i.Target, i.Dry, reset); err != nil {
		return fmt.Errorf("error at import: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Import successful.")

	return nil
}

//nolint:lll // tags can be long for flags
type RestoreHistCommand struct {
	ConfigFile string `short:"c" long:"config" description:"The configuration file to use"`
	File       string `short:"s" long:"source" required:"true" description:"The data file to import."`
	Dry        bool   `short:"d" long:"dry-run" description:"Do not make any changes, but simulate the import of the file"`
	Verbose    []bool `short:"v" long:"verbose" description:"Show verbose debug information. Can be repeated to increase verbosity"`
}

func (r *RestoreHistCommand) Execute([]string) error {
	var yes string

	fmt.Fprintln(os.Stdout, "You are about to restore a transfer history backup.")
	fmt.Fprintln(os.Stdout, "The restored history will replace the existing one.")
	fmt.Fprintln(os.Stdout, "This means that ALL the existing history entries will be lost.")
	fmt.Fprintln(os.Stdout, "This operation cannot be undone. Do you wish to proceed anyway ?")
	fmt.Fprintln(os.Stdout)
	fmt.Fprint(os.Stdout, "(Type 'YES' in all caps to proceed): ")

	if _, err := fmt.Fscanf(os.Stdin, "%s", &yes); yes != "YES" || err != nil {
		fmt.Fprintln(os.Stderr, "Import aborted.")

		return nil
	}

	db, logger, initErr := initImportExport(r.ConfigFile, r.Verbose)
	if initErr != nil {
		return fmt.Errorf("error at init: %w", initErr)
	}

	defer func() { _ = db.Stop(context.Background()) }() //nolint:errcheck // cannot handle the error

	f, opErr := os.Open(r.File)
	if opErr != nil {
		return fmt.Errorf("failed to open file: %w", opErr)
	}

	defer func() { _ = f.Close() }() //nolint:errcheck,gosec // Close() must be deferred

	if err := backup.ImportHistory(db, f, r.Dry); err != nil {
		return fmt.Errorf("error at restore: %w", err)
	}

	logger.Info("History successfully restored.")

	return nil
}
