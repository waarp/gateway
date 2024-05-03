package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func transferRole(isServer bool) string {
	return utils.If[string](isServer, roleServer, roleClient)
}

func displayTransfer(w io.Writer, trans *api.OutTransfer) {
	fmt.Fprintf(w, "%s%s (%s as %s) [%s]\n", style1.bulletPrefix,
		style1.color.Sprintf("Transfer %d", trans.ID),
		direction(trans.IsSend), transferRole(trans.IsServer),
		coloredStatus(trans.Status))

	style22.printL(w, "Remote ID", trans.RemoteID)
	style22.printL(w, "Protocol", trans.Protocol)

	switch {
	case trans.IsServer && trans.IsSend: // <- Server
		style22.printL(w, "File pulled", trans.SrcFilename)
	case trans.IsServer && !trans.IsSend: // -> Server
		style22.printL(w, "File pushed", trans.DestFilename)
	case !trans.IsServer && trans.IsSend: // Client ->
		style22.printL(w, "File to send", trans.SrcFilename)
		style22.printL(w, "File deposited as", trans.DestFilename)
	case !trans.IsServer && !trans.IsSend: // Client <-
		style22.printL(w, "File to receive", trans.SrcFilename)
		style22.printL(w, "File retrieved as", trans.DestFilename)
	}

	style22.printL(w, "Rule", trans.Rule)
	style22.printL(w, "Requested by", trans.Requester)
	style22.printL(w, "Requested to", trans.Requested)
	style22.option(w, "With client", trans.Client)
	style22.printL(w, "Full local path", trans.LocalFilepath)
	style22.printL(w, "Full remote path", trans.RemoteFilepath)
	style22.printL(w, "File size", prettyBytes(trans.Filesize))
	style22.printL(w, "Start date", trans.Start.Local().String())
	style22.printL(w, "End date",
		ifElse(trans.Stop.Valid, trans.Stop.Value.Local().String(), notApplicable))
	style22.printL(w, "Data transferred", prettyBytes(trans.Progress))

	if trans.Step != "" && trans.Step != types.StepNone.String() {
		style22.printL(w, "Current step", trans.Step)
	}

	style22.option(w, "Current task", cardinal(trans.TaskNumber))

	if trans.ErrorCode != "" && trans.ErrorCode != types.TeOk.String() {
		style22.printL(w, "Error code", trans.ErrorCode)
		style22.printL(w, "Error message", trans.ErrorMsg)
	}

	displayTransferInfo(w, trans.TransferInfo)
}

// ######################## ADD ##########################

//nolint:lll,tagliatelle // struct tags can be long for command line args
type TransferAdd struct {
	File         string             `required:"yes" short:"f" long:"file" description:"The file to transfer" json:"file,omitempty"`
	Out          string             `short:"o" long:"out" description:"The destination of the file" json:"output,omitempty"`
	Way          string             `required:"yes" short:"w" long:"way" description:"The direction of the transfer" choice:"send" choice:"receive" json:"-"`
	IsSend       bool               `json:"isSend,omitempty"`
	Client       string             `short:"c" long:"client" description:"The client with which the transfer is performed" json:"client,omitempty"`
	Partner      string             `required:"yes" short:"p" long:"partner" description:"The partner to which the transfer is requested" json:"partner,omitempty"`
	Account      string             `required:"yes" short:"l" long:"login" description:"The login of the account used to connect on the partner" json:"account,omitempty"`
	Rule         string             `required:"yes" short:"r" long:"rule" description:"The rule to use for the transfer" json:"rule,omitempty"`
	Date         string             `short:"d" long:"date" description:"The starting date (in ISO 8601 format) of the transfer" json:"start,omitempty"`
	TransferInfo map[string]confVal `short:"i" long:"info" description:"Custom information about the transfer, in key:val format. Can be repeated." json:"transferInfo,omitempty"`

	Name string `short:"n" long:"name" description:"[DEPRECATED] The name of the file after the transfer" json:"destPath,omitempty"` // Deprecated: the source name is used instead
}

func (t *TransferAdd) Execute([]string) error { return execute(t) }
func (t *TransferAdd) execute(w io.Writer) error {
	t.IsSend = t.Way == directionSend

	if t.Name != "" {
		fmt.Fprintln(w, "[WARNING] The '-n' ('--name') option is deprecated. "+
			"For simplicity, in the future, files will have the same name at "+
			"the source and the destination")
	}

	addr.Path = rest.TransfersPath

	loc, addErr := add(w, t)
	if addErr != nil {
		return addErr
	}

	fmt.Fprintf(w, "The transfer of file %q was successfully added under the ID: %s\n",
		t.File, filepath.Base(loc.Path))

	return nil
}

// ######################## GET ##########################

type TransferGet struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *TransferGet) Execute([]string) error { return execute(t) }
func (t *TransferGet) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/transfers/%d", t.Args.ID)

	trans := &api.OutTransfer{}
	if err := get(trans); err != nil {
		return err
	}

	displayTransfer(w, trans)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags can be long for command line args
type TransferList struct {
	ListOptions
	SortBy string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start+" choice:"start-" choice:"id+" choice:"id-" choice:"rule+" choice:"rule-" default:"start+"`
	Rules  []string `short:"r" long:"rule" description:"Filter the transfers based on the name of the transfer rule used. Can be repeated multiple times to filter multiple rules."`
	//nolint:misspell //spelling mistake CANCELLED must be kept for backward compatibility
	Statuses []string `short:"t" long:"status" description:"Filter the transfers based on the transfer's status. Can be repeated multiple times to filter multiple statuses." choice:"PLANNED" choice:"RUNNING" choice:"INTERRUPTED" choice:"PAUSED" choice:"CANCELLED" choice:"DONE" choice:"ERROR"`
	Start    string   `short:"d" long:"date" description:"Filter the transfers which started after a given date. Date must be in ISO 8601 format."`
}

func (t *TransferList) listURL() error {
	addr.Path = rest.TransfersPath
	query := url.Values{}
	query.Set("limit", utils.FormatUint(t.Limit))
	query.Set("offset", utils.FormatUint(t.Offset))
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

func (t *TransferList) Execute([]string) error { return execute(t) }

//nolint:dupl //history & transfer commands should be kept separate for future-proofing
func (t *TransferList) execute(w io.Writer) error {
	if err := t.listURL(); err != nil {
		return err
	}

	body := map[string][]*api.OutTransfer{}
	if err := list(&body); err != nil {
		return err
	}

	if transfers := body["transfers"]; len(transfers) > 0 {
		style0.printf(w, "=== Transfers ===")

		for _, transfer := range transfers {
			displayTransfer(w, transfer)
		}
	} else {
		fmt.Fprintln(w, "No transfers found.")
	}

	return nil
}

// ######################## PAUSE ##########################

type TransferPause struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *TransferPause) Execute([]string) error { return execute(t) }
func (t *TransferPause) execute(w io.Writer) error {
	return putTransferRequest(w, t.Args.ID, "pause",
		"paused. It can be resumed using the 'resume' command")
}

// ######################## RESUME ##########################

type TransferResume struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *TransferResume) Execute([]string) error { return execute(t) }
func (t *TransferResume) execute(w io.Writer) error {
	return putTransferRequest(w, t.Args.ID, "resume", "resumed")
}

// ######################## CANCEL ##########################

type TransferCancel struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *TransferCancel) Execute([]string) error { return execute(t) }
func (t *TransferCancel) execute(w io.Writer) error {
	return putTransferRequest(w, t.Args.ID, "cancel", "canceled")
}

// ######################## RESTART ##########################

//nolint:lll // struct tags can be long for command line args
type TransferRetry struct {
	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
	Date string `short:"d" long:"date" description:"Set the date at which the transfer should restart. Date must be in RFC3339 format."`
}

func (t *TransferRetry) Execute([]string) error { return execute(t) }

//nolint:dupl //history & transfer commands should be kept separate for future-proofing
func (t *TransferRetry) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/transfers/%d/retry", t.Args.ID)

	query := url.Values{}

	if t.Date != "" {
		start, err := time.Parse(time.RFC3339Nano, t.Date)
		if err != nil {
			return fmt.Errorf("'%s' is not a start valid date (accepted format: '%s'): %w",
				t.Date, time.RFC3339Nano, err)
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
	defer resp.Body.Close() //nolint:errcheck,gosec // error is irrelevant

	switch resp.StatusCode {
	case http.StatusCreated:
		loc, err := resp.Location()
		if err != nil {
			return fmt.Errorf("cannot get the resource location: %w", err)
		}

		id := filepath.Base(loc.Path)
		fmt.Fprintf(w, "The transfer will be retried under the ID: %q\n", id)

		return nil
	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

//nolint:lll // struct tags can be long for command line args
type TransferCancelAll struct {
	Target string `required:"yes" short:"t" long:"target" description:"The status of the transfers to cancel" choice:"planned" choice:"running" choice:"paused" choice:"interrupted" choice:"error" choice:"all"`
}

func (t *TransferCancelAll) Execute([]string) error { return execute(t) }
func (t *TransferCancelAll) execute(w io.Writer) error {
	addr.Path = rest.TransfersPath
	query := url.Values{}
	query.Set("target", t.Target)
	addr.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck,gosec // error is irrelevant

	switch resp.StatusCode {
	case http.StatusAccepted:
		fmt.Fprintln(w, "The transfers were successfully canceled.")

		return nil
	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)

	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}
