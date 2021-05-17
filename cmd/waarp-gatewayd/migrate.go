package main

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database/migrations"
)

type migrateCommand struct {
	ConfigFile string `required:"yes" short:"c" long:"config" description:"The configuration file to use"`
	Args       struct {
		Version string `default:"latest" positional-arg-name:"version" description:"The version to which the database should be migrated"`
	} `positional-args:"yes"`
}

func (cmd *migrateCommand) Execute([]string) error {
	version := cmd.Args.Version
	if version == "" {
		version = "latest"
	}

	config, err := conf.LoadServerConfig(cmd.ConfigFile)
	if err != nil {
		return err
	}

	return migrations.Execute(&config.Database, cmd.Args.Version)
}
