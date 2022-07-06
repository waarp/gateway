package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// Status regroups all the options of the 'status' command.
type Status struct{}

// showStatus writes the status of the gateway services in the given
// writer with colors, using ANSI coloration codes.
func showStatus(statuses api.Statuses, w io.Writer) {
	errors := make([]string, 0)
	actives := make([]string, 0)
	offlines := make([]string, 0)

	fmt.Fprintln(w, bold("Waarp-Gateway services:"))

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
		fmt.Fprintln(w, red("[Error]  "), bold(name), "("+
			statuses[name].Reason+")")
	}

	for _, name := range actives {
		fmt.Fprintln(w, green("[Active] "), bold(name))
	}

	for _, name := range offlines {
		fmt.Fprintln(w, yellow("[Offline]"), bold(name))
	}
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s Status) Execute([]string) error {
	addr.Path = "/api/status"

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // FIXME nothing to handle the error

	w := getColorable()

	switch resp.StatusCode {
	case http.StatusOK:
		statuses := api.Statuses{}
		if err := unmarshalBody(resp.Body, &statuses); err != nil {
			return err
		}

		showStatus(statuses, w)

		return nil

	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseMessage(resp))
	}
}
