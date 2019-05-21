package main

import "fmt"

var status statusCommand

// init adds the 'status' command to the program arguments parser
func init() {
	_, err := parser.AddCommand("status", "Show gateway status",
		"Displays the status of the queried waarp-gatewayd instance.",
		&status)
	if err != nil {
		panic(err.Error())
	}
}

// statusCommand regroups all the Options of the 'status' command
type statusCommand struct {
	Address string `required:"true" short:"a" long:"address" description:"The address of the waarp-gatewayd server to query"`
	Port    uint16 `required:"true" short:"p" long:"port" description:"The port of the waarp-gatewayd server"`
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s *statusCommand) Execute(_ []string) error {
	if s.Address == "" {
		return fmt.Errorf("missing address")
	} else if s.Port == 0 {
		return fmt.Errorf("missing port")
	} else {
		fmt.Printf("Requesting status to %s\n", s.Address)
		return nil
	}
}
