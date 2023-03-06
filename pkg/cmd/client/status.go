package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
)

type services []api.Service

func (s services) Len() int           { return len(s) }
func (s services) Less(i, j int) bool { return s[i].Name < s[j].Name }
func (s services) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// showStatus writes the status of the gateway services in the given
// writer with colors, using ANSI coloration codes.
func showStatus(w io.Writer, title string, services services) {
	if len(services) == 0 {
		fmt.Fprintln(w, bold("%s:", title), grey("<none>"))

		return
	}

	fmt.Fprintln(w, bold("%s:", title))

	errors := &strings.Builder{}
	actives := &strings.Builder{}
	offlines := &strings.Builder{}

	sort.Sort(services)

	for _, service := range services {
		switch service.State {
		case state.Running.Name():
			fmt.Fprintln(actives, green("[Active] "), bold(service.Name))
		case state.Error.Name():
			fmt.Fprintln(errors, red("[Error]  "), bold(service.Name), "("+
				service.Reason+")")
		default:
			fmt.Fprintln(offlines, yellow("[Offline]"), bold(service.Name))
		}
	}

	fmt.Fprint(w, errors.String())
	fmt.Fprint(w, actives.String())
	fmt.Fprint(w, offlines.String())
}

// Status regroups all the options of the 'status' command.
type Status struct{}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s Status) Execute([]string) error { return s.execute(os.Stdout) }

func (s Status) execute(w io.Writer) error {
	addr.Path = "/api/about"

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, reqErr := sendRequest(ctx, nil, http.MethodGet)
	if reqErr != nil {
		return reqErr
	}
	defer resp.Body.Close() //nolint:errcheck // FIXME nothing to handle the error

	switch resp.StatusCode {
	case http.StatusOK:
		body := map[string][]api.Service{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}

		w = makeColorable(w)

		fmt.Fprintln(w, yellow("Server info:"), resp.Header.Get("Server"))
		fmt.Fprintln(w, yellow("Local date:"), resp.Header.Get(api.DateHeader))
		fmt.Fprintln(w)

		showStatus(w, "Core services", body["coreServices"])
		fmt.Fprintln(w)
		showStatus(w, "Servers", body["servers"])
		fmt.Fprintln(w)
		showStatus(w, "Clients", body["clients"])
		fmt.Fprintln(w)

		return nil

	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseErrorMessage(resp))
	}
}
