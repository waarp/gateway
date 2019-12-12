package main

import (
	"fmt"
	"net/url"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type historyCommand struct {
	Get  historyGetCommand  `command:"get" description:"Consult a finished transfer"`
	List historyListCommand `command:"list" description:"List the finished transfers"`
}

func displayHistory(hist model.TransferHistory) {
	w := getColorable()

	fmt.Fprintf(w, "\033[37;1;4mTransfer %v=>\033[0m\n", hist.ID)
	fmt.Fprintf(w, "          \033[37mIsServer:\033[0m \033[37;1m%t\033[0m\n", hist.IsServer)
	fmt.Fprintf(w, "              \033[37mSend:\033[0m \033[37;1m%t\033[0m\n", hist.IsSend)
	fmt.Fprintf(w, "          \033[37mProtocol:\033[0m \033[37;1m%s\033[0m\n", hist.Protocol)
	fmt.Fprintf(w, "              \033[37mRule:\033[0m \033[37m%v\033[0m\n", hist.Rule)
	fmt.Fprintf(w, "           \033[37mAccount:\033[0m \033[37m%v\033[0m\n", hist.Account)
	fmt.Fprintf(w, "            \033[37mRemote:\033[0m \033[37m%v\033[0m\n", hist.Remote)
	fmt.Fprintf(w, "           \033[37mSrcFile:\033[0m \033[37m%s\033[0m\n", hist.SourceFilename)
	fmt.Fprintf(w, "          \033[37mDestFile:\033[0m \033[37m%s\033[0m\n", hist.DestFilename)
	fmt.Fprintf(w, "        \033[37mStart date:\033[0m \033[33m%s\033[0m\n",
		hist.Start.Format(time.RFC3339))
	fmt.Fprintf(w, "          \033[37mEnd date:\033[0m \033[33m%s\033[0m\n",
		hist.Stop.Format(time.RFC3339))
	fmt.Fprintf(w, "            \033[37mStatus:\033[0m \033[37;1m%s\033[0m\n", hist.Status)
	if hist.Error.Code != model.TeOk {
		fmt.Fprintf(w, "       \033[37mError code:\033[0m \033[33m%v\033[0m\n",
			hist.Error.Code)
	}
	if hist.Error.Details != "" {
		fmt.Fprintf(w, "    \033[37mError message:\033[0m \033[33m%s\033[0m\n",
			hist.Error.Details)
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
	conn.Path = admin.APIPath + admin.HistoryPath + "/" + args[0]

	res := model.TransferHistory{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	displayHistory(res)

	return nil
}

// ######################## LIST ##########################

type historyListCommand struct {
	listOptions
	SortBy   string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"start" choice:"id" choice:"source" choice:"destination" choice:"rule" default:"start"`
	Account  []string `long:"source" description:"Filter the transfers based on the transfer's source. Can be repeated multiple times to filter multiple sources."`
	Remote   []string `long:"destination" description:"Filter the transfers based on the transfer's destination. Can be repeated multiple times to filter multiple destinations."`
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

	conn.Path = admin.APIPath + admin.HistoryPath
	query := url.Values{}
	query.Set("limit", fmt.Sprint(h.Limit))
	query.Set("offset", fmt.Sprint(h.Offset))
	query.Set("sortby", h.SortBy)
	if h.DescOrder {
		query.Set("order", "desc")
	}
	for _, acc := range h.Account {
		query.Add("account", acc)
	}
	for _, remote := range h.Remote {
		query.Add("remote", remote)
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

func (h *historyListCommand) Execute(_ []string) error {
	conn, err := h.listURL()
	if err != nil {
		return err
	}

	res := map[string][]model.TransferHistory{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	history := res["history"]
	if len(history) > 0 {
		fmt.Fprintf(w, "\033[33mHistory:\033[0m\n")
		for _, hist := range history {
			displayHistory(hist)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo transfers found\033[0m")
	}

	return nil
}
