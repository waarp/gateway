package main

import (
	"fmt"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	flags "github.com/jessevdk/go-flags"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/gatewayd"
)

type options struct {
	ConfigFile string `short:"c" long:"config" description:"The configuration file to use"`
	Update     bool   `short:"u" long:"update" description:"Updates the configuration file at the location given with --config"`
	Create     bool   `short:"n" long:"create" description:"Creates a new configuration file at the location given with --config"`
}

func main() {
	opts := options{}
	parser := flags.NewParser(&opts, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if !flags.WroteHelp(err) {
			parser.WriteHelp(os.Stderr)
		}
		return
		// TODO must handle exit codes
	}

	switch {
	case opts.Create:
		if opts.ConfigFile == "" {
			fmt.Printf("the path to the configuration file must be given with the argument --config")
			// TODO must handle exit codes
			return
		}
		if err := conf.CreateServerConfig(opts.ConfigFile); err != nil {
			fmt.Printf("ERROR: %s", err.Error())
			// TODO must handle exit codes
			return
		}
		// TODO must handle exit codes
		return

	case opts.Update:
		if opts.ConfigFile == "" {
			fmt.Printf("the path to the configuration file must be given with the argument --config")
			// TODO must handle exit codes
			return
		}
		if err := conf.UpdateServerConfig(opts.ConfigFile); err != nil {
			fmt.Printf("ERROR: %s", err.Error())
			// TODO must handle exit codes
			return
		}
		// TODO must handle exit codes
		return
	}

	config, err := conf.LoadServerConfig(opts.ConfigFile)
	if err != nil {
		fmt.Printf("ERROR: %s", err.Error())
		return
	}

	if err := log.InitBackend(config.Log); err != nil {
		fmt.Printf("ERROR: %s", err.Error())
		return
	}
	s := gatewayd.NewWG(config)
	if err := s.Start(); err != nil {
		fmt.Printf("ERROR: %v\n", err.Error())
	}
}
