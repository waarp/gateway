package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/jedib0t/go-pretty/v6/text"
)

func authorize(w io.Writer, targetType, target, rule, direction string) error {
	if err := checkRuleDir(direction); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, reqErr := sendRequest(ctx, nil, http.MethodPut)
	if reqErr != nil {
		return reqErr
	}

	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusOK:
		wCol := makeColorable(w)

		if msg, err := io.ReadAll(resp.Body); err != nil {
			fmt.Fprintln(wCol, text.FgRed.Sprintf(
				"<WARNING: error while reading the response body: %v>", err))
		} else if len(msg) != 0 {
			fmt.Fprintln(wCol, string(msg))
		}

		fmt.Fprintln(wCol, "The", targetType, bold(target),
			"is now allowed to use the", direction, "rule", bold(rule),
			"for transfers.")

		return nil

	case http.StatusNotFound:
		return getResponseErrorMessage(resp)

	default:
		return fmt.Errorf("unexpected error: %w", getResponseErrorMessage(resp))
	}
}

func revoke(w io.Writer, targetType, target, rule, direction string) error {
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

	switch resp.StatusCode {
	case http.StatusOK:
		wCol := makeColorable(w)

		fmt.Fprintln(wCol, "The", targetType, bold(target),
			"is no longer allowed to use the", direction, "rule", bold(rule),
			"for transfers.")

		if msg := getResponseErrorMessage(resp).Error(); msg != "" {
			fmt.Fprintln(wCol, msg)
		}

		return nil

	case http.StatusNotFound:
		return getResponseErrorMessage(resp)

	default:
		return fmt.Errorf("unexpected error: %w", getResponseErrorMessage(resp))
	}
}
