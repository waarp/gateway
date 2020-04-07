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
	Add    transferAddCommand    `command:"add" description:"Add a new transfer to be executed"`
	Get    transferGetCommand    `command:"get" description:"Consult a transfer"`
	List   transferListCommand   `command:"list" description:"List the transfers"`
	Pause  transferPauseCommand  `command:"pause" description:"Pause a running transfer"`
	Resume transferResumeCommand `command:"resume" description:"Resume a paused transfer"`
	Cancel transferCancelCommand `command:"cancel" description:"Cancel a transfer"`
}

func coloredStatus(status model.TransferStatus) string {
	code := func() string {
		switch status {
		case model.StatusPlanned:
			return "36"
		case model.StatusRunning:
			return "34"
		case model.StatusPaused:
			return "33"
		case model.StatusInterrupted:
			return "33"
		case model.StatusCancelled:
			return "31"
		case model.StatusError:
			return "31"
		case model.StatusDone:
			return "32"
		default:
			return "97"
		}
	}()
	return fmt.Sprintf("\033[%sm%s\033[0m", code, status)
}

func displayTransfer(w io.Writer, trans rest.OutTransfer) {
	fmt.Fprintf(w, "\033[97;1m● Transfer n°%v\033[0m (%s)\n", trans.ID, coloredStatus(trans.Status))
	fmt.Fprintf(w, "  \033[97m-Rule ID         :\033[0m \033[97m%v\033[0m\n", trans.RuleID)
	fmt.Fprintf(w, "  \033[97m-Partner ID      :\033[0m \033[97m%v\033[0m\n", trans.AgentID)
	fmt.Fprintf(w, "  \033[97m-Account ID      :\033[0m \033[33m%v\033[0m\n", trans.AccountID)
	fmt.Fprintf(w, "  \033[97m-True filepath   :\033[0m \033[97m%s\033[0m\n", trans.TrueFilepath)
	fmt.Fprintf(w, "  \033[97m-Source file     :\033[0m \033[97m%s\033[0m\n", trans.SourcePath)
	fmt.Fprintf(w, "  \033[97m-Destination file:\033[0m \033[97m%s\033[0m\n", trans.DestPath)
	fmt.Fprintf(w, "  \033[97m-Start time      :\033[0m \033[33m%s\033[0m\n",
		trans.Start.Format(time.RFC3339))
	fmt.Fprintf(w, "  \033[97m-Step            :\033[0m \033[97m%s\033[0m\n", trans.Step)
	fmt.Fprintf(w, "  \033[97m-Progress        :\033[0m \033[97m%v\033[0m\n", trans.Progress)
	fmt.Fprintf(w, "  \033[97m-Task number     :\033[0m \033[97m%v\033[0m\n", trans.TaskNumber)
	if trans.ErrorCode != model.TeOk {
		fmt.Fprintf(w, "  \033[97m-Error code      :\033[0m \033[31m%s\033[0m\n",
			trans.ErrorCode)
	}
	if trans.ErrorMsg != "" {
		fmt.Fprintf(w, "  \033[97m-Error message   :\033[0m \033[97m%s\033[0m\n",
			trans.ErrorMsg)
	}
}

// ######################## ADD ##########################

type transferAddCommand struct {
	File      string `required:"true" short:"f" long:"file" description:"The file to transfer"`
	ServerID  uint64 `required:"true" short:"s" long:"server_id" description:"The remote server with which perform the transfer"`
	AccountID uint64 `required:"true" short:"c" long:"account_id" description:"The account used to connect on the server"`
	RuleID    uint64 `required:"true" short:"r" long:"rule" description:"The rule to use for the transfer"`
}

func (t *transferAddCommand) Execute([]string) error {
	newTransfer := rest.InTransfer{
		AgentID:    t.ServerID,
		AccountID:  t.AccountID,
		SourcePath: t.File,
		RuleID:     t.RuleID,
		DestPath:   t.File,
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath

	_, err = addCommand(newTransfer, conn)
	if err != nil {
		return err
	}

	return nil
}

// ######################## GET ##########################

type transferGetCommand struct{}

func (t *transferGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing transfer ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath + "/" + args[0]

	res := rest.OutTransfer{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	displayTransfer(w, res)

	return nil
}

// ######################## LIST ##########################

type transferListCommand struct {
	listOptions
	SortBy   string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start" choice:"id" choice:"agent_id" choice:"rule_id" default:"start"`
	Remotes  []uint64 `long:"server_id" description:"Filter the transfers based on the ID of the transfer partner. Can be repeated multiple times to filter multiple partners."`
	Accounts []uint64 `long:"account_id" description:"Filter the transfers based on the ID the account used. Can be repeated multiple times to filter multiple accounts."`
	Rules    []uint64 `long:"rule_id" description:"Filter the transfers based on the ID of the transfer rule used. Can be repeated multiple times to filter multiple rules."`
	Statuses []string `long:"status" description:"Filter the transfers based on the transfer's status. Can be repeated multiple times to filter multiple statuses." choice:"PLANNED" choice:"TRANSFER"`
	Start    string   `long:"start" on:"Filter the transfers which started after a given date. Date must be in RFC3339 format."`
}

func (t *transferListCommand) listURL() (*url.URL, error) {
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return nil, err
	}

	conn.Path = admin.APIPath + rest.TransfersPath
	query := url.Values{}
	query.Set("limit", fmt.Sprint(t.Limit))
	query.Set("offset", fmt.Sprint(t.Offset))
	if t.DescOrder {
		query.Set("sort", t.SortBy+"-")
	} else {
		query.Set("sort", t.SortBy+"+")
	}
	for _, rem := range t.Remotes {
		query.Add("agent", fmt.Sprint(rem))
	}
	for _, acc := range t.Accounts {
		query.Add("account", fmt.Sprint(acc))
	}
	for _, rul := range t.Rules {
		query.Add("rule", fmt.Sprint(rul))
	}
	for _, sta := range t.Statuses {
		query.Add("status", sta)
	}
	if t.Start != "" {
		date, err := time.Parse(time.RFC3339, t.Start)
		if err != nil {
			return nil, fmt.Errorf("'%s' is not a valid date (accepted format: '%s')",
				t.Start, time.RFC3339)
		}
		query.Set("start", date.Format(time.RFC3339))
	}
	conn.RawQuery = query.Encode()

	return conn, nil
}

func (t *transferListCommand) Execute([]string) error {
	conn, err := t.listURL()
	if err != nil {
		return err
	}

	res := map[string][]rest.OutTransfer{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	transfers := res["transfers"]
	if len(transfers) > 0 {
		fmt.Fprintf(w, "\033[33mTransfers:\033[0m\n")
		for _, trans := range transfers {
			displayTransfer(w, trans)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo transfers found\033[0m")
	}

	return nil
}

// ######################## PAUSE ##########################

type transferPauseCommand struct{}

func (t *transferPauseCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing transfer history ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath + "/" + args[0] + "/pause"

	req, err := http.NewRequest(http.MethodPut, conn.String(), nil)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, conn)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusCreated {
		return handleErrors(res, conn)
	}

	loc, err := res.Location()
	if err != nil {
		return err
	}
	loc.User = nil

	w := getColorable()
	fmt.Fprintf(w, "The transfer n°\033[33m%s\033[0m was successfully paused."+
		" It can be resumed using the 'resume' command.\n", args[0])

	return nil
}

// ######################## RESUME ##########################

type transferResumeCommand struct{}

func (t *transferResumeCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing transfer history ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath + "/" + args[0] + "/resume"

	req, err := http.NewRequest(http.MethodPut, conn.String(), nil)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, conn)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusCreated {
		return handleErrors(res, conn)
	}

	loc, err := res.Location()
	if err != nil {
		return err
	}
	loc.User = nil

	w := getColorable()
	fmt.Fprintf(w, "The transfer n°\033[33m%s\033[0m was successfully resumed.\n", args[0])

	return nil
}

// ######################## CANCEL ##########################

type transferCancelCommand struct{}

func (t *transferCancelCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing transfer history ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.TransfersPath + "/" + args[0] + "/cancel"

	req, err := http.NewRequest(http.MethodPut, conn.String(), nil)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, conn)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusCreated {
		return handleErrors(res, conn)
	}

	loc, err := res.Location()
	if err != nil {
		return err
	}
	loc.User = nil

	w := getColorable()
	fmt.Fprintf(w, "The transfer n°\033[33m%s\033[0m was successfully cancelled. "+
		"It can be consulted at the address: \033[97m%s\033[0m\n", args[0], loc.String())

	return nil
}
