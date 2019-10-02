package main

import (
	"fmt"
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type accessCommand struct {
	Get    accessGetCommand    `command:"get" description:"Retrieve a local account's information"`
	Add    accessAddCommand    `command:"add" description:"Add a new local account"`
	Delete accessDeleteCommand `command:"delete" description:"Delete a local account"`
	Update accessUpdateCommand `command:"update" description:"Update an existing account"`
	List   accessListCommand   `command:"list" description:"List the known local accounts"`
}

func displayLocalAccount(account model.LocalAccount) {
	w := getColorable()

	fmt.Fprintf(w, "\033[97;1mLocal account n°%v:\033[0m\n", account.ID)
	fmt.Fprintf(w, "├─\033[97mLogin:\033[0m \033[34;4m%s\033[0m\n", account.Login)
	fmt.Fprintf(w, "└─\033[97mServer ID:\033[0m \033[33;4m%v\033[0m\n", account.LocalAgentID)
}

// ######################## GET ##########################

type accessGetCommand struct{}

func (a *accessGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing account ID")
	}

	res := model.LocalAccount{}
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.LocalAccountsPath + "/" + args[0]

	if err := getCommand(&res, conn); err != nil {
		return err
	}

	displayLocalAccount(res)

	return nil
}

// ######################## ADD ##########################

type accessAddCommand struct {
	LocalAgentID uint64 `required:"true" long:"server_id" description:"The ID of the local agent the account is attached to"`
	Login        string `required:"true" short:"l" long:"login" description:"The account's login"`
	Password     string `required:"true" short:"p" long:"password" description:"The account's password"`
}

func (a *accessAddCommand) Execute(_ []string) error {
	newAccount := model.LocalAccount{
		Login:        a.Login,
		Password:     []byte(a.Password),
		LocalAgentID: a.LocalAgentID,
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.LocalAccountsPath

	loc, err := addCommand(newAccount, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The account \033[33m'%s'\033[0m was successfully added. "+
		"It can be consulted at the address: \033[37m%s\033[0m\n", newAccount.Login, loc)

	return nil
}

// ######################## DELETE ##########################

type accessDeleteCommand struct{}

func (a *accessDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing account ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.LocalAccountsPath + "/" + args[0]

	if err := deleteCommand(conn); err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The account n°\033[33m%s\033[0m was successfully deleted from "+
		"the database\n", args[0])

	return nil
}

// ######################## UPDATE ##########################

type accessUpdateCommand struct {
	LocalAgentID uint64 `long:"server_id" description:"The ID of the local agent the account is attached to"`
	Login        string `short:"l" long:"name" description:"The account's login"`
	Password     string `short:"p" long:"password" description:"The account's password"`
}

func (a *accessUpdateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing account ID")
	}

	newAccount := map[string]interface{}{}
	if a.Login != "" {
		newAccount["login"] = a.Login
	}
	if a.Password != "" {
		newAccount["password"] = []byte(a.Password)
	}
	if a.LocalAgentID != 0 {
		newAccount["localAgentID"] = a.LocalAgentID
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.LocalAccountsPath + "/" + args[0]

	_, err = updateCommand(newAccount, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The account n°\033[33m%s\033[0m was successfully updated", args[0])

	return nil
}

// ######################## LIST ##########################

type accessListCommand struct {
	listOptions
	SortBy       string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login" choice:"local_agent_id" default:"login"`
	LocalAgentID []uint64 `long:"server_id" description:"Filter the accounts based on the ID of the local agent they are attached to. Can be repeated multiple times to filter multiple agents."`
}

func (s *accessListCommand) Execute(_ []string) error {
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.LocalAccountsPath
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sortby", s.SortBy)
	if s.DescOrder {
		query.Set("order", "desc")
	}
	for _, agent := range s.LocalAgentID {
		query.Add("agent", fmt.Sprint(agent))
	}
	conn.RawQuery = query.Encode()

	res := map[string][]model.LocalAccount{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	accounts := res["localAccounts"]
	if len(accounts) > 0 {
		fmt.Fprintf(w, "\033[33mLocal accounts:\033[0m\n")
		for _, account := range accounts {
			displayLocalAccount(account)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo local accounts found\033[0m")
	}

	return nil
}
