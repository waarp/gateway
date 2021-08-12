package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

type transferCommand struct {
	Add    transferAdd    `command:"add" description:"Add a new transfer to be executed"`
	Get    transferGet    `command:"get" description:"Consult a transfer"`
	List   transferList   `command:"list" description:"List the transfers"`
	Pause  transferPause  `command:"pause" description:"Pause a running transfer"`
	Resume transferResume `command:"resume" description:"Resume a paused transfer"`
	Cancel transferCancel `command:"cancel" description:"Cancel a transfer"`
}

func coloredStatus(status types.TransferStatus) string {
	text := func() string {
		switch status {
		case types.StatusPlanned:
			return cyan(string(status))
		case types.StatusRunning:
			return cyan(string(status))
		case types.StatusPaused:
			return yellow(string(status))
		case types.StatusInterrupted:
			return yellow(string(status))
		case types.StatusCancelled:
			return red(string(status))
		case types.StatusError:
			return red(string(status))
		case types.StatusDone:
			return green(string(status))
		default:
			return bold(string(status))
		}
	}()
	return bold("[") + text + bold("]")
}

func displayTransfer(w io.Writer, trans *api.OutTransfer) {
	role := "client"
	if trans.IsServer {
		role = "server"
	}
	dir := "receive"
	if trans.IsSend {
		dir = "send"
	}

	fmt.Fprintln(w, bold("● Transfer", trans.ID, "("+dir+" as "+role+")"), coloredStatus(trans.Status))
	if trans.RemoteID != "" {
		fmt.Fprintln(w, orange("    Remote ID:       "), trans.RemoteID)
	}
	fmt.Fprintln(w, orange("    Rule:            "), trans.Rule)
	fmt.Fprintln(w, orange("    Requester:       "), trans.Requester)
	fmt.Fprintln(w, orange("    Requested:       "), trans.Requested)
	fmt.Fprintln(w, orange("    True filepath:   "), trans.TrueFilepath)
	fmt.Fprintln(w, orange("    Source file:     "), trans.SourcePath)
	fmt.Fprintln(w, orange("    Destination file:"), trans.DestPath)
	fmt.Fprintln(w, orange("    Start time:      "), trans.Start.Format(time.RFC3339Nano))
	fmt.Fprintln(w, orange("    Step:            "), trans.Step)
	fmt.Fprintln(w, orange("    Progress:        "), trans.Progress)
	fmt.Fprintln(w, orange("    Task number:     "), trans.TaskNumber)
	if trans.ErrorCode != types.TeOk.String() {
		fmt.Fprintln(w, orange("    Error code:      "), fmt.Sprint(trans.ErrorCode))
	}
	if trans.ErrorMsg != "" {
		fmt.Fprintln(w, orange("    Error message:   "), trans.ErrorMsg)
	}
}

// ######################## ADD ##########################

type transferAdd struct {
	File    string `required:"true" short:"f" long:"file" description:"The file to transfer"`
	Way     string `required:"true" short:"w" long:"way" description:"The direction of the transfer" choice:"send" choice:"receive"`
	Name    string `short:"n" long:"name" description:"The name of the file after the transfer"`
	Partner string `required:"true" short:"p" long:"partner" description:"The partner with which the transfer is performed"`
	Account string `required:"true" short:"l" long:"login" description:"The login of the account used to connect on the partner"`
	Rule    string `required:"true" short:"r" long:"rule" description:"The rule to use for the transfer"`
	Date    string `short:"d" long:"date" description:"The starting date (in ISO 8601 format) of the transfer"`
}

func (t *transferAdd) Execute([]string) (err error) {
	if t.Name == "" {
		t.Name = t.File
	}

	trans := api.InTransfer{
		Partner:    t.Partner,
		Account:    t.Account,
		IsSend:     utils.BoolPtr(t.Way == "send"),
		SourcePath: t.File,
		Rule:       t.Rule,
		DestPath:   t.Name,
	}
	if t.Date != "" {
		trans.Start, err = time.Parse(time.RFC3339Nano, t.Date)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid date", t.Date)
		}
	}
	addr.Path = "/api/transfers"

	if err := add(trans); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The transfer of file", t.File, "was successfully added.")
	return nil
}

// ######################## GET ##########################

type transferGet struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *transferGet) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/transfers/%d", t.Args.ID)

	trans := &api.OutTransfer{}
	if err := get(trans); err != nil {
		return err
	}
	displayTransfer(getColorable(), trans)
	return nil
}

// ######################## LIST ##########################

type transferList struct {
	listOptions
	SortBy   string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start+" choice:"start-" choice:"id+" choice:"id-" choice:"rule+" choice:"rule-" default:"start+"`
	Rules    []string `short:"r" long:"rule" description:"Filter the transfers based on the ID of the transfer rule used. Can be repeated multiple times to filter multiple rules."`
	Statuses []string `short:"t" long:"status" description:"Filter the transfers based on the transfer's status. Can be repeated multiple times to filter multiple statuses." choice:"PLANNED" choice:"RUNNING" choice:"INTERRUPTED" choice:"PAUSED"`
	Start    string   `short:"d" long:"date" description:"Filter the transfers which started after a given date. Date must be in RFC3339 format."`
}

func (t *transferList) listURL() error {
	addr.Path = "/api/transfers"
	query := url.Values{}
	query.Set("limit", fmt.Sprint(t.Limit))
	query.Set("offset", fmt.Sprint(t.Offset))
	query.Set("sort", t.SortBy)

	for _, rule := range t.Rules {
		query.Add("rule", rule)
	}
	for _, status := range t.Statuses {
		query.Add("status", status)
	}
	if t.Start != "" {
		_, err := time.Parse(time.RFC3339Nano, t.Start)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid date (accepted format: '%s')",
				t.Start, time.RFC3339Nano)
		}
		query.Set("start", t.Start)
	}
	addr.RawQuery = query.Encode()

	return nil
}

func (t *transferList) Execute([]string) error {
	if err := t.listURL(); err != nil {
		return err
	}

	body := map[string][]api.OutTransfer{}
	if err := list(&body); err != nil {
		return err
	}

	transfers := body["transfers"]
	w := getColorable()
	if len(transfers) > 0 {
		fmt.Fprintln(w, bold("Transfers:"))
		for _, t := range transfers {
			transfer := t
			displayTransfer(w, &transfer)
		}
	} else {
		fmt.Fprintln(w, "No transfers found.")
	}
	return nil
}

// ######################## PAUSE ##########################

type transferPause struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *transferPause) Execute([]string) error {
	id := fmt.Sprint(t.Args.ID)
	addr.Path = fmt.Sprintf("/api/transfers/%d/pause", t.Args.ID)

	resp, err := sendRequest(nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusAccepted:
		fmt.Fprintln(w, "The transfer", bold(id), "was successfully paused.",
			"It can be resumed using the 'resume' command.")
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## RESUME ##########################

type transferResume struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *transferResume) Execute([]string) error {
	id := fmt.Sprint(t.Args.ID)
	addr.Path = fmt.Sprintf("/api/transfers/%d/resume", t.Args.ID)

	resp, err := sendRequest(nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusAccepted:
		fmt.Fprintln(w, "The transfer", bold(id), "was successfully resumed.")
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## CANCEL ##########################

type transferCancel struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *transferCancel) Execute([]string) error {
	id := fmt.Sprint(t.Args.ID)
	addr.Path = fmt.Sprintf("/api/transfers/%d/cancel", t.Args.ID)

	resp, err := sendRequest(nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusAccepted:
		fmt.Fprintln(w, "The transfer", bold(id), "was successfully cancelled.")
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}
