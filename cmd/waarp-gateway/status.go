package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

// statusCommand regroups all the Options of the 'status' command
type statusCommand struct{}

// showStatus writes the status of the gateway services in the given
// writer with colors, using ANSI coloration codes.
func showStatus(statuses rest.Statuses) {
	var errors = make([]string, 0)
	var actives = make([]string, 0)
	var offlines = make([]string, 0)

	w := getColorable()

	fmt.Fprintln(w, "\033[97;1;4mWaarp-Gateway services:\033[0m")
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
		fmt.Fprintf(w, "[\033[31;1mError\033[0m]   \033[1m%s\033[0m (%s)\n", name, statuses[name].Reason)
	}
	for _, name := range actives {
		fmt.Fprintf(w, "[\033[32;1mActive\033[0m]  \033[1m%s\033[0m\n", name)
	}
	for _, name := range offlines {
		fmt.Fprintf(w, "[\033[37;1mOffline\033[0m] \033[1m%s\033[0m\n", name)
	}
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s *statusCommand) Execute(_ []string) error {
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.StatusPath

	req, err := http.NewRequest(http.MethodGet, conn.String(), nil)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, conn)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var statuses = make(rest.Statuses)
	if err = json.Unmarshal(body, &statuses); err != nil {
		return err
	}

	showStatus(statuses)

	return nil
}
