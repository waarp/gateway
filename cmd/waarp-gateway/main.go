package main

import (
	"os"

	"github.com/jessevdk/go-flags"
)

var parser = flags.NewNamedParser("waarp-gateway", flags.Default)

func main() {

	_, err := parser.Parse()

	if err != nil {
		os.Exit(1)
	}
}
