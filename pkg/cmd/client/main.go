// Package wg contains the code for the administration command line interface
// executable and all its sub-commands.
package wg

import (
	"fmt"

	"github.com/jessevdk/go-flags"
)

// InitParser initializes the given parser with the waarp-gateway options and
// subcommands.
func InitParser(parser *flags.Parser, data any) error {
	_, err := parser.AddGroup("Commands", "", data)
	if err != nil {
		return fmt.Errorf("failed to initialize the command parser: %w", err)
	}

	return nil
}

// Main parses & executes the waarp-gateway command using the given parser.
func Main(parser *flags.Parser, args []string) error {
	if _, err := parser.ParseArgs(args); err != nil && !flags.WroteHelp(err) {
		return fmt.Errorf("failed to parse the command arguments: %w", err)
	}

	return nil
}
