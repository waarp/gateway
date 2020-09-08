package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

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
	if isNotUpdate(object) {
		return fmt.Errorf("nothing to do")
	}

	resp, err := sendRequest(object, http.MethodPatch)
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
