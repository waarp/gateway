package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/jedib0t/go-pretty/v6/text"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func coloredStatus(status types.TransferStatus) string {
	switch status {
	case types.StatusPlanned:
		return text.FgWhite.Sprint(status)
	case types.StatusRunning:
		return text.FgCyan.Sprint(status)
	case types.StatusPaused:
		return text.FgYellow.Sprint(status)
	case types.StatusInterrupted:
		return text.FgHiRed.Sprint(status)
	case types.StatusCancelled:
		return text.FgHiBlack.Sprint(status)
	case types.StatusError:
		return text.FgRed.Sprint(status)
	case types.StatusDone:
		return text.FgHiGreen.Sprint(status)
	default:
		return string(status)
	}
}

func displayTransferFile(f *Formatter, trans *api.OutTransfer) {
	switch {
	case trans.IsServer && trans.IsSend: // <- Server
		f.Value("File pulled", trans.SrcFilename)
	case trans.IsServer && !trans.IsSend: // -> Server
		f.Value("File pushed", trans.DestFilename)
	case !trans.IsServer && trans.IsSend: // Client ->
		f.Value("File to send", trans.SrcFilename)
		f.ValueCond("File deposited as", trans.DestFilename)
	case !trans.IsServer && !trans.IsSend: // Client <-
		f.Value("File to retrieve", trans.SrcFilename)
		f.ValueCond("File saved as", trans.DestFilename)
	}
}

type pair struct {
	key string
	val any
}

func displayTransferInfo(f *Formatter, info map[string]any) {
	if len(info) == 0 {
		return
	}

	f.Title("Transfer values")
	f.Indent()

	defer f.UnIndent()

	displayMap(f, info)
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
