package main

import (
	"fmt"
	"io"
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
func showStatus(statuses rest.Statuses, w io.Writer) {
	var errors = make([]string, 0)
	var actives = make([]string, 0)
	var offlines = make([]string, 0)

	fmt.Fprintln(w, whiteBoldUL("Waarp-Gateway services:"))
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
		fmt.Fprintln(w, redBold("[Error]   ")+whiteBold(name)+white(" (",
			statuses[name].Reason, ")"))
	}
	for _, name := range actives {
		fmt.Fprintln(w, greenBold("[Active]  ")+whiteBold(name))
	}
	for _, name := range offlines {
		fmt.Fprintln(w, yellowBold("[Offline] ")+whiteBold(name))
	}
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s *statusCommand) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.StatusPath

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		statuses := rest.Statuses{}
		if err := unmarshalBody(resp.Body, &statuses); err != nil {
			return err
		}
		showStatus(statuses, w)
		return nil
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}
