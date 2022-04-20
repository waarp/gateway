// Package wg contains the code for the administration command line interface
// executable and all its sub-commands.
package wg

import (
	"fmt"
	"io"
	"os"

	"github.com/jessevdk/go-flags"
)

//nolint:gochecknoglobals // global var is used by design
var (
	in  io.Reader = os.Stdin
	out io.Writer = os.Stdout
)

// InitParser initializes the given parser with the waarp-gateway options and
// subcommands.
func InitParser(parser *flags.Parser, data any) {
	_, err := parser.AddGroup("Commands", "", data)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error())
		os.Exit(1)
	}
}

// Main parses & executes the waarp-gateway command using the given parser.
func Main(parser *flags.Parser) {
	if _, err := parser.Parse(); err != nil && !flags.WroteHelp(err) {
		os.Exit(1)
	}
}
