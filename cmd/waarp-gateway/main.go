package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/mattn/go-colorable"
	"golang.org/x/crypto/ssh/terminal"
)

const orderDesc = "&order=desc"

var (
	parser      = flags.NewNamedParser("waarp-gateway", flags.Default)
	envPassword = os.Getenv("WG_PASSWORD")

	auth ConnectionOptions
)

// ConnectionOptions regroups the flags common to all commands
type ConnectionOptions struct {
	Address  string `required:"true" short:"r" long:"remote" description:"The address of the remote waarp-gatewayd server to query, must be prefixed with either 'http://' or 'https:// depending on the gateway SSL configuration'"`
	Username string `required:"true" short:"u" long:"user" description:"The user's name for authentication"`
}

func init() {
	_, err := parser.AddGroup("Connection Options",
		"The information necessary to connect to the remote service.",
		&auth)
	if err != nil {
		panic(err.Error())
	}
}

func main() {

	_, err := parser.Parse()

	if err != nil && !flags.WroteHelp(err) {
		os.Exit(1)
	}
}

func executeRequest(req *http.Request, user string, in *os.File, out *os.File) (*http.Response, error) {

	for tries := 3; tries > 0; tries-- {
		password := ""
		if envPassword != "" {
			password = envPassword
		} else if terminal.IsTerminal(int(in.Fd())) && terminal.IsTerminal(int(out.Fd())) {
			fmt.Fprintf(out, "Enter %s's password: ", user)
			bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(out)
			if err != nil {
				return nil, err
			}
			password = string(bytePassword)
		} else {
			return nil, fmt.Errorf("cannot create password prompt, input is not a terminal")
		}
		req.SetBasicAuth(user, password)
		client := http.Client{}
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		switch res.StatusCode {
		case http.StatusOK:
			return res, nil
		case http.StatusCreated:
			return res, nil
		case http.StatusNoContent:
			return res, nil
		case http.StatusUnauthorized:
			fmt.Fprintln(os.Stderr, "Invalid authentication")
			if envPassword != "" {
				_ = res.Body.Close()
				// FIXME: maybe not the reason
				// FIXME: is it supposed to be a continue ?
				return nil, fmt.Errorf("invalid environment password")
			}
		default:
			body, err := ioutil.ReadAll(res.Body)
			msg := strings.TrimSpace(string(body))
			if err != nil {
				return nil, err
			}
			_ = res.Body.Close()
			return nil, fmt.Errorf(msg)
		}
	}
	return nil, fmt.Errorf("authentication failed too many times")
}

func getColorable(f *os.File) io.Writer {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		return colorable.NewColorable(f)
	}
	return colorable.NewNonColorable(f)
}
