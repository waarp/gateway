package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

type historyCommand struct {
	Get     historyGet   `command:"get" description:"Consult a finished transfer"`
	List    historyList  `command:"list" description:"List the finished transfers"`
	Restart historyRetry `command:"retry" description:"Retry a failed transfer"`
}

func displayHistory(w io.Writer, hist *api.OutHistory) {
	role := roleClient

	if hist.IsServer {
		role = roleServer
	}

	way := directionRecv

	if hist.IsSend {
		way = directionSend
	}

	size := sizeUnknown

	if hist.Filesize >= 0 {
		size = fmt.Sprint(hist.Filesize)
	}

	stop := "N/A"

	if hist.Stop != nil {
		stop = hist.Stop.Local().Format(time.RFC3339Nano)
	}

	fmt.Fprintln(w, orange(bold("● Transfer", hist.ID, "(as", role+")")), coloredStatus(hist.Status))

	if hist.RemoteID != "" {
		fmt.Fprintln(w, orange("    Remote ID:            "), hist.RemoteID)
	}

	fmt.Fprintln(w, orange("    Way:            "), way)
	fmt.Fprintln(w, orange("    Protocol:       "), hist.Protocol)
	fmt.Fprintln(w, orange("    Rule:           "), hist.Rule)
	fmt.Fprintln(w, orange("    Requester:      "), hist.Requester)
	fmt.Fprintln(w, orange("    Requested:      "), hist.Requested)
	fmt.Fprintln(w, orange("    Local filepath: "), hist.LocalFilepath)
	fmt.Fprintln(w, orange("    Remote filepath:"), hist.RemoteFilepath)
	fmt.Fprintln(w, orange("    File size:      "), size)
	fmt.Fprintln(w, orange("    Start date:     "), hist.Start.Format(time.RFC3339Nano))
	fmt.Fprintln(w, orange("    End date:       "), stop)

	if hist.ErrorCode != types.TeOk {
		fmt.Fprintln(w, orange("    Error code:     "), hist.ErrorCode)

		if hist.ErrorMsg != "" {
			fmt.Fprintln(w, orange("    Error message:  "), hist.ErrorMsg)
		}
	}

	if hist.Step != types.StepNone {
		fmt.Fprintln(w, orange("    Failed step:    "), hist.Step.String())

		switch hist.Step { //nolint:exhaustive // those are the only relevant cases here
		case types.StepData:
			fmt.Fprintln(w, orange("    Progress:       "), hist.Progress)
		case types.StepPreTasks, types.StepPostTasks:
			fmt.Fprintln(w, orange("    Failed task:    "), hist.TaskNumber)
		}
	}
}

// ######################## GET ##########################

type historyGet struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (h *historyGet) Execute([]string) error {
	addr.Path = path.Join("/api/history/", fmt.Sprint(h.Args.ID))

	trans := &api.OutHistory{}
	if err := get(trans); err != nil {
		return err
	}

	displayHistory(getColorable(), trans)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags can be long for command line args
type historyList struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start+" choice:"start-" choice:"id+" choice:"id-" choice:"start+" choice:"start-" choice:"rule+" choice:"rule-" choice:"requester+" choice:"requester-" choice:"requested+" choice:"requested-" default:"start+"`
	Requester []string `short:"q" long:"requester" description:"Filter the transfers based on the transfer's requester. Can be repeated multiple times to filter multiple sources."`
	Requested []string `short:"d" long:"requested" description:"Filter the transfers based on the transfer's requested. Can be repeated multiple times to filter multiple destinations."`
	Rules     []string `short:"r" long:"rule" description:"Filter the transfers based on the transfer rule used. Can be repeated multiple times to filter multiple rules."`
	Statuses  []string `short:"t" long:"status" description:"Filter the transfers based on the transfer's status. Can be repeated multiple times to filter multiple statuses." choice:"DONE" choice:"ERROR" choice:"CANCELED"`
	Protocol  []string `short:"p" long:"protocol" description:"Filter the transfers based on the protocol used. Can be repeated multiple times to filter multiple protocols."`
	Start     string   `short:"b" long:"start" description:"Filter the transfers which started after a given date. Date must be in RFC3339 format."`
	Stop      string   `short:"e" long:"stop" description:"Filter the transfers which ended before a given date. Date must be in RFC3339 format."`
}

func (h *historyList) listURL() error {
	addr.Path = "/api/history"
	query := url.Values{}
	query.Set("limit", fmt.Sprint(h.Limit))
	query.Set("offset", fmt.Sprint(h.Offset))
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

func (h *historyList) Execute([]string) error {
	if err := h.listURL(); err != nil {
		return err
	}

	body := map[string][]api.OutHistory{}
	if err := list(&body); err != nil {
		return err
	}

	history := body["history"]
	w := getColorable() //nolint:ifshort // decrease readability

	if len(history) > 0 {
		fmt.Fprintln(w, bold("History:"))

		for i := range history {
			displayHistory(w, &history[i])
		}
	} else {
		fmt.Fprintln(w, "No transfers found.")
	}

	return nil
}

// ######################## RESTART ##########################

//nolint:lll // struct tags can be long for command line args
type historyRetry struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
	Date string `short:"d" long:"date" description:"Set the date at which the transfer should restart. Date must be in RFC3339 format."`
}

func (h *historyRetry) Execute([]string) error {
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

	resp, err := sendRequest(nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	w := getColorable()

	switch resp.StatusCode {
	case http.StatusCreated:
		loc, err := resp.Location()
		if err != nil {
			return fmt.Errorf("cannot get the resource location: %w", err)
		}

		id := filepath.Base(loc.Path)
		fmt.Fprintln(w, "The transfer will be retried under the ID:", bold(id))

		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseMessage(resp))
	}
}
