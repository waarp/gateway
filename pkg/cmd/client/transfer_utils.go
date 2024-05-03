package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gookit/color"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func coloredStatus(status types.TransferStatus) string {
	switch status {
	case types.StatusPlanned:
		return color.HiWhite.Render(status)
	case types.StatusRunning:
		return color.Blue.Render(status)
	case types.StatusPaused:
		return color.HiYellow.Render(status)
	case types.StatusInterrupted:
		return color.HiYellow.Render(status)
	case types.StatusCancelled:
		return color.Gray.Render(status)
	case types.StatusError:
		return color.HiRed.Render(status)
	case types.StatusDone:
		return color.Green.Render(status)
	default:
		return color.OpItalic.Sprintf("<unrecognized=%q>", status)
	}
}

type pair struct {
	key string
	val any
}

func displayTransferInfo(w io.Writer, info map[string]any) {
	if len(info) == 0 {
		return
	}

	style22.printf(w, "Transfer values:")
	displayMap(w, style333, info)
}

func putTransferRequest(w io.Writer, id uint64, endpoint, action string) error {
	addr.Path = fmt.Sprintf("/api/transfers/%d/%s", id, endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck,gosec // error is irrelevant

	switch resp.StatusCode {
	case http.StatusAccepted:
		fmt.Fprintf(w, "The transfer \"%d\" was successfully %s.\n", id, action)

		return nil

	case http.StatusNotFound:
		return getResponseErrorMessage(resp)

	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)

	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseErrorMessage(resp))
	}
}
