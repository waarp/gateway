package main

import (
	"github.com/jessevdk/go-flags"

	wgd "code.waarp.fr/apps/gateway/gateway/pkg/cmd/server"
)

func main() {
	parser := flags.NewNamedParser("waarp-gatewayd", flags.Default)

	wgd.Main(parser)
}
