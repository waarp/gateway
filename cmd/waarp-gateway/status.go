package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/mattn/go-colorable"
	"golang.org/x/crypto/ssh/terminal"
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
type statusCommand struct{}

// requestStatus makes a status request to the address stored in the statusCommand
// parameter, using the provided credentials, and returns the body of the response.
// If the server did not reply, or if the response code was not '200 - OK', then
// the function returns an error.
func (s *statusCommand) requestStatus(in *os.File, out *os.File) (admin.Statuses, error) {
	addr := auth.Address + admin.RestURI + admin.StatusURI

	req, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}

	res, err := executeRequest(req, auth.Username, in, out)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var statuses = make(admin.Statuses)
	if err = json.Unmarshal(body, &statuses); err != nil {
		return nil, err
	}
	return statuses, nil
}

// showStatus writes the status of the gateway services in the given
// writer with colors, using ANSI coloration codes.
func showStatus(f *os.File, statuses admin.Statuses) {
	var errors = make([]string, 0)
	var actives = make([]string, 0)
	var offlines = make([]string, 0)

	var w io.Writer
	if terminal.IsTerminal(int(f.Fd())) {
		w = colorable.NewColorable(f)
	} else {
		w = colorable.NewNonColorable(f)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "\033[97;1;4mWaarp-Gateway services :\033[0m")
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
		fmt.Fprintln(w, "[\033[31;1mError\033[0m]   \033[1m"+name+
			"\033[0m : "+statuses[name].Reason)
	}
	for _, name := range actives {
		fmt.Fprintln(w, "[\033[32;1mActive\033[0m]  \033[1m"+name+"\033[0m")
	}
	for _, name := range offlines {
		fmt.Fprintln(w, "[\033[37;1mOffline\033[0m] \033[1m"+name+"\033[0m")
	}
	fmt.Fprintln(w)
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s *statusCommand) Execute(_ []string) error {
	statuses, err := s.requestStatus(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	showStatus(os.Stdout, statuses)

	return nil
}
