package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type transferCommand struct {
	Add    transferAdd    `command:"add" description:"Add a new transfer to be executed"`
	Get    transferGet    `command:"get" description:"Consult a transfer"`
	List   transferList   `command:"list" description:"List the transfers"`
	Pause  transferPause  `command:"pause" description:"Pause a running transfer"`
	Resume transferResume `command:"resume" description:"Resume a paused transfer"`
	Cancel transferCancel `command:"cancel" description:"Cancel a transfer"`
}

func coloredStatus(status model.TransferStatus) string {
	text := func() string {
		switch status {
		case model.StatusPlanned:
			return cyan(string(status))
		case model.StatusRunning:
			return cyan(string(status))
		case model.StatusPaused:
			return yellow(string(status))
		case model.StatusInterrupted:
			return yellow(string(status))
		case model.StatusCancelled:
			return red(string(status))
		case model.StatusError:
			return red(string(status))
		case model.StatusDone:
			return green(string(status))
		default:
			return bold(string(status))
		}
	}()
	return bold("[") + text + bold("]")
}

func displayTransfer(w io.Writer, trans *rest.OutTransfer) {
	role := "client"
	if trans.IsServer {
		role = "server"
	}

	fmt.Fprintln(w, bold("● Transfer", trans.ID, "(as "+role+")"), coloredStatus(trans.Status))
	fmt.Fprintln(w, orange("    Rule:            "), trans.Rule)
	fmt.Fprintln(w, orange("    Requester:       "), trans.Requester)
	fmt.Fprintln(w, orange("    Requested:       "), trans.Requested)
	fmt.Fprintln(w, orange("    True filepath:   "), trans.TrueFilepath)
	fmt.Fprintln(w, orange("    Source file:     "), trans.SourcePath)
	fmt.Fprintln(w, orange("    Destination file:"), trans.DestPath)
	fmt.Fprintln(w, orange("    Start time:      "), trans.Start.Format(time.RFC3339))
	fmt.Fprintln(w, orange("    Step:            "), string(trans.Step))
	fmt.Fprintln(w, orange("    Progress:        "), trans.Progress)
	fmt.Fprintln(w, orange("    Task number:     "), trans.TaskNumber)
	if trans.ErrorCode != model.TeOk {
		fmt.Fprintln(w, orange("    Error code:      "), fmt.Sprint(trans.ErrorCode))
	}
	if trans.ErrorMsg != "" {
		fmt.Fprintln(w, orange("    Error message:   "), trans.ErrorMsg)
	}
}

// ######################## ADD ##########################

type transferAdd struct {
	File    string `required:"true" short:"f" long:"file" description:"The file to transfer"`
	Way     string `required:"true" short:"w" long:"way" description:"The direction of the transfer" choice:"pull" choice:"push"`
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

	trans := rest.InTransfer{
		Partner:    t.Partner,
		Account:    t.Account,
		IsSend:     t.Way == "push",
		SourcePath: t.File,
		Rule:       t.Rule,
		DestPath:   t.Name,
	}
	if t.Date != "" {
		trans.Start, err = time.Parse(time.RFC3339, t.Date)
		if err != nil {
			return fmt.Errorf("'%s' is not a valid date", t.Date)
		}
	}
	path := admin.APIPath + rest.TransfersPath

	if err := add(path, trans); err != nil {
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
	path := admin.APIPath + rest.TransfersPath + "/" + fmt.Sprint(t.Args.ID)

	trans := &rest.OutTransfer{}
	if err := get(path, trans); err != nil {
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

func (t *transferList) listURL() (*url.URL, error) {
	conn, err := url.Parse(commandLine.Address)
	if err != nil {
		return nil, err
	}

	conn.Path = admin.APIPath + rest.TransfersPath
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
		_, err := time.Parse(time.RFC3339, t.Start)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid date (accepted format: '%s')",
				t.Start, time.RFC3339)
		}
		query.Set("start", t.Start)
	}
	conn.RawQuery = query.Encode()

	return conn, nil
}

func (t *transferList) Execute([]string) error {
	addr, err := t.listURL()
	if err != nil {
		return err
	}

	body := map[string][]rest.OutTransfer{}
	if err := list(addr, &body); err != nil {
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
	conn, err := url.Parse(commandLine.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath + "/" + id + "/pause"

	resp, err := sendRequest(conn, nil, http.MethodPut)
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
	conn, err := url.Parse(commandLine.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath + "/" + id + "/resume"

	resp, err := sendRequest(conn, nil, http.MethodPut)
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
	conn, err := url.Parse(commandLine.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath + "/" + id + "/cancel"

	resp, err := sendRequest(conn, nil, http.MethodPut)
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
