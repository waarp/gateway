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

type historyCommand struct {
	Get     historyGetCommand     `command:"get" description:"Consult a finished transfer"`
	List    historyListCommand    `command:"list" description:"List the finished transfers"`
	Restart historyRestartCommand `command:"restart" description:"Restart a failed transfer"`
}

func displayHistory(w io.Writer, hist rest.OutHistory) {
	fmt.Fprintf(w, "\033[97;1m● Transfer n°%v\033[0m (%s)\n", hist.ID, coloredStatus(hist.Status))
	fmt.Fprintf(w, "  \033[97m-IsServer     :\033[0m \033[36m%t\033[0m\n", hist.IsServer)
	fmt.Fprintf(w, "  \033[97m-Send         :\033[0m \033[36m%t\033[0m\n", hist.IsSend)
	fmt.Fprintf(w, "  \033[97m-Protocol     :\033[0m \033[33m%s\033[0m\n", hist.Protocol)
	fmt.Fprintf(w, "  \033[97m-Rule         :\033[0m \033[97m%v\033[0m\n", hist.Rule)
	fmt.Fprintf(w, "  \033[97m-Account      :\033[0m \033[97m%v\033[0m\n", hist.Account)
	fmt.Fprintf(w, "  \033[97m-Agent        :\033[0m \033[97m%v\033[0m\n", hist.Agent)
	fmt.Fprintf(w, "  \033[97m-SrcFile      :\033[0m \033[97m%s\033[0m\n", hist.SourceFilename)
	fmt.Fprintf(w, "  \033[97m-DestFile     :\033[0m \033[97m%s\033[0m\n", hist.DestFilename)
	fmt.Fprintf(w, "  \033[97m-Start date   :\033[0m \033[97m%s\033[0m\n",
		hist.Start.Format(time.RFC3339))
	fmt.Fprintf(w, "  \033[97m-End date     :\033[0m \033[97m%s\033[0m\n",
		hist.Stop.Format(time.RFC3339))
	if hist.ErrorCode != model.TeOk {
		fmt.Fprintf(w, "  \033[97m-Error code   :\033[0m \033[31m%v\033[0m\n",
			hist.ErrorCode)
	}
	if hist.ErrorMsg != "" {
		fmt.Fprintf(w, "  \033[97m-Error message:\033[0m \033[97m%s\033[0m\n",
			hist.ErrorMsg)
	}
	if hist.Step != "" {
		fmt.Fprintf(w, "  \033[97m-Failed step  :\033[0m \033[97;1m%s\033[0m\n", hist.Step)
	}
	if hist.Progress != 0 {
		fmt.Fprintf(w, "  \033[97m-Progress     :\033[0m \033[97m%v\033[0m\n", hist.Progress)
	}
	if hist.TaskNumber != 0 {
		fmt.Fprintf(w, "  \033[97m-Task number  :\033[0m \033[97m%v\033[0m\n", hist.TaskNumber)
	}
}

// ######################## GET ##########################

type historyGetCommand struct{}

func (h *historyGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing transfer history ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.HistoryPath + "/" + args[0]

	res := rest.OutHistory{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	displayHistory(getColorable(), res)

	return nil
}

// ######################## LIST ##########################

type historyListCommand struct {
	listOptions
	SortBy   string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start" choice:"id" choice:"source" choice:"destination" choice:"rule" default:"start"`
	Account  []string `long:"account" description:"Filter the transfers based on the transfer's account. Can be repeated multiple times to filter multiple sources."`
	Agent    []string `long:"agent" description:"Filter the transfers based on the transfer's agent. Can be repeated multiple times to filter multiple destinations."`
	Rules    []string `long:"rule" description:"Filter the transfers based on the transfer rule used. Can be repeated multiple times to filter multiple rules."`
	Statuses []string `long:"status" description:"Filter the transfers based on the transfer's status. Can be repeated multiple times to filter multiple statuses." choice:"DONE" choice:"ERROR"`
	Protocol []string `long:"protocol" description:"Filter the transfers based on the protocol used. Can be repeated multiple times to filter multiple protocols." choice:"sftp"`
	Start    string   `long:"start" description:"Filter the transfers which started after a given date. Date must be in RFC3339 format."`
	Stop     string   `long:"stop" description:"Filter the transfers which ended before a given date. Date must be in RFC3339 format."`
}

func (h *historyListCommand) listURL() (*url.URL, error) {
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return nil, err
	}

	conn.Path = admin.APIPath + rest.HistoryPath
	query := url.Values{}
	query.Set("limit", fmt.Sprint(h.Limit))
	query.Set("offset", fmt.Sprint(h.Offset))
	if h.DescOrder {
		query.Set("sort", h.SortBy+"-")
	} else {
		query.Set("sort", h.SortBy+"+")
	}
	for _, acc := range h.Account {
		query.Add("account", acc)
	}
	for _, agent := range h.Agent {
		query.Add("agent", agent)
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

func (h *historyListCommand) Execute([]string) error {
	conn, err := h.listURL()
	if err != nil {
		return err
	}

	res := map[string][]rest.OutHistory{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	history := res["history"]
	if len(history) > 0 {
		fmt.Fprintf(w, "\033[33mHistory:\033[0m\n")
		for _, hist := range history {
			displayHistory(w, hist)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo transfers found\033[0m")
	}

	return nil
}

// ######################## RESTART ##########################

type historyRestartCommand struct {
	Date string `short:"d" long:"date" description:"Set the date at which the transfer should restart. Date must be in RFC3339 format."`
}

func (h *historyRestartCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing transfer history ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.HistoryPath + "/" + args[0] + "/restart"

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
	fmt.Fprintf(w, "The transfer n°\033[33m%s\033[0m was successfully restarted. "+
		"It can be consulted at the address: \033[37m%s\033[0m\n", args[0], loc.String())

	return nil
}
