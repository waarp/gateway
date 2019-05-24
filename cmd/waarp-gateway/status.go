package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"

	"code.waarp.fr/waarp/gateway-ng/pkg/admin"
	"code.waarp.fr/waarp/gateway-ng/pkg/tk/service"
	"golang.org/x/crypto/ssh/terminal"
)

var status statusCommand

// init adds the 'status' command to the program arguments parser
func init() {
	status.envPassword = os.Getenv("WG_PASSWORD")
	_, err := parser.AddCommand("status", "Show gateway status",
		"Displays the status of the queried waarp-gatewayd instance.",
		&status)
	if err != nil {
		panic(err.Error())
	}
}

// statusCommand regroups all the Options of the 'status' command
type statusCommand struct {
	Address     string `required:"true" short:"a" long:"address" description:"The address of the waarp-gatewayd server to query"`
	Username    string `required:"true" short:"u" long:"user" description:"The user's name for authentication"`
	envPassword string
}

// requestStatus makes a status request to the address stored in the statusCommand
// parameter, using the provided credentials, and returns the generated http.Response.
// If the server did not reply, or if the response code was not '200 - OK', then
// the function returns an error.
func (s *statusCommand) requestStatus(in *os.File, out *os.File) (*http.Response, error) {
	addr := s.Address + admin.RestURI + admin.StatusURI

	req, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}

	for tries := 3; tries > 0; tries-- {
		password := ""
		if s.envPassword != "" {
			password = s.envPassword
		} else if terminal.IsTerminal(int(in.Fd())) && terminal.IsTerminal(int(out.Fd())) {
			fmt.Fprintf(out, "Enter %s's password: ", s.Username)
			bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(out)
			if err != nil {
				return nil, err
			}
			password = string(bytePassword)
		} else {
			return nil, fmt.Errorf("cannot create password prompt, input is not a terminal")
		}
		req.SetBasicAuth(s.Username, password)
		client := http.Client{}
		res, err := client.Do(req)

		if err != nil {
			return nil, err
		}
		switch res.StatusCode {
		case http.StatusOK:
			return res, nil
		case http.StatusUnauthorized:
			fmt.Fprintln(os.Stderr, "Invalid authentication")
			if s.envPassword != "" {
				return nil, fmt.Errorf("invalid environment password")
			}
		default:
			return nil, fmt.Errorf(http.StatusText(res.StatusCode))
		}
	}
	return nil, fmt.Errorf("authentication failed too many times")
}

func showStatusANSI(statuses admin.Statuses, w io.Writer) {
	var errors = make([]string, 0)
	var actives = make([]string, 0)
	var offlines = make([]string, 0)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "\033[30;1;4mWaarp-Gateway services :\033[0m")
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

func showStatusNoANSI(statuses admin.Statuses, w io.Writer) {
	var errors = make([]string, 0)
	var actives = make([]string, 0)
	var offlines = make([]string, 0)

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Waarp-Gateway services :")
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
		fmt.Fprintln(w, "[Error]   "+name+" : "+statuses[name].Reason)
	}
	for _, name := range actives {
		fmt.Fprintln(w, "[Active]  "+name)
	}
	for _, name := range offlines {
		fmt.Fprintln(w, "[Offline] "+name)
	}
	fmt.Fprintln(w)
}

func showStatus(statuses admin.Statuses, w io.Writer) {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		showStatusANSI(statuses, w)
	} else {
		showStatusNoANSI(statuses, w)
	}
}

// Execute executes the 'status' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (s *statusCommand) Execute(_ []string) error {

	res, err := s.requestStatus(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var statuses = make(admin.Statuses)
	if err = json.Unmarshal(body, &statuses); err != nil {
		return err
	}
	showStatus(statuses, os.Stdout)

	return nil
}
