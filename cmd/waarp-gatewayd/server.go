package main

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
)

type serverCommand struct {
	ConfigFile string `short:"c" long:"config" description:"The configuration file to use"`
	Update     bool   `short:"u" long:"update" description:"Updates the configuration file at the location given with --config"`
	Create     bool   `short:"n" long:"create" description:"Creates a new configuration file at the location given with --config"`
}

func (cmd *serverCommand) Execute([]string) error {
	switch {
	case cmd.Create:
		if cmd.ConfigFile == "" {
			return fmt.Errorf("the path to the configuration file must be given with the argument --config")
		}
		if err := conf.CreateServerConfig(cmd.ConfigFile); err != nil {
			return err
		}
		return nil

	case cmd.Update:
		if cmd.ConfigFile == "" {
			return fmt.Errorf("the path to the configuration file must be given with the argument --config")
		}
		if err := conf.UpdateServerConfig(cmd.ConfigFile); err != nil {
			return err
		}
		return nil
	}

	config, err := conf.LoadServerConfig(cmd.ConfigFile)
	if err != nil {
		return err
	}

	if err := log.InitBackend(config.Log); err != nil {
		return err
	}
	s := gatewayd.NewWG(config)
	if err := s.Start(); err != nil {
		return err
	}
	return nil
}
