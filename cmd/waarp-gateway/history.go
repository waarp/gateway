package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type historyCommand struct {
	Get     historyGet   `command:"get" description:"Consult a finished transfer"`
	List    historyList  `command:"list" description:"List the finished transfers"`
	Restart historyRetry `command:"retry" description:"Retry a failed transfer"`
}

func displayHistory(w io.Writer, hist *rest.OutHistory) {
	role := "client"
	if hist.IsServer {
		role = "server"
	}
	way := "RECEIVE"
	if hist.IsSend {
		way = "SEND"
	}

	fmt.Fprintln(w, bold("● Transfer", hist.ID, "(as", role+")"), coloredStatus(hist.Status))
	fmt.Fprintln(w, orange("                Way:"), way)
	fmt.Fprintln(w, orange("           Protocol:"), hist.Protocol)
	fmt.Fprintln(w, orange("               Rule:"), hist.Rule)
	fmt.Fprintln(w, orange("          Requester:"), hist.Requester)
	fmt.Fprintln(w, orange("          Requested:"), hist.Requested)
	fmt.Fprintln(w, orange("        Source file:"), hist.SourceFilename)
	fmt.Fprintln(w, orange("   Destination file:"), hist.DestFilename)
	fmt.Fprintln(w, orange("         Start date:"), hist.Start.Format(time.RFC3339))
	fmt.Fprintln(w, orange("           End date:"), hist.Stop.Format(time.RFC3339))
	if hist.ErrorCode != model.TeOk {
		fmt.Fprintln(w, orange("         Error code:"), hist.ErrorCode)
	}
	if hist.ErrorMsg != "" {
		fmt.Fprintln(w, orange("      Error message:"), hist.ErrorMsg)
	}
	if hist.Step != "" {
		fmt.Fprintln(w, orange("        Failed step:"), hist.Step)
	}
	if hist.Progress != 0 {
		fmt.Fprintln(w, orange("           Progress:"), hist.Progress)
	}
	if hist.TaskNumber != 0 {
		fmt.Fprintln(w, orange("        Task number:"), hist.TaskNumber)
	}
}

// ######################## GET ##########################

type historyGet struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (h *historyGet) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.HistoryPath + "/" + fmt.Sprint(h.Args.ID)

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		trans := &rest.OutHistory{}
		if err := unmarshalBody(resp.Body, trans); err != nil {
			return err
		}
		displayHistory(w, trans)
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## LIST ##########################

type historyList struct {
	listOptions
	SortBy    string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start+" choice:"start-" choice:"id+" choice:"id-" choice:"start+" choice:"start-" choice:"rule+" choice:"rule-" choice:"requester+" choice:"requester-" choice:"requested+" choice:"requested-" default:"start+"`
	Requester []string `long:"requester" description:"Filter the transfers based on the transfer's requester. Can be repeated multiple times to filter multiple sources."`
	Requested []string `long:"requested" description:"Filter the transfers based on the transfer's requested. Can be repeated multiple times to filter multiple destinations."`
	Rules     []string `long:"rule" description:"Filter the transfers based on the transfer rule used. Can be repeated multiple times to filter multiple rules."`
	Statuses  []string `long:"status" description:"Filter the transfers based on the transfer's status. Can be repeated multiple times to filter multiple statuses." choice:"DONE" choice:"ERROR" choice:"CANCELLED"`
	Protocol  []string `long:"protocol" description:"Filter the transfers based on the protocol used. Can be repeated multiple times to filter multiple protocols."`
	Start     string   `long:"start" description:"Filter the transfers which started after a given date. Date must be in RFC3339 format."`
	Stop      string   `long:"stop" description:"Filter the transfers which ended before a given date. Date must be in RFC3339 format."`
}

func (h *historyList) listURL() (*url.URL, error) {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return nil, err
	}

	conn.Path = admin.APIPath + rest.HistoryPath
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
		start, err := time.Parse(time.RFC3339, h.Start)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a start valid date (accepted format: '%s')",
				h.Start, time.RFC3339)
		}
		query.Set("start", start.Format(time.RFC3339))
	}
	if h.Stop != "" {
		stop, err := time.Parse(time.RFC3339, h.Stop)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a end valid date (accepted format: '%s')",
				h.Start, time.RFC3339)
		}
		query.Set("stop", stop.Format(time.RFC3339))
	}
	conn.RawQuery = query.Encode()

	return conn, nil
}

func (h *historyList) Execute([]string) error {
	conn, err := h.listURL()
	if err != nil {
		return err
	}

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		body := map[string][]rest.OutHistory{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}
		history := body["history"]
		if len(history) > 0 {
			fmt.Fprintln(w, bold("History:"))
			for _, h := range history {
				history := h
				displayHistory(w, &history)
			}
		} else {
			fmt.Fprintln(w, "No transfers found.")
		}
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## RESTART ##########################

type historyRetry struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
	Date string `short:"d" long:"date" description:"Set the date at which the transfer should restart. Date must be in RFC3339 format."`
}

func (h *historyRetry) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.HistoryPath + "/" + fmt.Sprint(h.Args.ID) + "/retry"

	query := url.Values{}
	if h.Date != "" {
		start, err := time.Parse(time.RFC3339, h.Date)
		if err != nil {
			return fmt.Errorf("'%s' is not a start valid date (accepted format: '%s')",
				h.Date, time.RFC3339)
		}
		query.Set("date", start.Format(time.RFC3339))
	}
	conn.RawQuery = query.Encode()

	resp, err := sendRequest(conn, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		loc, err := resp.Location()
		if err != nil {
			return err
		}
		id := filepath.Base(loc.Path)
		fmt.Fprintln(w, "The transfer will be retried under the ID:", bold(id))
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}
