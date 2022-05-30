package wg

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const httpTimeout = 5 * time.Second

func getHTTPClient() *http.Client {
	//nolint:forcetypeassert //type assertion will always succeed
	customTransport := http.DefaultTransport.(*http.Transport).Clone()

	//nolint:gosec // needed to pass the option given by the user
	customTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: commandLine.addrOpt.Insecure,
		MinVersion:         tls.VersionTLS12,
	}

	return &http.Client{Transport: customTransport}
}

func sendRequest(ctx context.Context, object interface{}, method string) (*http.Response, error) {
	var body io.Reader

	if object != nil {
		content, err := json.Marshal(object)
		if err != nil {
			return nil, fmt.Errorf("cannot parse the request body: %w", err)
		}

		body = bytes.NewReader(content)
	}

	user := addr.User.Username()
	passwd, _ := addr.User.Password()

	req, err := http.NewRequestWithContext(ctx, method, addr.String(), body)
	if err != nil {
		return nil, fmt.Errorf("cannot prepare request: %w", err)
	}

	req.SetBasicAuth(user, passwd)

	resp, err := getHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while sending the HTTP request: %w", err)
	}

	return resp, nil
}

func add(object interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, object, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseMessage(resp))
	}
}

func list(target interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusOK:
		return unmarshalBody(resp.Body, target)
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseMessage(resp))
	}
}

func get(target interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusOK:
		return unmarshalBody(resp.Body, target)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %w", getResponseMessage(resp))
	}
}

func update(object interface{}) error {
	if isNotUpdate(object) {
		return errNothingToDo
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, object, http.MethodPatch)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%v): %w", resp.StatusCode,
			getResponseMessage(resp))
	}
}

func remove() error {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %w", getResponseMessage(resp))
	}
}
