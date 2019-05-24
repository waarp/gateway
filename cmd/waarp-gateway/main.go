package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

var parser = flags.NewNamedParser("waarp-gateway", flags.HelpFlag|flags.PassDoubleDash)

func main() {

	_, err := parser.Parse()

	if err != nil {
		if flags.WroteHelp(err) {
			parser.WriteHelp(os.Stdout)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
