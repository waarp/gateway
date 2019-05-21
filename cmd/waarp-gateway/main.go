package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
)

var parser = flags.NewNamedParser("waarp-gateway", flags.Default)

func main() {

	_, err := parser.Parse()

	if err != nil {
		if !flags.WroteHelp(err) {
			fmt.Fprintln(os.Stderr, "")
			parser.WriteHelp(os.Stderr)
			fmt.Fprintln(os.Stderr, "")
		}
		os.Exit(1)
	}
}
