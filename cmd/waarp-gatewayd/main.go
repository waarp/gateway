package main

import (
	"os"
	"fmt"

	flags "github.com/jessevdk/go-flags"

	"code.waarp.fr/waarp/gateway-ng/pkg/conf"
)
type options struct {
	ConfigFile string `short:"c" long:"config" description:"the configuration file"`
	Update     bool   `short:"u" long:"update"`
	Create     bool   `short:"n" long:"create"`
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

	s := NewWG(config)
	if err := s.Start(); err != nil {
		fmt.Printf("ERROR: %v\n", err.Error())
	}
}
