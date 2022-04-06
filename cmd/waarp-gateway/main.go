package main

import (
	"github.com/jessevdk/go-flags"

	wg "code.waarp.fr/apps/gateway/gateway/pkg/cmd/client"
)

func main() {
	parser := flags.NewNamedParser("waarp-gateway", flags.Default)

	wg.Main(parser)
}
