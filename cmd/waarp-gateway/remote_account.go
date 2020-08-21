package main

import (
	"fmt"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type remoteAccountCommand struct {
	Args struct {
		Partner string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
	} `positional-args:"yes"`
	Get       remAccGet       `command:"get" description:"Retrieve a remote account's information"`
	Add       remAccAdd       `command:"add" description:"Add a new remote account"`
	Delete    remAccDelete    `command:"delete" description:"Delete a remote account"`
	Update    remAccUpdate    `command:"update" description:"Update an existing remote account"`
	List      remAccList      `command:"list" description:"List the known remote accounts"`
	Authorize remAccAuthorize `command:"authorize" description:"Give an account permission to use a rule"`
	Revoke    remAccRevoke    `command:"revoke" description:"Revoke an account's permission to use a rule"`
	Cert      struct {
		Args struct {
			Account string `required:"yes" positional-arg-name:"account" description:"The account's name"`
		} `positional-args:"yes"`
		certificateCommand
	} `command:"cert" description:"Manage an account's certificates"`
}

// ######################## GET ##########################

type remAccGet struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (r *remAccGet) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	path := admin.APIPath + rest.PartnersPath + "/" + partner +
		rest.AccountsPath + "/" + r.Args.Login

	account := &rest.OutAccount{}
	if err := get(path, account); err != nil {
		return err
	}
	displayAccount(getColorable(), account)
	return nil
}

// ######################## ADD ##########################

type remAccAdd struct {
	Login    string `required:"true" short:"l" long:"login" description:"The account's login"`
	Password string `required:"true" short:"p" long:"password" description:"The account's password"`
}

func (r *remAccAdd) Execute([]string) error {
	account := rest.InAccount{
		Login:    r.Login,
		Password: []byte(r.Password),
	}
	partner := commandLine.Account.Remote.Args.Partner
	path := admin.APIPath + rest.PartnersPath + "/" + partner + rest.AccountsPath

	if err := add(path, account); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The account", bold(account.Login), "was successfully added.")
	return nil
}

// ######################## DELETE ##########################

type remAccDelete struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (r *remAccDelete) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	path := admin.APIPath + rest.PartnersPath + "/" + partner +
		rest.AccountsPath + "/" + r.Args.Login

	if err := remove(path); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The account", bold(r.Args.Login), "was successfully deleted.")
	return nil
}

// ######################## UPDATE ##########################

type remAccUpdate struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
	Login    string `short:"l" long:"name" description:"The account's login"`
	Password string `short:"p" long:"protocol" description:"The account's password"`
}

func (r *remAccUpdate) Execute([]string) error {
	account := rest.InAccount{
		Login:    r.Login,
		Password: []byte(r.Password),
	}
	partner := commandLine.Account.Remote.Args.Partner
	path := admin.APIPath + rest.PartnersPath + "/" + partner +
		rest.AccountsPath + "/" + r.Args.Login

	if err := update(path, account); err != nil {
		return err
	}
	login := r.Args.Login
	if account.Login != "" {
		login = account.Login
	}
	fmt.Fprintln(getColorable(), "The account", bold(login), "was successfully updated.")
	return nil
}

// ######################## LIST ##########################

type remAccList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login+" choice:"login-" default:"login+"`
}

func (r *remAccList) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	path := rest.PartnersPath + "/" + partner + rest.AccountsPath
	addr, err := accountListURL(path, &r.listOptions, r.SortBy)
	if err != nil {
		return err
	}

	body := map[string][]rest.OutAccount{}
	if err := list(addr, &body); err != nil {
		return err
	}

	accounts := body["remoteAccounts"]
	w := getColorable()
	if len(accounts) > 0 {
		fmt.Fprintln(w, bold("Accounts of partner '"+partner+"':"))
		for _, a := range accounts {
			account := a
			displayAccount(w, &account)
		}
	} else {
		fmt.Fprintln(w, "Partner", bold(partner), "has no accounts.")
	}
	return nil
}

// ######################## AUTHORIZE ##########################

type remAccAuthorize struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (r *remAccAuthorize) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	path := admin.APIPath + rest.PartnersPath + "/" + partner +
		rest.AccountsPath + "/" + r.Args.Login + "/authorize/" + r.Args.Rule +
		"/" + strings.ToLower(r.Args.Direction)

	return authorize(path, "remote account", r.Args.Login, r.Args.Rule, r.Args.Direction)
}

// ######################## REVOKE ##########################

type remAccRevoke struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (r *remAccRevoke) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	path := admin.APIPath + rest.PartnersPath + "/" + partner +
		rest.AccountsPath + "/" + r.Args.Login + "/revoke/" + r.Args.Rule +
		"/" + strings.ToLower(r.Args.Direction)

	return revoke(path, "remote account", r.Args.Login, r.Args.Rule, r.Args.Direction)
}
