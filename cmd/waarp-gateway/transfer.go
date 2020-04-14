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
			return cyanBold(string(status))
		case model.StatusRunning:
			return cyanBold(string(status))
		case model.StatusPaused:
			return yellowBold(string(status))
		case model.StatusInterrupted:
			return yellowBold(string(status))
		case model.StatusCancelled:
			return redBold(string(status))
		case model.StatusError:
			return redBold(string(status))
		case model.StatusDone:
			return greenBold(string(status))
		default:
			return whiteBold(string(status))
		}
	}()
	return whiteBold("[") + text + whiteBold("]")
}

func displayTransfer(w io.Writer, trans *rest.OutTransfer) {
	role := "client"
	if trans.IsServer {
		role = "server"
	}

	fmt.Fprintln(w, whiteBold("● Transfer"), whiteBoldUL(fmt.Sprint(trans.ID)),
		whiteBold("(as ", role, ")"), coloredStatus(trans.Status))
	fmt.Fprintln(w, whiteBold("  -Rule:            "), white(trans.Rule))
	fmt.Fprintln(w, whiteBold("  -Requester:       "), white(trans.Requester))
	fmt.Fprintln(w, whiteBold("  -Requested:       "), white(trans.Requested))
	fmt.Fprintln(w, whiteBold("  -True filepath:   "), white(trans.TrueFilepath))
	fmt.Fprintln(w, whiteBold("  -Source file:     "), white(trans.SourcePath))
	fmt.Fprintln(w, whiteBold("  -Destination file:"), white(trans.DestPath))
	fmt.Fprintln(w, whiteBold("  -Start time:      "), white(trans.Start.Format(time.RFC3339)))
	fmt.Fprintln(w, whiteBold("  -Step:            "), white(string(trans.Step)))
	fmt.Fprintln(w, whiteBold("  -Progress:        "), white(fmt.Sprint(trans.Progress)))
	fmt.Fprintln(w, whiteBold("  -Task number:     "), white(fmt.Sprint(trans.TaskNumber)))
	if trans.ErrorCode != model.TeOk {
		fmt.Fprintln(w, whiteBold("  -Error code:      "), red(fmt.Sprint(trans.ErrorCode)))
	}
	if trans.ErrorMsg != "" {
		fmt.Fprintln(w, whiteBold("  -Error message:   "), red(trans.ErrorMsg))
	}
}

// ######################## ADD ##########################

type transferAdd struct {
	File    string `required:"true" short:"f" long:"file" description:"The file to transfer"`
	Way     string `required:"true" short:"w" long:"way" description:"The direction of the transfer" choice:"pull" choice:"push"`
	Dest    string `short:"d" long:"dest" description:"The name of the file after the transfer"`
	Partner string `required:"true" short:"p" long:"partner" description:"The partner with which the transfer is performed"`
	Account string `required:"true" short:"a" long:"account" description:"The account used to connect on the partner"`
	Rule    string `required:"true" short:"r" long:"rule" description:"The rule to use for the transfer"`
}

func (t *transferAdd) Execute([]string) error {
	if t.Dest == "" {
		t.Dest = t.File
	}
	trans := rest.InTransfer{
		Partner:    t.Partner,
		Account:    t.Account,
		IsSend:     t.Way == "push",
		SourcePath: t.File,
		Rule:       t.Rule,
		DestPath:   t.Dest,
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath

	resp, err := sendRequest(conn, trans, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, whiteBold("The transfer of file '")+whiteBoldUL(t.File)+
			whiteBold("' was successfully added."))
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## GET ##########################

type transferGet struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *transferGet) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath + "/" + fmt.Sprint(t.Args.ID)

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		trans := &rest.OutTransfer{}
		if err := unmarshalBody(resp.Body, trans); err != nil {
			return err
		}
		displayTransfer(w, trans)
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
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
	conn, err := url.Parse(commandLine.Args.Address)
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
	conn, err := t.listURL()
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
		body := map[string][]rest.OutTransfer{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}
		transfers := body["transfers"]
		if len(transfers) > 0 {
			fmt.Fprintln(w, yellowBold("Transfers:"))
			for _, t := range transfers {
				transfer := t
				displayTransfer(w, &transfer)
			}
		} else {
			fmt.Fprintln(w, yellow("No transfers found."))
		}
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## PAUSE ##########################

type transferPause struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *transferPause) Execute([]string) error {
	id := fmt.Sprint(t.Args.ID)
	conn, err := url.Parse(commandLine.Args.Address)
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
		fmt.Fprintln(w, whiteBold("The transfer ")+whiteBoldUL(id)+whiteBold(
			" was successfully paused. It can be resumed using the 'resume' command."))
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
	conn, err := url.Parse(commandLine.Args.Address)
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
		fmt.Fprintln(w, whiteBold("The transfer ")+whiteBoldUL(id)+
			whiteBold(" was successfully resumed."))
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
	conn, err := url.Parse(commandLine.Args.Address)
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
		fmt.Fprintln(w, whiteBold("The transfer ")+whiteBoldUL(id)+whiteBold(
			" was successfully cancelled. It was moved to the transfer history."))
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}
