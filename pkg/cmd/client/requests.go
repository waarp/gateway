package wg

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	httpTimeout = 5 * time.Second
	jsonIndent  = "    "
)

func getHTTPClient(insecure bool) *http.Client {
	//nolint:forcetypeassert //type assertion will always succeed
	customTransport := &http.Transport{}

	//nolint:gosec //needed to pass the option given by the user
	customTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: insecure,
		MinVersion:         tls.VersionTLS12,
	}

	return &http.Client{Transport: customTransport}
}

func sendRequest(ctx context.Context, object any, method string) (*http.Response, error) {
	return SendRequest(ctx, object, method, addr, insecure)
}

func SendRequest(ctx context.Context, object any, method string,
	addr *url.URL, insecure bool,
) (*http.Response, error) {
	var body io.Reader

	if object != nil {
		content, err := json.MarshalIndent(object, "", jsonIndent)
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

	resp, err := getHTTPClient(insecure).Do(req)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while sending the HTTP request: %w", err)
	}

	return resp, nil
}

func add(w io.Writer, object any) (*url.URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, object, http.MethodPost)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusCreated:
		loc, err := resp.Location()
		if err != nil && !errors.Is(err, http.ErrNoLocation) {
			return nil, fmt.Errorf("cannot get the resource location: %w", err)
		}

		return loc, displayResponseMessage(w, resp)
	case http.StatusBadRequest:
		return nil, getResponseErrorMessage(resp)
	case http.StatusNotFound:
		return nil, getResponseErrorMessage(resp)
	default:
		return nil, fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

func list(target any) error {
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
		return getResponseErrorMessage(resp)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

func get(target any) error {
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
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

func update(w io.Writer, object any) error {
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
		return displayResponseMessage(w, resp)
	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected response (%v): %w", resp.StatusCode,
			getResponseErrorMessage(resp))
	}
}

func remove(w io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusNoContent:
		return displayResponseMessage(w, resp)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

func exec(w io.Writer, path string) error {
	addr.Path = path

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusAccepted:
		return displayResponseMessage(w, resp)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}
