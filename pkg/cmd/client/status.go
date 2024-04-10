package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/jedib0t/go-pretty/v6/text"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:varnamelen //formatter name is kept short for readability
func showStatus(f *Formatter, title string, services []*api.Service) {
	if len(services) == 0 {
		f.Title("%s: %s", title, "<none>")

		return
	}

	var (
		running = text.Colors{text.FgHiGreen}.Sprint("[ACTIVE] ")
		inError = text.Colors{text.FgHiRed}.Sprint("[ERROR]  ")
		offline = text.Colors{text.FgHiBlack}.Sprint("[OFFLINE]")

		nameColor = text.Colors{text.Bold}
	)

	f.Title(title)
	f.Indent()

	defer f.UnIndent()

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
		f.Println("%s %s (%s)", inError, nameColor.Sprint(service.Name), service.Reason)
	}

	for _, service := range actives {
		f.Println("%s %s", running, nameColor.Sprint(service.Name))
	}

	for _, service := range offlines {
		f.Println("%s %s", offline, nameColor.Sprint(service.Name))
	}
}

// Status regroups all the options of the 'status' command.
type Status struct{}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s Status) Execute([]string) error { return s.execute(stdOutput) }

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

		f := NewFormatter(w)

		defer f.Render()

		f.Value("Server version", resp.Header.Get("Server"))
		f.Value("Local date", resp.Header.Get(api.DateHeader))

		showStatus(f, "Core services", body["coreServices"])
		showStatus(f, "Servers", body["servers"])
		showStatus(f, "Clients", body["clients"])

		return nil

	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseErrorMessage(resp))
	}
}
