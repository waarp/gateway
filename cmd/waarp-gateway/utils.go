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

func sendRequest(addr *url.URL, object interface{}, method string) (*http.Response, error) {
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
	req, err := http.NewRequestWithContext(ctx, method, addr.String(), body)
	if err != nil {
		return nil, err
	}

	if pwd, hasPwd := addr.User.Password(); hasPwd {
		req.SetBasicAuth(addr.User.Username(), pwd)
	} else if terminal.IsTerminal(int(in.Fd())) && terminal.IsTerminal(int(out.Fd())) {
		fmt.Fprint(out, "Enter password:")
		password, err := terminal.ReadPassword(int(in.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(addr.User.Username(), string(password))
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

func agentListURL(path string, s *listOptions, sort string, protos []string) (*url.URL, error) {
	addr, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return nil, err
	}

	addr.Path = admin.APIPath + path
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)

	for _, proto := range protos {
		query.Add("protocol", proto)
	}
	addr.RawQuery = query.Encode()

	return addr, nil
}

func accountListURL(path string, s *listOptions, sort string) (*url.URL, error) {
	addr, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return nil, err
	}
	addr.Path = admin.APIPath + path
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)
	addr.RawQuery = query.Encode()

	return addr, nil
}

func listURL(path string, s *listOptions, sort string) (*url.URL, error) {
	addr, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return nil, err
	}
	addr.Path = path
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)

	addr.RawQuery = query.Encode()

	return addr, nil
}

func add(path string, object interface{}) error {
	addr, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	addr.Path = path

	resp, err := sendRequest(addr, object, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

func list(addr *url.URL, target interface{}) error {
	resp, err := sendRequest(addr, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return unmarshalBody(resp.Body, target)
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

func get(path string, target interface{}) error {
	addr, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %s", err)
	}
	addr.Path = path

	resp, err := sendRequest(addr, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return unmarshalBody(resp.Body, target)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

func update(path string, object interface{}) error {
	addr, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	addr.Path = path

	resp, err := sendRequest(addr, object, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %v - %s", resp.StatusCode,
			getResponseMessage(resp).Error())
	}
}

func remove(path string) error {
	addr, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return fmt.Errorf("failed to parse server URL: %s", err)
	}
	addr.Path = path

	resp, err := sendRequest(addr, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}
