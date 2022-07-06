// Package wgd contains the code for the gateway service executable and all its
// sub-commands.
package wgd

import (
	"fmt"

	"github.com/jessevdk/go-flags"
)

type Command interface {
	flags.Commander
}

// InitParser initializes the given parser with the given options and subcommands.
func InitParser(parser *flags.Parser, data any) error {
	_, err := parser.AddGroup("Waarp-Gatewayd", "", data)
	if err != nil {
		return fmt.Errorf("failed to initialize the command parser: %w", err)
	}

	return nil
}

// Main parses & executes the waarp-gatewayd command using the given parser.
func Main(parser *flags.Parser, args []string) error {
	if _, err := parser.ParseArgs(args); err != nil && !flags.WroteHelp(err) {
		return fmt.Errorf("failed to parse the command arguments: %w", err)
	}

	return nil
}
