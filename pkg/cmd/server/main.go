// Package wgd contains the code for the gateway service executable and all its
// sub-commands.
package wgd

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

type Command interface {
	flags.Commander
}

// InitParser initializes the given parser with the given options and subcommands.
func InitParser(parser *flags.Parser, data any) {
	_, err := parser.AddGroup("Waarp-Gatewayd", "", data)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// Main parses & executes the waarp-gatewayd command using the given parser.
func Main(parser *flags.Parser) {
	if _, err := parser.Parse(); err != nil && !flags.WroteHelp(err) {
		os.Exit(1)
	}
}
