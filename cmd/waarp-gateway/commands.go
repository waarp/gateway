package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

type listOptions struct {
	Limit     int  `short:"l" long:"limit" description:"Max number of returned entries" default:"20"`
	Offset    int  `short:"o" long:"offset" description:"Index of the first returned entry" default:"0"`
	DescOrder bool `short:"d" long:"desc" description:"If set, entries will be sorted in descending order instead of ascending"`
}

func executeRequest(req *http.Request, conn *url.URL) (*http.Response, error) {

	if pwd, hasPwd := conn.User.Password(); hasPwd {
		req.SetBasicAuth(conn.User.Username(), pwd)
	} else if terminal.IsTerminal(int(in.Fd())) && terminal.IsTerminal(int(out.Fd())) {
		fmt.Fprint(out, "Enter password:")
		password, err := terminal.ReadPassword(int(in.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(conn.User.Username(), string(password))
	} else {
		return nil, fmt.Errorf("missing user password")
	}

	return http.DefaultClient.Do(req)
}

func handleErrors(res *http.Response, url *url.URL) error {
	switch res.StatusCode {
	case http.StatusNotFound:
		url.User = nil
		return fmt.Errorf("404 - The resource '%s' does not exist", url.String())
	case http.StatusBadRequest:
		body, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("400 - Invalid request: %s", strings.TrimSpace(string(body)))
	case http.StatusUnauthorized:
		return fmt.Errorf("401 - Invalid credentials")
	case http.StatusForbidden:
		return fmt.Errorf("403 - You do not have sufficient privileges to perform this action")
	default:
		if res.Body != nil {
			body, _ := ioutil.ReadAll(res.Body)
			return fmt.Errorf("%v - Unexpected error: %s", res.StatusCode,
				strings.TrimSpace(string(body)))
		}
		return fmt.Errorf("%v - An unexpected error occurred", res.StatusCode)
	}
}

func getCommand(bean interface{}, conn *url.URL) error {
	req, err := http.NewRequest(http.MethodGet, conn.String(), nil)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, conn)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return handleErrors(res, conn)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, bean); err != nil {
		return err
	}

	return nil
}

func addCommand(bean interface{}, conn *url.URL) (string, error) {
	return sendBean(bean, conn, http.MethodPost)
}

func updateCommand(bean interface{}, conn *url.URL) (string, error) {
	return sendBean(bean, conn, http.MethodPatch)
}

func sendBean(bean interface{}, conn *url.URL, method string) (string, error) {
	content, err := json.Marshal(bean)
	if err != nil {
		return "", err
	}
	body := bytes.NewReader(content)

	req, err := http.NewRequest(method, conn.String(), body)
	if err != nil {
		return "", err
	}

	res, err := executeRequest(req, conn)
	if err != nil {
		return "", err
	}

	// TODO May be usefull to delete http.StatusAcepted when it is not longer used
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusAccepted {
		return "", handleErrors(res, conn)
	}

	loc, err := res.Location()
	if err != nil {
		return "", nil
	}
	loc.User = nil
	return loc.String(), nil
}

func deleteCommand(conn *url.URL) error {
	req, err := http.NewRequest(http.MethodDelete, conn.String(), nil)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, conn)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusNoContent {
		return handleErrors(res, conn)
	}
	return nil
}
