package main

import (
	"fmt"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
)

func initImportExport(configFile string) (*database.DB, error) {
	config, err := conf.LoadServerConfig(configFile)
	if err != nil {
		return nil, err
	}

	err = log.InitBackend(conf.LogConfig{
		LogTo: "stdout",
		Level: "DEBUG",
	})
	if err != nil {
		return nil, err
	}

	database := &database.DB{Conf: config}

	err = database.Start()
	if err != nil {
		return nil, err
	}

	return database, nil
}

type importCommand struct {
	ConfigFile string `short:"c" long:"config" required:"true" description:"The configuration file to use"`
	File       string `short:"s" long:"source" required:"true" description:"The data file to import"`
	Target     string `short:"t" long:"target" default:"all" description:"Limit the import to a subset of data. Available options are 'rules' for the transfer rules, 'servers' for local servers and accounts, 'partners' for remote partners and accounts, or 'all' for all data. Several groups can be given separated by ','"`
	Dry        bool   `short:"d" long:"dry-run" description:"Do not make any changes, but simulate the import of the file"`
}

func (i *importCommand) Execute([]string) error {
	database, err := initImportExport(i.ConfigFile)
	if err != nil {
		return fmt.Errorf("error at init: %w", err)
	}

	importFile, err := os.Open(i.File)
	if err != nil {
		return err
	}

	defer func() { _ = importFile.Close() }()

	if err := backup.ImportData(database, importFile, i.Target, i.Dry); err != nil {
		return fmt.Errorf("error at import: %w", err)
	}

	return nil
}
