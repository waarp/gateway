package main

import (
	"context"
	"fmt"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
)

func initImportExport(configFile string, verbose []bool) (*database.DB, error) {
	config, err := conf.LoadServerConfig(configFile)
	if err != nil {
		return nil, err
	}

	logConf := conf.LogConfig{LogTo: "/dev/null"}
	switch len(verbose) {
	case 0:
	case 1:
		logConf = conf.LogConfig{LogTo: "stderr", Level: "WARNING"}
	case 2:
		logConf = conf.LogConfig{LogTo: "stderr", Level: "INFO"}
	default:
		logConf = conf.LogConfig{LogTo: "stderr", Level: "DEBUG"}
	}
	if err := log.InitBackend(logConf); err != nil {
		return nil, err
	}

	db := &database.DB{Conf: config}

	err = db.Start()
	if err != nil {
		return nil, err
	}

	return db, nil
}

type importCommand struct {
	ConfigFile string   `short:"c" long:"config" required:"true" description:"The configuration file to use"`
	File       string   `short:"s" long:"source" description:"The data file to import"`
	Target     []string `short:"t" long:"target" default:"all" choice:"rules" choice:"servers" choice:"partners" choice:"all" description:"Limit the import to a subset of data. Can be repeated to import multiple subsets."`
	Dry        bool     `short:"d" long:"dry-run" description:"Do not make any changes, but simulate the import of the file"`
	Verbose    []bool   `short:"v" long:"verbose" description:"Show verbose debug information. Can be repeated to increase verbosity"`
}

func (i *importCommand) Execute([]string) error {
	db, err := initImportExport(i.ConfigFile, i.Verbose)
	if err != nil {
		return fmt.Errorf("error at init: %w", err)
	}
	defer func() { _ = db.Stop(context.Background()) }()

	f := os.Stdin
	if i.File != "" {
		f, err = os.Open(i.File)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
	}

	if err := backup.ImportData(db, f, i.Target, i.Dry); err != nil {
		return fmt.Errorf("error at import: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Import successful.")

	return nil
}
