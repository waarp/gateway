package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"golang.org/x/net/publicsuffix"
)

type httpRestError struct {
	code int
}

func (h httpRestError) Error() string {
	return fmt.Sprintf("the request did not end with success, return code was %d", h.code)
}

type httpClient struct {
	c *http.Client
}

func newClient() (*httpClient, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("cannot initialize the HTTP client: %w", err)
	}

	c := &httpClient{
		c: &http.Client{
			Jar: jar,
		},
	}

	return c, nil
}

func (h *httpClient) getJSON(address string, respObj any) error {
	logger := getLogger()

	req, err := http.NewRequestWithContext(context.Background(),
		http.MethodGet, address, http.NoBody)
	if err != nil {
		return fmt.Errorf("cannot prepare request to create a partner: %w", err)
	}

	resp, err := h.c.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send HTTP Request to Manager: %w", err)
	}

	defer func() {
		if err2 := resp.Body.Close(); err2 != nil {
			logger.Warningf("This error occurred while reading the response: %v", err2)
		}
	}()

	bodyContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("cannot read response from Manager: %w", err)
	}

	logger.Debugf("Running request %s %s -> [%d] %s",
		http.MethodGet, address, resp.StatusCode, string(bodyContent))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return httpRestError{resp.StatusCode}
	}

	if respObj != nil {
		if err = json.Unmarshal(bodyContent, respObj); err != nil {
			return fmt.Errorf("cannot parse response: %w", err)
		}
	}

	return nil
}

func (h *httpClient) postJSON(address string, data, respObj any) error {
	return h.xxxJSON(http.MethodPost, address, data, respObj)
}

func (h *httpClient) putJSON(address string, data, respObj any) error {
	return h.xxxJSON(http.MethodPut, address, data, respObj)
}

func (h *httpClient) xxxJSON(method, address string, data, respObj any) error {
	logger := getLogger()

	msgBytes, jErr := json.Marshal(data)
	if jErr != nil {
		return fmt.Errorf("cannot prepare data for the request to create a partner: %w", jErr)
	}

	req, reqErr := http.NewRequestWithContext(context.Background(),
		method, address, bytes.NewReader(msgBytes))
	if reqErr != nil {
		return fmt.Errorf("cannot prepare request to create a partner: %w", reqErr)
	}

	req.Header.Add("Content-Type", "application/json")

	resp, doErr := h.c.Do(req)
	if doErr != nil {
		return fmt.Errorf("cannot send HTTP Request to Manager: %w", doErr)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warningf("This error occurred while reading the response: %v", err)
		}
	}()

	bodyContent, rErr := io.ReadAll(resp.Body)
	if rErr != nil {
		return fmt.Errorf("cannot read response from Manager: %w", rErr)
	}

	logger.Debugf("Running request %s %s -> [%d] %s",
		method, address, resp.StatusCode, string(bodyContent))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return httpRestError{resp.StatusCode}
	}

	if respObj != nil {
		if err := json.Unmarshal(bodyContent, respObj); err != nil {
			return fmt.Errorf("cannot parse response: %w", err)
		}
	}

	return nil
}

func (h *httpClient) postForm(address string, data url.Values) error {
	logger := getLogger()

	msgBytes := []byte(data.Encode())

	req, err := http.NewRequestWithContext(context.Background(),
		http.MethodPost, address, bytes.NewReader(msgBytes))
	if err != nil {
		return fmt.Errorf("cannot prepare request to create a partner: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.c.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send HTTP Request to Manager: %w", err)
	}

	defer func() {
		if err2 := resp.Body.Close(); err2 != nil {
			logger.Warningf("This error occurred while reading the response: %v", err2)
		}
	}()

	bodyContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("cannot read response from Manager: %w", err)
	}

	logger.Debugf("Call to %s -> [%d] %s",
		address, resp.StatusCode, string(bodyContent))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return httpRestError{resp.StatusCode}
	}

	return nil
}
