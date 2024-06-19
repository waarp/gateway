package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gookit/color"
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
		if msg, err := io.ReadAll(resp.Body); err != nil {
			fmt.Fprintln(w, color.Red.Sprintf(
				"<WARNING: error while reading the response body: %v>", err))
		} else if len(msg) != 0 {
			fmt.Fprintln(w, string(msg))
		}

		fmt.Fprintf(w, "The %s %q is now allowed to use the %s rule %q for transfers.\n",
			targetType, target, direction, rule)

		return nil

	case http.StatusNotFound:
		return getResponseErrorMessage(resp)

	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

func revoke(w io.Writer, targetType, target, rule, direction string) error {
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
		fmt.Fprintf(w, "The %s %q is no longer allowed to use the %s rule %q for transfers.\n",
			targetType, target, direction, rule)

		if msg, err := io.ReadAll(resp.Body); err != nil {
			fmt.Fprintln(w, color.Red.Sprintf(
				"<WARNING: error while reading the response body: %v>", err))
		} else if len(msg) != 0 {
			fmt.Fprintln(w, string(msg))
		}

		return nil

	case http.StatusNotFound:
		return getResponseErrorMessage(resp)

	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}
