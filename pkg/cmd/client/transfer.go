package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const apiTransfersPath = "/api/transfers"

func transferRole(isServer bool) string {
	return utils.If[string](isServer, roleServer, roleClient)
}

func displayTransfers(w io.Writer, transfers []*api.OutTransfer) error {
	Style0.Printf(w, "=== Transfers ===")
	for _, transfer := range transfers {
		if err := displayTransfer(w, transfer); err != nil {
			return err
		}
	}

	return nil
}

func displayTransfer(w io.Writer, trans *api.OutTransfer) error {
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

	return nil
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
	NumberOfAttempts     int8               `long:"nb-of-attempts" description:"The number of times the transfer will be automatically retried if it failed" json:"nbOfAttempts,omitempty"`
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
	OutputFormat

	Args struct {
		ID uint64 `required:"yes" positional-arg-name:"id" description:"The transfer's ID"`
	} `positional-args:"yes"`
}

func (t *TransferGet) Execute([]string) error { return execute(t) }
func (t *TransferGet) execute(w io.Writer) error {
	SetAddrPath("/api/transfers/%s", utils.FormatUint(t.Args.ID))

	trans := &api.OutTransfer{}
	if err := get(trans); err != nil {
		return err
	}

	return outputObject(w, trans, &t.OutputFormat, displayTransfer)
}

// ######################## LIST ##########################

//nolint:lll // struct tags can be long for command line args
type TransferList struct {
	ListOptions

	SortBy string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start+" choice:"start-" choice:"id+" choice:"id-" choice:"rule+" choice:"rule-" choice:"requested+" choice:"requested-" choice:"requester+" choice:"requester-" choice:"status+" choice:"status-" default:"start-"`
	Rules  []string `short:"r" long:"rule" description:"Filter the transfers based on the name of the transfer rule used. Can be repeated multiple times to filter multiple rules."`
	//nolint:misspell //spelling mistake CANCELLED must be kept for backward compatibility
	Statuses []string `short:"t" long:"status" description:"Filter the transfers based on the transfer's status. Can be repeated multiple times to filter multiple statuses." choice:"PLANNED" choice:"RUNNING" choice:"INTERRUPTED" choice:"PAUSED" choice:"CANCELLED" choice:"DONE" choice:"ERROR" choice:"AVAILABLE"`
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
		return outputObject(w, transfers, &t.OutputFormat, displayTransfers)
	}

	fmt.Fprintln(w, "No transfers found.")

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
	SetAddrPath("/api/transfers/%s/retry", utils.FormatUint(t.Args.ID))

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

// ######################## SETS ##########################

// transferSetOptions name the set of transfers a gesture is applied to at
// once (#793): the filters of the list (the rules, the statuses, the start
// date, the follow ID), minus the transfers left out. Without any, the set is
// every transfer. The status has no short option: "-t" stays the target of
// "cancel-all".
//
//nolint:lll // struct tags are long
type transferSetOptions struct {
	Rules    []string `short:"r" long:"rule" description:"Only the transfers using the given rule. Can be repeated to name several rules."`
	Statuses []string `long:"status" description:"Only the transfers in the given status. Can be repeated to name several statuses." choice:"PLANNED" choice:"RUNNING" choice:"PAUSED" choice:"INTERRUPTED" choice:"ERROR" choice:"CANCELLED" choice:"DONE"`
	Start    string   `long:"start" description:"Only the transfers started at or after the given date. Date must be in RFC3339 format."`
	FollowID string   `short:"f" long:"follow-id" description:"Only the transfers with the given follow ID."`
	Except   []uint64 `long:"except" description:"Leave the transfer with the given ID out of the set. Can be repeated."`
}

// given says whether any option names the set.
func (o *transferSetOptions) given() bool {
	return len(o.Rules) > 0 || len(o.Statuses) > 0 || o.Start != "" || o.FollowID != "" || len(o.Except) > 0
}

// values are the parameters of the request the options make.
func (o *transferSetOptions) values() (url.Values, error) {
	query := url.Values{}

	for _, rule := range o.Rules {
		query.Add("rule", rule)
	}

	for _, status := range o.Statuses {
		query.Add("status", status)
	}

	if o.Start != "" {
		if _, err := time.Parse(time.RFC3339Nano, o.Start); err != nil {
			return nil, fmt.Errorf("'%s' is not a valid date (accepted format: '%s'): %w",
				o.Start, time.RFC3339Nano, errInvalidDate)
		}

		query.Set("start", o.Start)
	}

	if o.FollowID != "" {
		query.Set("followID", o.FollowID)
	}

	for _, id := range o.Except {
		query.Add("except", utils.FormatUint(id))
	}

	return query, nil
}

// setRequest sends a gesture on a set and prints its counted answer.
func setRequest(w io.Writer, method string, query url.Values, done string) error {
	addr.RawQuery = query.Encode()

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, method)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck,gosec // error is irrelevant

	switch resp.StatusCode {
	case http.StatusOK:
		var answer api.OutTransferSet
		if decodeErr := unmarshalBody(resp.Body, &answer); decodeErr != nil {
			return decodeErr
		}

		displayTransferSet(w, &answer, done)

		return nil
	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}

// displayTransferSet prints what a gesture did on a set: how many transfers,
// how many were left aside and why, the new transfers of a retry.
func displayTransferSet(w io.Writer, answer *api.OutTransferSet, done string) {
	fmt.Fprintf(w, "%s %s, %d refused.\n", countTransfers(answer.Done), done, answer.Refused)

	for _, refusal := range answer.Refusals {
		fmt.Fprintf(w, "  - transfer %d: %s\n", refusal.ID, refusalText(refusal.Reason))
	}

	if len(answer.Created) > 0 {
		created := make([]string, len(answer.Created))
		for i, id := range answer.Created {
			created[i] = utils.FormatInt(id)
		}

		fmt.Fprintf(w, "New transfers: %s\n", strings.Join(created, ", "))
	}
}

// countTransfers counts the transfers in words.
func countTransfers(n int) string {
	if n == 1 {
		return "1 transfer"
	}

	return fmt.Sprintf("%d transfers", n)
}

// refusalText says in clear why a transfer of the set was left aside.
func refusalText(reason string) string {
	switch reason {
	case api.RefusalServer:
		return "a server transfer resumes or is retried on the initiative of the partner"
	case api.RefusalSync:
		return "the child of a synchronous transfer resumes through its parent"
	case api.RefusalState:
		return "the state of the transfer does not allow it"
	case api.RefusalMissing:
		return "the transfer no longer exists"
	case api.RefusalGone:
		return "the rule, the partner, the account or the client of the transfer no longer exists"
	case api.RefusalElsewhere:
		return "the transfer runs on another node of the cluster"
	case api.RefusalBusy:
		return "the running transfer did not stop in time"
	default:
		return reason
	}
}

// ######################## RESUME ALL ##########################

type TransferResumeAll struct {
	transferSetOptions
}

func (t *TransferResumeAll) Execute([]string) error { return execute(t) }
func (t *TransferResumeAll) execute(w io.Writer) error {
	addr.Path = apiTransfersPath + "/resume-all"

	query, err := t.values()
	if err != nil {
		return err
	}

	return setRequest(w, http.MethodPut, query, "resumed")
}

// ######################## RETRY ALL ##########################

//nolint:lll // struct tags are long
type TransferRetryAll struct {
	transferSetOptions

	Date string `short:"d" long:"date" description:"Set the date at which the new transfers start. Date must be in RFC3339 format."`
}

func (t *TransferRetryAll) Execute([]string) error { return execute(t) }
func (t *TransferRetryAll) execute(w io.Writer) error {
	addr.Path = apiTransfersPath + "/retry-all"

	query, err := t.values()
	if err != nil {
		return err
	}

	if t.Date != "" {
		start, parseErr := time.Parse(time.RFC3339Nano, t.Date)
		if parseErr != nil {
			return fmt.Errorf("'%s' is not a valid date (accepted format: '%s'): %w",
				t.Date, time.RFC3339Nano, errInvalidDate)
		}

		query.Set("date", start.Format(time.RFC3339Nano))
	}

	return setRequest(w, http.MethodPut, query, "retried")
}

// ######################## CANCEL ALL ##########################

//nolint:lll // struct tags are long
type TransferCancelAll struct {
	transferSetOptions

	Target string `short:"t" long:"target" description:"The status of the transfers to cancel. Required when no other option names the set." choice:"planned" choice:"running" choice:"paused" choice:"interrupted" choice:"error" choice:"all"`
}

func (t *TransferCancelAll) Execute([]string) error { return execute(t) }
func (t *TransferCancelAll) execute(w io.Writer) error {
	addr.Path = apiTransfersPath

	query, err := t.values()
	if err != nil {
		return err
	}

	if t.Target != "" {
		query.Set("target", t.Target)
	}

	// The set named by its options is cancelled and counted (#793); a target
	// alone cancels the transfers of that status as before.
	if t.given() {
		return setRequest(w, http.MethodDelete, query, "canceled")
	}

	if t.Target == "" {
		return errMissingTarget
	}

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
