package main

import (
	"fmt"
	"net/http"

	"code.waarp.fr/waarp/gateway-ng/pkg/admin"
)

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
	Address  string `required:"true" short:"a" long:"address" description:"The address of the waarp-gatewayd server to query"`
	Username string `required:"true" short:"u" long:"username" description:"The user's name for authentication"`
	Password string `required:"true" short:"p" long:"password" description:"The user's password for authentication"`
}

// makeRequest makes a status request to the address stored in the statusCommand
// parameter, using the provided credentials. Returns the generated http.Response
// or an error.
func (s *statusCommand) makeRequest() (*http.Response, error) {
	addr := s.Address + admin.RestURI + admin.StatusURI
	req, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.Username, s.Password)
	client := http.Client{}
	return client.Do(req)
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s *statusCommand) Execute(_ []string) error {
	res, err := s.makeRequest()
	if err != nil {
		return err
	}
	body, err := readJSON(res)
	if err != nil {
		return err
	}

	fmt.Printf("Waarp-Gatewayd services status :\n%v", body)
	return nil
}
