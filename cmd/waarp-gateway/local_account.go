package main

import (
	"fmt"
	"io"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type localAccountCommand struct {
	Args struct {
		Server string `required:"yes" positional-arg-name:"server" description:"The server's name"`
	} `positional-args:"yes"`
	Get       locAccGet       `command:"get" description:"Retrieve a local account's information"`
	Add       locAccAdd       `command:"add" description:"Add a new local account"`
	Delete    locAccDelete    `command:"delete" description:"Delete a local account"`
	Update    locAccUpdate    `command:"update" description:"Update an existing account"`
	List      locAccList      `command:"list" description:"List the known local accounts"`
	Authorize locAccAuthorize `command:"authorize" description:"Give an account permission to use a rule"`
	Revoke    locAccRevoke    `command:"revoke" description:"Revoke an account's permission to use a rule"`
	Cert      struct {
		Args struct {
			Account string `required:"yes" positional-arg-name:"account" description:"The account's name"`
		} `positional-args:"yes"`
		certificateCommand
	} `command:"cert" description:"Manage an account's certificates"`
}

func displayAccount(w io.Writer, account *rest.OutAccount) {
	send := strings.Join(account.AuthorizedRules.Sending, ", ")
	recv := strings.Join(account.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, orange(bold("● Account", account.Login)))
	fmt.Fprintln(w, orange("    Authorized rules"))
	fmt.Fprintln(w, bold("    ├─  Sending:"), send)
	fmt.Fprintln(w, bold("    └─Reception:"), recv)
}

// ######################## ADD ##########################

type locAccAdd struct {
	Login    string `required:"yes" short:"l" long:"login" description:"The account's login"`
	Password string `required:"yes" short:"p" long:"password" description:"The account's password"`
}

func (l *locAccAdd) Execute([]string) error {
	account := &rest.InAccount{
		Login:    l.Login,
		Password: []byte(l.Password),
	}
	server := commandLine.Account.Local.Args.Server
	addr.Path = admin.APIPath + rest.ServersPath + "/" + server + rest.AccountsPath

	if err := add(account); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The account", bold(account.Login), "was successfully added.")
	return nil
}

// ######################## GET ##########################

type locAccGet struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (l *locAccGet) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	addr.Path = admin.APIPath + rest.ServersPath + "/" + server +
		rest.AccountsPath + "/" + l.Args.Login

	account := &rest.OutAccount{}
	if err := get(account); err != nil {
		return err
	}
	displayAccount(getColorable(), account)
	return nil
}

// ######################## UPDATE ##########################

type locAccUpdate struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
	Login    string `short:"l" long:"name" description:"The account's login"`
	Password string `short:"p" long:"password" description:"The account's password"`
}

func (l *locAccUpdate) Execute([]string) error {
	account := &rest.InAccount{
		Login:    l.Login,
		Password: []byte(l.Password),
	}
	server := commandLine.Account.Local.Args.Server
	addr.Path = admin.APIPath + rest.ServersPath + "/" + server +
		rest.AccountsPath + "/" + l.Args.Login

	if err := update(account); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The account", bold(account.Login), "was successfully updated.")
	return nil
}

// ######################## DELETE ##########################

type locAccDelete struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (l *locAccDelete) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	path := admin.APIPath + rest.ServersPath + "/" + server +
		rest.AccountsPath + "/" + l.Args.Login

	if err := remove(path); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The account", bold(l.Args.Login), "was successfully deleted.")
	return nil
}

// ######################## LIST ##########################

type locAccList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login+" choice:"login-" default:"login+"`
}

func (l *locAccList) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	addr.Path = rest.APIPath + rest.ServersPath + "/" + server + rest.AccountsPath
	listURL(&l.listOptions, l.SortBy)

	body := map[string][]rest.OutAccount{}
	if err := list(&body); err != nil {
		return err
	}

	accounts := body["localAccounts"]
	w := getColorable()
	if len(accounts) > 0 {
		fmt.Fprintln(w, bold("Accounts of server '"+server+"':"))
		for _, a := range accounts {
			account := a
			displayAccount(w, &account)
		}
	} else {
		fmt.Fprintln(w, "Server", bold(server), "has no accounts.")
	}
	return nil
}

// ######################## AUTHORIZE ##########################

type locAccAuthorize struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (l *locAccAuthorize) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	addr.Path = admin.APIPath + rest.ServersPath + "/" + server +
		rest.AccountsPath + "/" + l.Args.Login + "/authorize/" + l.Args.Rule +
		"/" + strings.ToLower(l.Args.Direction)

	return authorize("local account", l.Args.Login, l.Args.Rule, l.Args.Direction)
}

// ######################## REVOKE ##########################

type locAccRevoke struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (l *locAccRevoke) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	addr.Path = admin.APIPath + rest.ServersPath + "/" + server +
		rest.AccountsPath + "/" + l.Args.Login + "/revoke/" + l.Args.Rule +
		"/" + strings.ToLower(l.Args.Direction)

	return revoke("local account", l.Args.Login, l.Args.Rule, l.Args.Direction)
}
