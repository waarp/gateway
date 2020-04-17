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
	path := admin.APIPath + rest.LocalAgentsPath + "/" + server + rest.LocalAccountsPath

	if err := add(path, account); err != nil {
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
	path := admin.APIPath + rest.LocalAgentsPath + "/" + server +
		rest.LocalAccountsPath + "/" + l.Args.Login

	account := &rest.OutAccount{}
	if err := get(path, account); err != nil {
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
	path := admin.APIPath + rest.LocalAgentsPath + "/" + server +
		rest.LocalAccountsPath + "/" + l.Args.Login

	if err := update(path, account); err != nil {
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
	path := admin.APIPath + rest.LocalAgentsPath + "/" + server +
		rest.LocalAccountsPath + "/" + l.Args.Login

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
	path := rest.LocalAgentsPath + "/" + server + rest.LocalAccountsPath
	addr, err := accountListURL(path, &l.listOptions, l.SortBy)
	if err != nil {
		return err
	}

	body := map[string][]rest.OutAccount{}
	if err := list(addr, &body); err != nil {
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
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule  string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (l *locAccAuthorize) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	path := admin.APIPath + rest.LocalAgentsPath + "/" + server +
		rest.LocalAccountsPath + "/" + l.Args.Login + "/authorize/" + l.Args.Rule

	return authorize(path, "local account", l.Args.Login, l.Args.Rule)
}

// ######################## REVOKE ##########################

type locAccRevoke struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule  string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (l *locAccRevoke) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	path := admin.APIPath + rest.LocalAgentsPath + "/" + server +
		rest.LocalAccountsPath + "/" + l.Args.Login + "/revoke/" + l.Args.Rule

	return revoke(path, "local account", l.Args.Login, l.Args.Rule)
}
