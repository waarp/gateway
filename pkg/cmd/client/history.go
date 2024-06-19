package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func displayHistory(w io.Writer, hist *api.OutHistory) {
	fmt.Fprintf(w, "%s%s (%s as %s) [%s]\n", style1.bulletPrefix,
		style1.color.Sprintf("Transfer %d", hist.ID),
		direction(hist.IsSend), transferRole(hist.IsServer),
		coloredStatus(hist.Status))

	style22.printL(w, "Remote ID", hist.RemoteID)
	style22.printL(w, "Protocol", hist.Protocol)
	style22.printL(w, "Rule", hist.Rule)
	style22.printL(w, "Requested by", hist.Requester)
	style22.printL(w, "Requested to", hist.Requested)
	style22.printL(w, "Full local path", hist.LocalFilepath)
	style22.printL(w, "Full remote path", hist.RemoteFilepath)
	style22.printL(w, "File size", prettyBytes(hist.Filesize))
	style22.printL(w, "Start date", hist.Start.Local().String())
	style22.printL(w, "End date",
		ifElse(hist.Stop.Valid, hist.Stop.Value.Local().String(), notApplicable))
	style22.printL(w, "Data transferred", prettyBytes(hist.Progress))

	if hist.Step != types.StepNone {
		style22.printL(w, "Current step", hist.Step)
	}

	style22.option(w, "Current task", cardinal(hist.TaskNumber))

	if hist.ErrorCode != types.TeOk {
		style22.printL(w, "Error code", hist.ErrorCode)
		style22.printL(w, "Error message", hist.ErrorMsg)
	}

	displayTransferInfo(w, hist.TransferInfo)
}

// ######################## GET ##########################

type HistoryGet struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (h *HistoryGet) Execute([]string) error { return h.execute(stdOutput) }
func (h *HistoryGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/history/", utils.FormatUint(h.Args.ID))

	trans := &api.OutHistory{}
	if err := get(trans); err != nil {
		return err
	}

	displayHistory(w, trans)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags can be long for command line args
type HistoryList struct {
	ListOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start+" choice:"start-" choice:"id+" choice:"id-" choice:"start+" choice:"start-" choice:"stop+" choice:"stop-" choice:"rule+" choice:"rule-" choice:"requester+" choice:"requester-" choice:"requested+" choice:"requested-" default:"start+"`
	Requester []string `short:"q" long:"requester" description:"Filter the transfers based on the transfer's requester. Can be repeated multiple times to filter multiple sources."`
	Requested []string `short:"d" long:"requested" description:"Filter the transfers based on the transfer's requested. Can be repeated multiple times to filter multiple destinations."`
	Rules     []string `short:"r" long:"rule" description:"Filter the transfers based on the transfer rule used. Can be repeated multiple times to filter multiple rules."`
	Statuses  []string `short:"t" long:"status" description:"Filter the transfers based on the transfer's status. Can be repeated multiple times to filter multiple statuses." choice:"DONE" choice:"ERROR" choice:"CANCELED"`
	Protocol  []string `short:"p" long:"protocol" description:"Filter the transfers based on the protocol used. Can be repeated multiple times to filter multiple protocols."`
	Start     string   `short:"b" long:"start" description:"Filter the transfers which started after a given date. Date must be in RFC3339 format."`
	Stop      string   `short:"e" long:"stop" description:"Filter the transfers which ended before a given date. Date must be in RFC3339 format."`
}

func (h *HistoryList) listURL() error {
	addr.Path = "/api/history"
	query := url.Values{}
	query.Set("limit", utils.FormatUint(h.Limit))
	query.Set("offset", utils.FormatUint(h.Offset))
	query.Set("sort", h.SortBy)

	for _, acc := range h.Requester {
		query.Add("requester", acc)
	}

	for _, agent := range h.Requested {
		query.Add("requested", agent)
	}

	for _, rul := range h.Rules {
		query.Add("rule", rul)
	}

	for _, prt := range h.Protocol {
		query.Add("protocol", prt)
	}

	for _, sta := range h.Statuses {
		query.Add("status", sta)
	}

	if h.Start != "" {
		start, err := time.Parse(time.RFC3339Nano, h.Start)
		if err != nil {
			return fmt.Errorf("'%s' is not a start valid date (accepted format: '%s'): %w",
				h.Start, time.RFC3339Nano, err)
		}

		query.Set("start", start.Format(time.RFC3339Nano))
	}

	if h.Stop != "" {
		stop, err := time.Parse(time.RFC3339Nano, h.Stop)
		if err != nil {
			return fmt.Errorf("'%s' is not a end valid date (accepted format: '%s'): %w",
				h.Start, time.RFC3339Nano, err)
		}

		query.Set("stop", stop.Format(time.RFC3339Nano))
	}

	addr.RawQuery = query.Encode()

	return nil
}

func (h *HistoryList) Execute([]string) error { return h.execute(stdOutput) }

//nolint:dupl //history & transfer commands should be kept separate for future-proofing
func (h *HistoryList) execute(w io.Writer) error {
	if err := h.listURL(); err != nil {
		return err
	}

	body := map[string][]*api.OutHistory{}
	if err := list(&body); err != nil {
		return err
	}

	if history := body["history"]; len(history) > 0 {
		style0.printf(w, "=== History ===")

		for _, entry := range history {
			displayHistory(w, entry)
		}
	} else {
		fmt.Fprintln(w, "No transfers found.")
	}

	return nil
}

// ######################## RETRY ##########################

//nolint:lll // struct tags can be long for command line args
type HistoryRetry struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
	Date string `short:"d" long:"date" description:"Set the date at which the transfer should restart. Date must be in RFC3339 format."`
}

func (h *HistoryRetry) Execute([]string) error { return h.execute(stdOutput) }

//nolint:dupl //history & transfer commands should be kept separate for future-proofing
func (h *HistoryRetry) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/history/%d/retry", h.Args.ID)

	query := url.Values{}

	if h.Date != "" {
		start, err := time.Parse(time.RFC3339Nano, h.Date)
		if err != nil {
			return fmt.Errorf("'%s' is not a start valid date (accepted format: '%s'): %w",
				h.Date, time.RFC3339Nano, err)
		}

		query.Set("date", start.Format(time.RFC3339Nano))
	}

	addr.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	switch resp.StatusCode {
	case http.StatusCreated:
		loc, err := resp.Location()
		if err != nil {
			return fmt.Errorf("cannot get the resource location: %w", err)
		}

		id := filepath.Base(loc.Path)
		fmt.Fprintf(w, "The transfer will be retried under the ID %q\n", id)

		return nil
	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseErrorMessage(resp))
	}
}
