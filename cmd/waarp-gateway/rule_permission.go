package main

import (
	"context"
	"fmt"
	"net/http"
)

func authorize(targetType, target, rule, direction string) error {
	if err := checkRuleDir(direction); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}

	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

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
		return fmt.Errorf("unexpected error: %w", getResponseMessage(resp))
	}
}

func revoke(targetType, target, rule, direction string) error {
	if err := checkRuleDir(direction); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

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
		return fmt.Errorf("unexpected error: %w", getResponseMessage(resp))
	}
}
