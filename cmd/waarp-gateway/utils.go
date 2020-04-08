package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"golang.org/x/crypto/ssh/terminal"
)

type listOptions struct {
	Limit  int `short:"l" long:"limit" description:"Max number of returned entries" default:"20"`
	Offset int `short:"o" long:"offset" description:"Index of the first returned entry" default:"0"`
}

// Deprecated
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

func sendRequest(conn *url.URL, object interface{}, method string) (*http.Response, error) {
	var body io.Reader
	if object != nil {
		content, err := json.Marshal(object)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(content)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, conn.String(), body)
	if err != nil {
		return nil, err
	}

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

func unmarshalBody(body io.Reader, object interface{}) error {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %s", err.Error())
	}

	if err := json.Unmarshal(b, object); err != nil {
		return fmt.Errorf("invalid JSON response object: %s", err.Error())
	}
	return nil
}

func getResponseMessage(resp *http.Response) error {
	body, _ := ioutil.ReadAll(resp.Body)
	return fmt.Errorf(strings.TrimSpace(string(body)))
}

// Deprecated
func handleErrors(res *http.Response, url *url.URL) error {
	switch res.StatusCode {
	case http.StatusNotFound:
		url.User = nil
		return fmt.Errorf("404 - The resource %s does not exist", url.String())
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

// Deprecated
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

// Deprecated
func addCommand(bean interface{}, conn *url.URL) (string, error) {
	return sendBean(bean, conn, http.MethodPost)
}

// Deprecated
func updateCommand(bean interface{}, conn *url.URL) (string, error) {
	return sendBean(bean, conn, http.MethodPatch)
}

// Deprecated
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

	if res.StatusCode != http.StatusCreated {
		return "", handleErrors(res, conn)
	}

	loc, err := res.Location()
	if err != nil {
		return "", err
	}

	if res.ContentLength > 0 {
		msg, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", err
		}
		fmt.Fprint(out, string(msg))
	}

	loc.User = nil
	return loc.String(), nil
}

// Deprecated
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

func agentListURL(path string, s *listOptions, sort string, protos []string) (*url.URL, error) {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return nil, err
	}

	conn.Path = admin.APIPath + path
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)

	for _, proto := range protos {
		query.Add("protocol", proto)
	}
	conn.RawQuery = query.Encode()

	return conn, nil
}

func accountListURL(path string, s *listOptions, sort string) (*url.URL, error) {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return nil, err
	}
	conn.Path = admin.APIPath + path
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)
	conn.RawQuery = query.Encode()

	return conn, nil
}

func listURL(path string, s *listOptions, sort string) (*url.URL, error) {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return nil, err
	}
	conn.Path = admin.APIPath + path
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)

	conn.RawQuery = query.Encode()

	return conn, nil
}
