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

func DisplayHistory(w io.Writer, hist *api.OutHistory) {
	f := NewFormatter(w)
	defer f.Render()

	displayHistory(f, hist)
}

//nolint:varnamelen //formatter name is kept short for readability
func displayHistory(f *Formatter, hist *api.OutHistory) {
	role := roleClient
	if hist.IsServer {
		role = roleServer
	}

	size := sizeUnknown
	if hist.Filesize >= 0 {
		size = utils.FormatInt(hist.Filesize)
	}

	stop := NotApplicable
	if hist.Stop.Valid {
		stop = hist.Stop.Value.Local().Format(time.RFC3339Nano)
	}

	title := f.titleColor().Sprintf("Transfer %d (%s as %s)", hist.ID, role,
		direction(hist.IsSend))

	f.PlainTitle(title + " " + coloredStatus(hist.Status))
	f.Indent()

	defer f.UnIndent()

	f.ValueCond("Remote ID", hist.RemoteID)
	f.Value("Way", direction(hist.IsSend))
	f.Value("Protocol", hist.Protocol)
	f.Value("Rule", hist.Rule)
	f.Value("Requester", hist.Requester)
	f.Value("Requested", hist.Requested)
	f.Value("Local filepath", hist.LocalFilepath)
	f.Value("Remote filepath", hist.RemoteFilepath)
	f.Value("File size", size)
	f.Value("Start date", hist.Start.Format(time.RFC3339Nano))
	f.Value("End date", stop)

	if hist.ErrorCode != types.TeOk {
		f.Value("Error code", hist.ErrorCode.String())
		f.ValueCond("Error message", hist.ErrorMsg)
	}

	if hist.Step != types.StepNone {
		f.Value("Failed step", hist.Step.String())

		if hist.Step == types.StepData {
			f.Value("Progress", hist.Progress)
		} else if hist.Step == types.StepPreTasks || hist.Step == types.StepPostTasks {
			f.Value("Task number", hist.TaskNumber)
		}
	}

	displayTransferInfo(f, hist.TransferInfo)
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

	DisplayHistory(w, trans)

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
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("History:")

		for _, entry := range history {
			displayHistory(f, entry)
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
