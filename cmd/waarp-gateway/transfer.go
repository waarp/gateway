package main

import (
	"fmt"
	"net/url"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type transferCommand struct {
	Add  transferAddCommand  `command:"add" description:"Add a new transfer to be executed"`
	Get  transferGetCommand  `command:"get" description:"Consult a planned transfer"`
	List transferListCommand `command:"list" description:"List the planned transfers"`
}

func displayTransfer(trans rest.OutTransfer) {
	w := getColorable()

	fmt.Fprintf(w, "\033[37;1;4mTransfer n°%v:\033[0m\n", trans.ID)
	fmt.Fprintf(w, "          \033[37mRule ID:\033[0m \033[33m%v\033[0m\n", trans.RuleID)
	fmt.Fprintf(w, "       \033[37mPartner ID:\033[0m \033[33m%v\033[0m\n", trans.AgentID)
	fmt.Fprintf(w, "       \033[37mAccount ID:\033[0m \033[33m%v\033[0m\n", trans.AccountID)
	fmt.Fprintf(w, "      \033[37mSource file:\033[0m \033[37m%s\033[0m\n", trans.SourcePath)
	fmt.Fprintf(w, " \033[37mDestination file:\033[0m \033[37m%s\033[0m\n", trans.DestPath)
	fmt.Fprintf(w, "       \033[37mStart time:\033[0m \033[33m%s\033[0m\n",
		trans.Start.Format(time.RFC3339))
	fmt.Fprintf(w, "           \033[37mStatus:\033[0m \033[37;1m%s\033[0m\n", trans.Status)
	if trans.ErrorCode != model.TeOk {
		fmt.Fprintf(w, "       \033[37mError code:\033[0m \033[33m%s\033[0m\n",
			trans.ErrorCode)
	}
	if trans.ErrorMsg != "" {
		fmt.Fprintf(w, "    \033[37mError message:\033[0m \033[33m%s\033[0m\n",
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

func (t *transferAddCommand) Execute(_ []string) error {
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

	displayTransfer(res)

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

func (t *transferListCommand) Execute(_ []string) error {
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
			displayTransfer(trans)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo transfers found\033[0m")
	}

	return nil
}
