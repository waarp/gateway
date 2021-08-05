package main

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/gatewayd"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
)

type serverCommand struct {
	ConfigFile   string `short:"c" long:"config" description:"The configuration file to use"`
	Update       bool   `short:"u" long:"update" description:"Updates the configuration file at the location given with --config"`
	Create       bool   `short:"n" long:"create" description:"Creates a new configuration file at the location given with --config"`
	InstanceName string `short:"i" long:"instance" description:"The unique identifier of the instance inside a cluster"`
}

func (cmd *serverCommand) Execute([]string) error {
	switch {
	case cmd.Create:
		if cmd.ConfigFile == "" {
			return fmt.Errorf("the path to the configuration file must be given with the argument --config")
		}
		if err := conf.CreateGatewayConfig(cmd.ConfigFile, cmd.InstanceName); err != nil {
			return err
		}
		return nil

	case cmd.Update:
		if cmd.ConfigFile == "" {
			return fmt.Errorf("the path to the configuration file must be given with the argument --config")
		}
		if err := conf.UpdateGatewayConfig(cmd.ConfigFile, cmd.InstanceName); err != nil {
			return err
		}
		return nil
	}

	if err := conf.LoadGatewayConfig(cmd.ConfigFile, cmd.InstanceName); err != nil {
		return err
	}

	if err := log.InitBackend(conf.GlobalConfig.ServerConf.Log); err != nil {
		return err
	}
	s := gatewayd.NewWG()
	if err := s.Start(); err != nil {
		return err
	}
	return nil
}
