package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
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
	role := roleClient
	if trans.IsServer {
		role = roleServer
	}

	dir := directionRecv
	if trans.IsSend {
		dir = directionSend
	}

	size := sizeUnknown
	if trans.Filesize >= 0 {
		size = fmt.Sprint(trans.Filesize)
	}

	fmt.Fprintln(w, bold("● Transfer", trans.ID, "("+dir+" as "+role+")"), coloredStatus(trans.Status))

	if trans.RemoteID != "" {
		fmt.Fprintln(w, orange("    Remote ID:      "), trans.RemoteID)
	}

	fmt.Fprintln(w, orange("    Rule:           "), trans.Rule)
	fmt.Fprintln(w, orange("    Protocol:       "), trans.Protocol)
	fmt.Fprintln(w, orange("    Requester:      "), trans.Requester)
	fmt.Fprintln(w, orange("    Requested:      "), trans.Requested)
	fmt.Fprintln(w, orange("    Local filepath: "), trans.LocalFilepath)
	fmt.Fprintln(w, orange("    Remote filepath:"), trans.RemoteFilepath)
	fmt.Fprintln(w, orange("    File size:      "), size)
	fmt.Fprintln(w, orange("    Start date:     "), trans.Start.Format(time.RFC3339Nano))
	fmt.Fprintln(w, orange("    Step:           "), trans.Step)
	fmt.Fprintln(w, orange("    Progress:       "), trans.Progress)
	fmt.Fprintln(w, orange("    Task number:    "), trans.TaskNumber)

	if trans.ErrorCode != types.TeOk.String() {
		fmt.Fprintln(w, orange("    Error code:     "), fmt.Sprint(trans.ErrorCode))
	}

	if trans.ErrorMsg != "" {
		fmt.Fprintln(w, orange("    Error message:  "), trans.ErrorMsg)
	}
}

// ######################## ADD ##########################

//nolint:lll // struct tags can be long for command line args
type transferAdd struct {
	File    string `required:"yes" short:"f" long:"file" description:"The file to transfer"`
	Way     string `required:"yes" short:"w" long:"way" description:"The direction of the transfer" choice:"send" choice:"receive"`
	Partner string `required:"yes" short:"p" long:"partner" description:"The partner with which the transfer is performed"`
	Account string `required:"yes" short:"l" long:"login" description:"The login of the account used to connect on the partner"`
	Rule    string `required:"yes" short:"r" long:"rule" description:"The rule to use for the transfer"`
	Date    string `short:"d" long:"date" description:"The starting date (in ISO 8601 format) of the transfer"`

	Name *string `short:"n" long:"name" description:"[DEPRECATED] The name of the file after the transfer"` // Deprecated: the source name is used instead
}

func (t *transferAdd) Execute([]string) (err error) {
	trans := api.InTransfer{
		Partner: t.Partner,
		Account: t.Account,
		IsSend:  dirToBoolPtr(t.Way),
		File:    t.File,
		Rule:    t.Rule,
	}

	if t.Name != nil {
		fmt.Fprintln(out, "[WARNING] The '-n' ('--name') option is deprecated. "+
			"For simplicity, in the future, files will have the same name at "+
			"the source and the destination")

		trans.DestPath = *t.Name
	}

	if t.Date != "" {
		trans.Start, err = time.Parse(time.RFC3339Nano, t.Date)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid date: %w", t.Date, errInvalidDate)
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

//nolint:lll // struct tags can be long for command line args
type transferList struct {
	listOptions
	SortBy   string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start+" choice:"start-" choice:"id+" choice:"id-" choice:"rule+" choice:"rule-" default:"start+"`
	Rules    []string `short:"r" long:"rule" description:"Filter the transfers based on the name of the transfer rule used. Can be repeated multiple times to filter multiple rules."`
	Statuses []string `short:"t" long:"status" description:"Filter the transfers based on the transfer's status. Can be repeated multiple times to filter multiple statuses." choice:"PLANNED" choice:"RUNNING" choice:"INTERRUPTED" choice:"PAUSED"`
	Start    string   `short:"d" long:"date" description:"Filter the transfers which started after a given date. Date must be in ISO 8601 format."`
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
			return fmt.Errorf("'%s' is not a valid date (accepted format: '%s'): %w",
				t.Start, time.RFC3339Nano, errInvalidDate)
		}

		query.Set("start", t.Start)
	}

	addr.RawQuery = query.Encode()

	return nil
}

//nolint:dupl // duplicated code is about two different types
func (t *transferList) Execute([]string) error {
	if err := t.listURL(); err != nil {
		return err
	}

	body := map[string][]api.OutTransfer{}
	if err := list(&body); err != nil {
		return err
	}

	w := getColorable() //nolint:ifshort // decrease readability

	if transfers := body["transfers"]; len(transfers) > 0 {
		fmt.Fprintln(w, bold("Transfers:"))

		for i := range transfers {
			transfer := transfers[i]
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

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // no logger to handle the error

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
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseMessage(resp))
	}
}

// ######################## RESUME ##########################

type transferResume struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

//nolint:dupl // hard to factorize
func (t *transferResume) Execute([]string) error {
	id := fmt.Sprint(t.Args.ID)
	addr.Path = fmt.Sprintf("/api/transfers/%d/resume", t.Args.ID)

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

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
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseMessage(resp))
	}
}

// ######################## CANCEL ##########################

type transferCancel struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

//nolint:dupl // hard to factorize
func (t *transferCancel) Execute([]string) error {
	id := fmt.Sprint(t.Args.ID)
	addr.Path = fmt.Sprintf("/api/transfers/%d/cancel", t.Args.ID)

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}

	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	w := getColorable()

	switch resp.StatusCode {
	case http.StatusAccepted:
		fmt.Fprintln(w, "The transfer", bold(id), "was successfully canceled.")

		return nil

	case http.StatusNotFound:
		return getResponseMessage(resp)

	case http.StatusBadRequest:
		return getResponseMessage(resp)

	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status, getResponseMessage(resp))
	}
}
