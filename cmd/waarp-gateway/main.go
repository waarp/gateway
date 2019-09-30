package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"

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

func init() {
	var partner partnerCommand
	p, err := parser.AddCommand("partner", "Manage waarp-gateway partners",
		"The command to manage the waarp-gateway partners.",
		&partner)
	if err != nil {
		panic(err.Error())
	}

	var get partnerGetCommand
	_, err = p.AddCommand("get", "Get partner",
		"Retrieve a single partner entry.",
		&get)
	if err != nil {
		panic(err.Error())
	}

	var list partnerListCommand
	_, err = p.AddCommand("list", "List partners",
		"List the partner entries.",
		&list)
	if err != nil {
		panic(err.Error())
	}

	var create partnerCreateCommand
	_, err = p.AddCommand("create", "Create partner",
		"Create a partner and add it to the database.",
		&create)
	if err != nil {
		panic(err.Error())
	}

	var update partnerUpdateCommand
	_, err = p.AddCommand("update", "Update partner",
		"Updates a partner entry in the database.",
		&update)
	if err != nil {
		panic(err.Error())
	}

	var del partnerDeleteCommand
	_, err = p.AddCommand("delete", "Delete partner",
		"Removes a partner entry from the database.",
		&del)
	if err != nil {
		panic(err.Error())
	}

	var account accountCommand
	a, err := parser.AddCommand("account", "Manage waarp-gateway accounts",
		"The command to manage the waarp-gateway accounts.",
		&account)
	if err != nil {
		panic(err.Error())
	}

	var listAccount accountListCommand
	_, err = a.AddCommand("list", "List accounts",
		"List the account entries.",
		&listAccount)
	if err != nil {
		panic(err.Error())
	}

	var createAccount accountCreateCommand
	_, err = a.AddCommand("create", "Create account",
		"Create an account and add it to the database.",
		&createAccount)
	if err != nil {
		panic(err.Error())
	}

	var updateAccount accountUpdateCommand
	_, err = a.AddCommand("update", "Update account",
		"Updates an account entry in the database.",
		&updateAccount)
	if err != nil {
		panic(err.Error())
	}

	var deleteAccount accountDeleteCommand
	_, err = a.AddCommand("delete", "Delete account",
		"Removes an account entry from the database.",
		&deleteAccount)
	if err != nil {
		panic(err.Error())
	}

	var getAccount accountGetCommand
	_, err = a.AddCommand("get", "Get account",
		"Retrieves an account entry from the database.",
		&getAccount)
	if err != nil {
		panic(err.Error())
	}

	var certificate certificateCommand
	c, err := parser.AddCommand("certificate", "Manage waarp-gateway certificates",
		"The command to manage the waarp-gateway certificates.",
		&certificate)
	if err != nil {
		panic(err.Error())
	}

	var updateCert certificateUpdateCommand
	_, err = c.AddCommand("update", "Update certificate",
		"Updates a certificate entry in the database.",
		&updateCert)
	if err != nil {
		panic(err.Error())
	}

	var deleteCert certificateDeleteCommand
	_, err = c.AddCommand("delete", "Delete certificate",
		"Removes a certificate entry from the database.",
		&deleteCert)
	if err != nil {
		panic(err.Error())
	}

	var getCert certificateGetCommand
	_, err = c.AddCommand("get", "Get certificate",
		"Retrieve a certificate entry from the database.",
		&getCert)
	if err != nil {
		panic(err.Error())
	}

	var listCert certificateListCommand
	_, err = c.AddCommand("list", "List certificates",
		"List the certificate entries.",
		&listCert)
	if err != nil {
		panic(err.Error())
	}

	var createCert certificateCreateCommand
	_, err = c.AddCommand("create", "Create certificate",
		"Create a certificate and add it to the database.",
		&createCert)
	if err != nil {
		panic(err.Error())
	}

	var inter interfaceCommand
	i, err := parser.AddCommand("interface", "Manage waarp-gateway interfaces",
		"The command to manage the waarp-gateway interfaces.",
		&inter)
	if err != nil {
		panic(err.Error())
	}

	var updateInter interfaceUpdateCommand
	_, err = i.AddCommand("update", "Update interface",
		"Updates an interface entry in the database.",
		&updateInter)
	if err != nil {
		panic(err.Error())
	}

	var deleteInter interfaceDeleteCommand
	_, err = i.AddCommand("delete", "Delete interface",
		"Removes an interface entry from the database.",
		&deleteInter)
	if err != nil {
		panic(err.Error())
	}

	var getInter interfaceGetCommand
	_, err = i.AddCommand("get", "Get interface",
		"Retrieve an interface entry from the database.",
		&getInter)
	if err != nil {
		panic(err.Error())
	}

	var listInter interfaceListCommand
	_, err = i.AddCommand("list", "List interfaces",
		"List the interface entries.",
		&listInter)
	if err != nil {
		panic(err.Error())
	}

	var createInter interfaceCreateCommand
	_, err = i.AddCommand("create", "Create interface",
		"Create an interface and add it to the database.",
		&createInter)
	if err != nil {
		panic(err.Error())
	}
}

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

func sendBean(bean interface{}, in, out *os.File, addr, method string) (string, error) {

	content, err := json.Marshal(bean)
	if err != nil {
		return "", err
	}
	body := bytes.NewReader(content)

	req, err := http.NewRequest(method, addr, body)
	if err != nil {
		return "", err
	}

	res, err := executeRequest(req, auth.Username, in, out)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	return res.Header.Get("Location"), nil
}

func getColorable(f *os.File) io.Writer {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		return colorable.NewColorable(f)
	}
	return colorable.NewNonColorable(f)
}

func getCommand(in, out *os.File, subpath string, bean interface{}) error {
	addr := auth.Address + admin.RestURI + subpath

	req, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, auth.Username, in, out)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, bean); err != nil {
		return err
	}

	return nil
}
