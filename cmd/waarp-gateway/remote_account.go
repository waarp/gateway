package main

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

//nolint:lll // struct tags for command line arguments can be long
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
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", partner, r.Args.Login)

	account := &api.OutAccount{}
	if err := get(account); err != nil {
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
	account := api.InAccount{
		Login:    &r.Login,
		Password: &r.Password,
	}
	partner := commandLine.Account.Remote.Args.Partner
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts", partner)

	if err := add(account); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The account", bold(r.Login), "was successfully added.")

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
	uri := fmt.Sprintf("/api/partners/%s/accounts/%s", partner, r.Args.Login)

	if err := remove(uri); err != nil {
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
	Login    *string `short:"l" long:"name" description:"The account's login"`
	Password *string `short:"p" long:"password" description:"The account's password"`
}

//nolint:dupl // FIXME too hard to refactor?
func (r *remAccUpdate) Execute([]string) error {
	account := &api.InAccount{
		Login:    r.Login,
		Password: r.Password,
	}

	partner := commandLine.Account.Remote.Args.Partner
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", partner, r.Args.Login)

	if err := update(account); err != nil {
		return err
	}

	login := r.Args.Login

	if account.Login != nil && *account.Login != "" {
		login = *account.Login
	}

	fmt.Fprintln(getColorable(), "The account", bold(login), "was successfully updated.")

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type remAccList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login+" choice:"login-" default:"login+"`
}

//nolint:dupl // FIXME too hard to refactor?
func (r *remAccList) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts", partner)

	listURL(&r.listOptions, r.SortBy)

	body := map[string][]api.OutAccount{}
	if err := list(&body); err != nil {
		return err
	}

	accounts := body["remoteAccounts"]

	w := getColorable() //nolint:ifshort // decrease readability

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
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s/authorize/%s/%s", partner,
		r.Args.Login, r.Args.Rule, r.Args.Direction)

	return authorize("remote account", r.Args.Login, r.Args.Rule, r.Args.Direction)
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
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s/revoke/%s/%s", partner,
		r.Args.Login, r.Args.Rule, r.Args.Direction)

	return revoke("remote account", r.Args.Login, r.Args.Rule, r.Args.Direction)
}
