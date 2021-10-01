package main

import (
	"context"
	"fmt"
	"os"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup"
)

//nolint:lll // tags can be long for flags
type exportCommand struct {
	ConfigFile string   `short:"c" long:"config" description:"The configuration file to use"`
	File       string   `short:"f" long:"file" description:"The destination file. If none is given, the content of the export will be written to the standard output"`
	Target     []string `short:"t" long:"target" default:"all" choice:"rules" choice:"servers" choice:"partners" choice:"all" description:"Limit the export to a subset of data. Can be repeated to export multiple subsets."`
	Verbose    []bool   `short:"v" long:"verbose" description:"Show verbose debug information. Can be repeated to increase verbosity"`
}

func (e *exportCommand) Execute([]string) error {
	db, err := initImportExport(e.ConfigFile, e.Verbose)
	if err != nil {
		return fmt.Errorf("error at init: %w", err)
	}

	defer func() { _ = db.Stop(context.Background()) }() //nolint:errcheck // cannot handle the error

	f := os.Stdout
	if e.File != "" {
		f, err = os.OpenFile(e.File, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open the output file: %w", err)
		}

		defer func() { _ = f.Close() }() //nolint:errcheck // cannot handle the error
	}

	if err := backup.ExportData(db, f, e.Target); err != nil {
		return fmt.Errorf("error during export: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Export successful.")

	return nil
}
