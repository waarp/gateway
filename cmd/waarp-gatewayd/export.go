package main

import (
	"fmt"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/backup"
)

type exportCommand struct {
	ConfigFile string `short:"c" long:"config" required:"true" description:"The configuration file to use"`
	File       string `short:"f" long:"file" default:"waarp-gateway-export.json" description:"The destination file"`
	Target     string `short:"t" long:"target" default:"all" description:"Limit the export to a subset of data. Available options are 'rules' for the transfer rules, 'servers' for local servers and accounts, 'partners' for remote partners and accounts, or 'all' for all data. Several groups can be given separated by ','"`
}

func (i *exportCommand) Execute([]string) error {
	database, err := initImportExport(i.ConfigFile)
	if err != nil {
		return fmt.Errorf("error at init: %s", err.Error())
	}

	f, err := os.OpenFile(i.File, os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := backup.ExportData(database, f, i.Target); err != nil {
		return fmt.Errorf("error during export: %s", err.Error())
	}

	return nil
}
