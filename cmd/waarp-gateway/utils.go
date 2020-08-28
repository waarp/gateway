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

func isTerminal() bool {
	return terminal.IsTerminal(int(in.Fd())) && terminal.IsTerminal(int(out.Fd()))
}

func promptUser() (string, error) {
	if !isTerminal() {
		return "", fmt.Errorf("the username is missing from the URL")
	}

	var user string
	fmt.Fprintf(out, "Username:")
	if _, err := fmt.Fscanln(in, &user); err != nil {
		return "", err
	}
	return user, nil
}

func promptPassword() (string, error) {
	if !isTerminal() {
		return "", fmt.Errorf("the user password is missing from the URL")
	}

	fmt.Fprint(out, "Password:")
	pwd, err := terminal.ReadPassword(int(in.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Fprintln(out)

	return string(pwd), nil
}

func sendRequest(object interface{}, method string) (*http.Response, error) {
	var body io.Reader
	if object != nil {
		content, err := json.Marshal(object)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(content)
	}

	user := addr.User.Username()
	passwd, _ := addr.User.Password()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, addr.String(), body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, passwd)

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

func agentListURL(path string, s *listOptions, sort string, protos []string) {
	addr.Path = admin.APIPath + path
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)

	for _, proto := range protos {
		query.Add("protocol", proto)
	}
	addr.RawQuery = query.Encode()
}

func listURL(s *listOptions, sort string) {
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)
	addr.RawQuery = query.Encode()
}

func add(object interface{}) error {
	resp, err := sendRequest(object, http.MethodPost)
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

func list(target interface{}) error {
	resp, err := sendRequest(nil, http.MethodGet)
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

func get(target interface{}) error {
	resp, err := sendRequest(nil, http.MethodGet)
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

func update(object interface{}) error {
	resp, err := sendRequest(object, http.MethodPut)
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
	addr.Path = path

	resp, err := sendRequest(nil, http.MethodDelete)
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

type addrOpt struct {
	Address gwAddr `short:"a" long:"address" description:"The address of the gateway" env:"WAARP_GATEWAY_ADDRESS"`
}

type gwAddr struct{}

func (*gwAddr) UnmarshalFlag(value string) error {
	if value == "" {
		return fmt.Errorf("the address flags '-a' is missing")
	}

	if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
		value = "http://" + value
	}

	parsedURL, err := url.ParseRequestURI(value)
	if err != nil {
		return err.(*url.Error).Err
	}

	if _, hasPwd := parsedURL.User.Password(); !hasPwd {
		user := parsedURL.User.Username()
		if user == "" {
			var err error
			if user, err = promptUser(); err != nil {
				return err
			}
		}

		pwd, err := promptPassword()
		if err != nil {
			return err
		}
		parsedURL.User = url.UserPassword(user, pwd)
	}

	addr = parsedURL
	return nil
}
