package main

import (
	"fmt"
	"io"
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type accountCommand struct {
	Get    accountGetCommand    `command:"get" description:"Retrieve a remote account's information"`
	Add    accountAddCommand    `command:"add" description:"Add a new remote account"`
	Delete accountDeleteCommand `command:"delete" description:"Delete a remote account"`
	Update accountUpdateCommand `command:"update" description:"Update an existing remote account"`
	List   accountListCommand   `command:"list" description:"List the known remote accounts"`
}

func displayRemoteAccount(w io.Writer, account rest.OutAccount) {
	fmt.Fprintf(w, "\033[97;1m● %s\033[0m (ID %v)\n", account.Login, account.ID)
	fmt.Fprintf(w, "  \033[97m-Partner ID:\033[0m \033[33m%v\033[0m\n", account.AgentID)
}

// ######################## GET ##########################

type accountGetCommand struct{}

func (a *accountGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing account ID")
	}

	res := rest.OutAccount{}
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAccountsPath + "/" + args[0]

	if err := getCommand(&res, conn); err != nil {
		return err
	}

	displayRemoteAccount(getColorable(), res)

	return nil
}

// ######################## ADD ##########################

type accountAddCommand struct {
	PartnerID uint64 `required:"true" long:"partner_id" description:"The ID of the remote agent the account is attached to"`
	Login     string `required:"true" short:"l" long:"login" description:"The account's login"`
	Password  string `required:"true" short:"p" long:"password" description:"The account's password"`
}

func (a *accountAddCommand) Execute([]string) error {
	newAccount := rest.InAccount{
		Login:    a.Login,
		Password: []byte(a.Password),
		AgentID:  a.PartnerID,
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAccountsPath

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

type accountDeleteCommand struct{}

func (a *accountDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing account ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAccountsPath + "/" + args[0]

	if err := deleteCommand(conn); err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The account n°\033[33m%s\033[0m was successfully deleted from "+
		"the database\n", args[0])

	return nil
}

// ######################## UPDATE ##########################

type accountUpdateCommand struct {
	PartnerID uint64 `long:"partner_id" description:"The ID of the remote agent the account is attached to"`
	Login     string `short:"l" long:"name" description:"The account's login"`
	Password  string `short:"p" long:"protocol" description:"The account's password"`
}

func (a *accountUpdateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing account ID")
	}

	newAccount := rest.InAccount{
		Login:    a.Login,
		Password: []byte(a.Password),
		AgentID:  a.PartnerID,
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAccountsPath + "/" + args[0]

	_, err = updateCommand(newAccount, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The account n°\033[33m%s\033[0m was successfully updated\n", args[0])

	return nil
}

// ######################## LIST ##########################

type accountListCommand struct {
	listOptions
	SortBy        string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login" choice:"remote_agent_id" default:"login"`
	RemoteAgentID []uint64 `long:"partner_id" description:"Filter accounts based on the ID of the remote agent they are attached to. Can be repeated multiple times to filter multiple agents."`
}

func (s *accountListCommand) Execute([]string) error {
	conn, err := accountListURL(rest.RemoteAccountsPath, &s.listOptions, s.SortBy,
		s.RemoteAgentID)
	if err != nil {
		return err
	}

	res := map[string][]rest.OutAccount{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	accounts := res["remoteAccounts"]
	if len(accounts) > 0 {
		fmt.Fprintf(w, "\033[33;1mRemote accounts:\033[0m\n")
		for _, account := range accounts {
			displayRemoteAccount(w, account)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo remote accounts found\033[0m")
	}

	return nil
}
