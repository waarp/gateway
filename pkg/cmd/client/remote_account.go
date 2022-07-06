package wg

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

//nolint:gochecknoglobals //a global var is required here
var RemoteAccount string

type RemAccArg struct{}

func (*RemAccArg) UnmarshalFlag(value string) error {
	RemoteAccount = value

	return nil
}

// ######################## GET ##########################

type RemAccGet struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (r *RemAccGet) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, r.Args.Login)

	account := &api.OutAccount{}
	if err := get(account); err != nil {
		return err
	}

	displayAccount(getColorable(), account)

	return nil
}

// ######################## ADD ##########################

type RemAccAdd struct {
	Login    string `required:"true" short:"l" long:"login" description:"The account's login"`
	Password string `required:"true" short:"p" long:"password" description:"The account's password"`
}

func (r *RemAccAdd) Execute([]string) error {
	account := api.InAccount{
		Login:    &r.Login,
		Password: &r.Password,
	}
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts", Partner)

	if err := add(account); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The account", bold(r.Login), "was successfully added.")

	return nil
}

// ######################## DELETE ##########################

type RemAccDelete struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (r *RemAccDelete) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, r.Args.Login)

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The account", bold(r.Args.Login), "was successfully deleted.")

	return nil
}

// ######################## UPDATE ##########################

type RemAccUpdate struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
	Login    *string `short:"l" long:"name" description:"The account's login"`
	Password *string `short:"p" long:"password" description:"The account's password"`
}

//nolint:dupl // FIXME too hard to refactor?
func (r *RemAccUpdate) Execute([]string) error {
	account := &api.InAccount{
		Login:    r.Login,
		Password: r.Password,
	}

	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, r.Args.Login)

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
type RemAccList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login+" choice:"login-" default:"login+"`
}

//nolint:dupl // FIXME too hard to refactor?
func (r *RemAccList) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts", Partner)

	listURL(&r.ListOptions, r.SortBy)

	body := map[string][]api.OutAccount{}
	if err := list(&body); err != nil {
		return err
	}

	accounts := body["remoteAccounts"]

	w := getColorable() //nolint:ifshort // decrease readability

	if len(accounts) > 0 {
		fmt.Fprintln(w, bold("Accounts of partner '"+Partner+"':"))

		for _, a := range accounts {
			account := a
			displayAccount(w, &account)
		}
	} else {
		fmt.Fprintln(w, "Partner", bold(Partner), "has no accounts.")
	}

	return nil
}

// ######################## AUTHORIZE ##########################

type RemAccAuthorize struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (r *RemAccAuthorize) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s/authorize/%s/%s", Partner,
		r.Args.Login, r.Args.Rule, r.Args.Direction)

	return authorize("remote account", r.Args.Login, r.Args.Rule, r.Args.Direction)
}

// ######################## REVOKE ##########################

type RemAccRevoke struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (r *RemAccRevoke) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s/revoke/%s/%s", Partner,
		r.Args.Login, r.Args.Rule, r.Args.Direction)

	return revoke("remote account", r.Args.Login, r.Args.Rule, r.Args.Direction)
}
