package main

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
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

const (
	httpTimeout = 5 * time.Second
	jsonIndent  = "    "
)

var errNotImplemented = errors.New("HTTP Not Implemented")

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

func sendRequest(ctx context.Context, object any, method string,
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
		if errors.Is(err, context.DeadlineExceeded) {
			//nolint:err113 //dynamic error is better here for readability
			return nil, fmt.Errorf("%s %s: request timed out", req.Method, req.URL.Redacted())
		}

		return nil, fmt.Errorf("an error occurred while sending the HTTP request: %w", err)
	}

	return resp, nil
}

func get(target any, path string, insecure bool) error {
	requestURL, err := url.Parse(path)
	if err != nil {
		return fmt.Errorf("fail to parse URL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, target, http.MethodGet, requestURL, insecure)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusOK:
		return unmarshalBody(resp.Body, target)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	case http.StatusNotImplemented:
		return fmt.Errorf("%w: %w", getResponseErrorMessage(resp), errNotImplemented)
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

func add(object any, path string, insecure bool) error {
	requestURL, urlErr := url.Parse(path)
	if urlErr != nil {
		return fmt.Errorf("fail to parse URL: %w", urlErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, reqErr := sendRequest(ctx, object, http.MethodPost, requestURL, insecure)
	if reqErr != nil {
		return reqErr
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusCreated:
		_, err := resp.Location()
		if err != nil && !errors.Is(err, http.ErrNoLocation) {
			return fmt.Errorf("cannot get the resource location: %w", err)
		}

		return nil
	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

func getCreds(credNames []string, addr string, insecure bool) ([]api.OutCred, error) {
	res := []api.OutCred{}

	for _, credName := range credNames {
		restPath, urlErr := url.JoinPath(addr, credName)
		if urlErr != nil {
			return nil, fmt.Errorf("failed to build URL: %w", urlErr)
		}

		apiCred := api.OutCred{}
		if err := get(&apiCred, restPath, insecure); err != nil {
			return res, err
		}
		res = append(res, apiCred)
	}

	return res, nil
}

func getRealAddress(targetAddr, addr string, insecure bool) (string, error) {
	var apiOverride map[string]string

	if err := get(&apiOverride, addr, insecure); err != nil {
		if errors.Is(err, errNotImplemented) {
			// If no override return targetAddr
			return targetAddr, nil
		}

		return "", err
	}

	overrideAddr, ok := apiOverride[targetAddr]
	if !ok || overrideAddr == "" {
		return targetAddr, nil
	}

	return overrideAddr, nil
}

func unmarshalBody(body io.Reader, object any) error {
	decoder := json.NewDecoder(body)
	decoder.UseNumber()

	if err := decoder.Decode(object); err != nil {
		return fmt.Errorf("invalid JSON response object: %w", err)
	}

	return nil
}

func getResponseErrorMessage(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read the response body: %w", err)
	}

	cleanMsg := strings.TrimSpace(string(body))

	return fmt.Errorf("%s: %s", resp.Status, cleanMsg) //nolint:err113 // too specific
}
