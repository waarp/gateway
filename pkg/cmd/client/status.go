package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/gookit/color"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func showStatuses(headers http.Header) func(io.Writer, map[string][]*api.Service) error {
	return func(w io.Writer, servicesMap map[string][]*api.Service) error {
		Style1.PrintL(w, "Server version", headers.Get("Server"))
		Style1.PrintL(w, "Local date", headers.Get(api.DateHeader))

		showStatus(w, "Core services", servicesMap["coreServices"])
		showStatus(w, "Servers", servicesMap["servers"])
		showStatus(w, "Clients", servicesMap["clients"])

		return nil
	}
}

//nolint:varnamelen //formatter name is kept short for readability
func showStatus(w io.Writer, title string, services []*api.Service) {
	if len(services) == 0 {
		Style1.Printf(w, "%s: %s\n", title, none)

		return
	}

	var (
		running = "[" + color.HiGreen.Sprint("ACTIVE ") + "]"
		inError = "[" + color.HiRed.Sprint("ERROR  ") + "]"
		offline = "[" + color.Gray.Sprint("OFFLINE") + "]"

		nameColor = color.Bold
	)

	Style1.PrintV(w, title+":")

	var errors, actives, offlines []*api.Service

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	for _, service := range services {
		switch service.State {
		case utils.StateRunning.String():
			actives = append(actives, service)
		case utils.StateError.String():
			errors = append(errors, service)
		default:
			offlines = append(offlines, service)
		}
	}

	for _, service := range errors {
		color.Fprintf(w, "%s%s %s (%s)\n", Style22.bulletPrefix, inError,
			nameColor.Render(service.Name), service.Reason)
	}

	for _, service := range actives {
		color.Fprintf(w, "%s%s %s\n", Style22.bulletPrefix, running,
			nameColor.Render(service.Name))
	}

	for _, service := range offlines {
		color.Fprintf(w, "%s%s %s\n", Style22.bulletPrefix, offline,
			nameColor.Render(service.Name))
	}
}

// Status regroups all the options of the 'status' command.
type Status struct {
	OutputFormat
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s Status) Execute([]string) error { return execute(s) }

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
		body := map[string][]*api.Service{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}

		return outputObject(w, body, &s.OutputFormat, showStatuses(resp.Header))

	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseErrorMessage(resp))
	}
}
