package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const apiTransfersPath = "/api/transfers"

func transferRole(isServer bool) string {
	return utils.If[string](isServer, roleServer, roleClient)
}

func displayTransfer(w io.Writer, trans *api.OutTransfer) {
	fmt.Fprintf(w, "%s%s (%s as %s) [%s]\n", Style1.bulletPrefix,
		Style1.color.Sprintf("Transfer %d", trans.ID),
		direction(trans.IsSend), transferRole(trans.IsServer),
		coloredStatus(trans.Status))

	Style22.PrintL(w, "Remote ID", trans.RemoteID)
	Style22.PrintL(w, "Protocol", trans.Protocol)

	switch {
	case trans.IsServer && trans.IsSend: // <- Server
		Style22.PrintL(w, "File pulled", trans.SrcFilename)
	case trans.IsServer && !trans.IsSend: // -> Server
		Style22.PrintL(w, "File pushed", trans.DestFilename)
	case !trans.IsServer && trans.IsSend: // Client ->
		Style22.PrintL(w, "File to send", trans.SrcFilename)
		Style22.PrintL(w, "File deposited as", trans.DestFilename)
	case !trans.IsServer && !trans.IsSend: // Client <-
		Style22.PrintL(w, "File to receive", trans.SrcFilename)
		Style22.PrintL(w, "File retrieved as", trans.DestFilename)
	}

	Style22.PrintL(w, "Rule", trans.Rule)
	Style22.PrintL(w, "Requested by", trans.Requester)
	Style22.PrintL(w, "Requested to", trans.Requested)
	Style22.Option(w, "With client", trans.Client)
	Style22.PrintL(w, "Full local path", trans.LocalFilepath)
	Style22.PrintL(w, "Full remote path", trans.RemoteFilepath)
	Style22.PrintL(w, "File size", prettyBytes(trans.Filesize))
	Style22.PrintL(w, "Start date", trans.Start.Local().String())
	Style22.PrintL(w, "End date",
		ifElse(trans.Stop.Valid, trans.Stop.Value.Local().String(), notApplicable))
	Style22.PrintL(w, "Next attempt",
		ifElse(!trans.NextAttempt.IsZero(), trans.NextAttempt.Local().String(), notApplicable))

	if trans.NextRetryDelay != 0 {
		delay := (time.Duration(trans.NextRetryDelay) * time.Second).String()

		Style22.PrintL(w, "Remaining attempts", trans.RemainingAttempts)
		Style22.PrintL(w, "Next retry delay", delay)
		Style22.PrintL(w, "Retry increment factor", trans.RetryIncrementFactor)
	}

	Style22.PrintL(w, "Data transferred", prettyBytes(trans.Progress))

	if trans.Step != "" && trans.Step != types.StepNone.String() {
		Style22.PrintL(w, "Current step", trans.Step)
	}

	Style22.Option(w, "Current task", cardinal(trans.TaskNumber))

	if trans.ErrorCode != "" && trans.ErrorCode != types.TeOk.String() {
		Style22.PrintL(w, "Error code", trans.ErrorCode)
		Style22.PrintL(w, "Error message", trans.ErrorMsg)
	}

	displayTransferInfo(w, trans.TransferInfo)
}

// ######################## ADD ##########################

//nolint:lll,tagliatelle // struct tags can be long for command line args
type TransferAdd struct {
	File                 string             `required:"yes" short:"f" long:"file" description:"The file to transfer" json:"file,omitempty"`
	Out                  string             `short:"o" long:"out" description:"The destination of the file" json:"output,omitempty"`
	Way                  string             `required:"yes" short:"w" long:"way" description:"The direction of the transfer" choice:"send" choice:"receive" json:"-"`
	IsSend               bool               `json:"isSend"`
	Client               string             `short:"c" long:"client" description:"The client with which the transfer is performed" json:"client,omitempty"`
	Partner              string             `required:"yes" short:"p" long:"partner" description:"The partner to which the transfer is requested" json:"partner,omitempty"`
	Account              string             `required:"yes" short:"l" long:"login" description:"The login of the account used to connect on the partner" json:"account,omitempty"`
	Rule                 string             `required:"yes" short:"r" long:"rule" description:"The rule to use for the transfer" json:"rule,omitempty"`
	Date                 string             `short:"d" long:"date" description:"The starting date (in ISO 8601 format) of the transfer" json:"start,omitempty"`
	TransferInfo         map[string]confVal `short:"i" long:"info" description:"Custom information about the transfer, in key:val format. Can be repeated." json:"transferInfo,omitempty"`
	NumberOfTries        int8               `long:"nb-of-attempts" description:"The number of times the transfer will be automatically retried if it failed" json:"numberOfTries,omitempty"`
	FirstRetryDelay      time.Duration      `long:"retry-delay" description:"The amount of time between automatic retries. Accepted units: 's', 'm' & 'h'" json:"-"`
	FirstRetryDelaySec   int32              `json:"firstRetryDelay,omitempty"`
	RetryIncrementFactor float32            `long:"retry-increment-factor" description:"The factor by which the retry delay will increase after each retry" json:"retryIncrementFactor,omitempty"`

	Name string `short:"n" long:"name" description:"[DEPRECATED] The name of the file after the transfer" json:"destPath,omitempty"` // Deprecated: the source name is used instead
}

func (t *TransferAdd) Execute([]string) error { return execute(t) }
func (t *TransferAdd) execute(w io.Writer) error {
	t.IsSend = t.Way == directionSend

	if t.FirstRetryDelay != 0 {
		t.FirstRetryDelaySec = int32(t.FirstRetryDelay.Seconds())
	}

	if t.Name != "" {
		fmt.Fprintln(w, "[WARNING] The '-n' ('--name') option is deprecated. "+
			"For simplicity, in the future, files will have the same name at "+
			"the source and the destination")
	}

	addr.Path = apiTransfersPath

	loc, addErr := add(w, t)
	if addErr != nil {
		return addErr
	}

	fmt.Fprintf(w, "The transfer of file %q was successfully added under the ID: %s\n",
		t.File, path.Base(loc.Path))

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
	FollowID string   `short:"f" long:"follow-id" description:"Filter the transfers based on their follow ID."`
}

func (t *TransferList) listURL() error {
	addr.Path = apiTransfersPath
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

	if t.FollowID != "" {
		query.Set("followID", t.FollowID)
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
		Style0.Printf(w, "=== Transfers ===")

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
		`paused. It can be resumed using the "resume" command`)
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

	resp, reqErr := sendRequest(ctx, nil, http.MethodPut)
	if reqErr != nil {
		return reqErr
	}
	defer resp.Body.Close() //nolint:errcheck,gosec // error is irrelevant

	switch resp.StatusCode {
	case http.StatusCreated:
		loc, err := resp.Location()
		if err != nil {
			return fmt.Errorf("cannot get the resource location: %w", err)
		}

		id := path.Base(loc.Path)
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
	addr.Path = apiTransfersPath
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

//nolint:lll //struct tags are long
type TransferPreregister struct {
	File         string             `required:"yes" short:"f" long:"file" description:"The file to transfer" json:"file,omitempty"`
	Rule         string             `required:"yes" short:"r" long:"rule" description:"The rule to use for the transfer" json:"rule,omitempty"`
	Way          string             `required:"yes" short:"w" long:"way" description:"The direction of the transfer" choice:"send" choice:"receive" json:"-"`
	Server       string             `required:"yes" short:"s" long:"server" description:"The name of the local server" json:"server,omitempty"`
	Account      string             `required:"yes" short:"l" long:"login" description:"The login of the account used to by partner" json:"account,omitempty"`
	DueDate      string             `required:"yes" short:"d" long:"due-date" description:"The expiration date (in ISO 8601 format) of the transfer" json:"dueDate,omitempty"`
	TransferInfo map[string]confVal `short:"i" long:"info" description:"Custom information about the transfer, in key:val format. Can be repeated." json:"transferInfo,omitempty"`

	IsSend bool `json:"isSend"`
}

func (t *TransferPreregister) Execute([]string) error { return execute(t) }
func (t *TransferPreregister) execute(w io.Writer) error {
	t.IsSend = t.Way == directionSend
	addr.Path = apiTransfersPath

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, addErr := sendRequest(ctx, t, http.MethodPut)
	if addErr != nil {
		return addErr
	}

	defer resp.Body.Close() //nolint:errcheck,gosec // error is irrelevant

	switch resp.StatusCode {
	case http.StatusCreated:
	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}

	loc, locErr := resp.Location()
	if locErr != nil {
		return fmt.Errorf("cannot get the resource location: %w", locErr)
	}

	fmt.Fprintf(w, "The transfer of file %q was successfully preregistered under the ID: %s\n",
		t.File, path.Base(loc.Path))

	return nil
}
