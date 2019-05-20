package main

import (
	"fmt"
)

var status statusCommand

// init adds the 'status' command to the program arguments parser
func init() {
	parser.AddCommand("status", "Show gateway status",
		"Displays the status of the queried waarp-gatewayd instance.",
		&status)
}

// statusCommand regroups all the Options of the 'status' command
type statusCommand struct {
	Address string `required:"true" short:"a" long:"address" description:"The address of the waarp-gatewayd server to query"`
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s *statusCommand) Execute(args []string) error {
	fmt.Printf("Requesting status to %s\n", s.Address)
	return nil
}
