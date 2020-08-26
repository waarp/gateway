package main

import (
	"fmt"
	"net/http"
	"net/url"
)

func authorize(path, targetType, target, rule, direction string) error {
	if err := checkRuleDir(direction); err != nil {
		return err
	}

	conn, err := url.Parse(commandLine.Address)
	if err != nil {
		return err
	}
	conn.Path = path

	resp, err := sendRequest(conn, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		if msg := getResponseMessage(resp).Error(); msg != "" {
			fmt.Fprintln(w, msg)
		}
		fmt.Fprintln(w, "The", targetType, bold(target),
			"is now allowed to use the", direction, "rule", bold(rule),
			"for transfers.")
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

func revoke(path, targetType, target, rule, direction string) error {
	if err := checkRuleDir(direction); err != nil {
		return err
	}

	conn, err := url.Parse(commandLine.Address)
	if err != nil {
		return err
	}
	conn.Path = path

	resp, err := sendRequest(conn, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Fprintln(w, "The", targetType, bold(target),
			"is no longer allowed to use the", direction, "rule", bold(rule),
			"for transfers.")
		if msg := getResponseMessage(resp).Error(); msg != "" {
			fmt.Fprintln(w, msg)
		}
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}
