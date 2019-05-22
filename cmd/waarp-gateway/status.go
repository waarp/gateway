package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"

	"code.waarp.fr/waarp/gateway-ng/pkg/admin"
	"code.waarp.fr/waarp/gateway-ng/pkg/tk/service"
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

func showStatus(statuses admin.Statuses) {
	var errors = make([]string, 0)
	var actives = make([]string, 0)
	var offlines = make([]string, 0)

	fmt.Println("\033[30;1;4mWaarp-Gateway services :\033[0m")
	fmt.Println()
	for name, status := range statuses {
		switch status.State {
		case service.Running.Name():
			actives = append(actives, name)
		case service.Error.Name():
			errors = append(errors, name)
		default:
			offlines = append(offlines, name)
		}
	}

	sort.Strings(errors)
	sort.Strings(actives)
	sort.Strings(offlines)

	for _, name := range errors {
		fmt.Println("[\033[31;1mError\033[0m]   \033[1m" + name +
			"\033[0m : " + statuses[name].Reason)
	}
	for _, name := range actives {
		fmt.Println("[\033[32;1mActive\033[0m]  \033[1m" + name + "\033[0m")
	}
	for _, name := range offlines {
		fmt.Println("[\033[37;1mOffline\033[0m] \033[1m" + name + "\033[0m")
	}
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s *statusCommand) Execute(_ []string) error {
	res, err := s.makeRequest()
	if err != nil {
		return err
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var statuses = make(admin.Statuses)
	err = json.Unmarshal(body, &statuses)
	if err != nil {
		return err
	}
	showStatus(statuses)

	return nil
}
