package wg

import (
	"fmt"
	"io"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

//nolint:gochecknoglobals //a global var is required here
var LocalAccount string

type LocAccArg struct{}

func (LocAccArg) UnmarshalFlag(value string) error {
	LocalAccount = value

	return nil
}

func displayAccount(w io.Writer, account *api.OutAccount) {
	send := strings.Join(account.AuthorizedRules.Sending, ", ")
	recv := strings.Join(account.AuthorizedRules.Reception, ", ")

	fmt.Fprintln(w, orange(bold("● Account", account.Login)))
	fmt.Fprintln(w, orange("    Authorized rules"))
	fmt.Fprintln(w, bold("    ├─  Sending:"), send)
	fmt.Fprintln(w, bold("    └─Reception:"), recv)
}

// ######################## ADD ##########################

type LocAccAdd struct {
	Login    string `required:"yes" short:"l" long:"login" description:"The account's login"`
	Password string `required:"yes" short:"p" long:"password" description:"The account's password"`
}

func (l *LocAccAdd) Execute([]string) error {
	account := &api.InAccount{
		Login:    &l.Login,
		Password: &l.Password,
	}
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts", Server)

	if err := add(account); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The account", bold(l.Login), "was successfully added.")

	return nil
}

// ######################## GET ##########################

type LocAccGet struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (l *LocAccGet) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s", Server, l.Args.Login)

	account := &api.OutAccount{}
	if err := get(account); err != nil {
		return err
	}

	displayAccount(getColorable(), account)

	return nil
}

// ######################## UPDATE ##########################

type LocAccUpdate struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
	Login    *string `short:"l" long:"name" description:"The account's login"`
	Password *string `short:"p" long:"password" description:"The account's password"`
}

//nolint:dupl // FIXME too hard to refactor?
func (l *LocAccUpdate) Execute([]string) error {
	account := &api.InAccount{
		Login:    l.Login,
		Password: l.Password,
	}

	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s", Server, l.Args.Login)

	if err := update(account); err != nil {
		return err
	}

	login := l.Args.Login
	if l.Login != nil && *l.Login != "" {
		login = *l.Login
	}

	fmt.Fprintln(getColorable(), "The account", bold(login), "was successfully updated.")

	return nil
}

// ######################## DELETE ##########################

type LocAccDelete struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (l *LocAccDelete) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s", Server, l.Args.Login)

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The account", bold(l.Args.Login), "was successfully deleted.")

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type LocAccList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login+" choice:"login-" default:"login+"`
}

//nolint:dupl // FIXME too hard to refactor?
func (l *LocAccList) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts", Server)

	listURL(&l.ListOptions, l.SortBy)

	body := map[string][]api.OutAccount{}
	if err := list(&body); err != nil {
		return err
	}

	accounts := body["localAccounts"]

	w := getColorable() //nolint:ifshort // decrease readability

	if len(accounts) > 0 {
		fmt.Fprintln(w, bold("Accounts of server '"+Server+"':"))

		for i := range accounts {
			displayAccount(w, &accounts[i])
		}
	} else {
		fmt.Fprintln(w, "Server", bold(Server), "has no accounts.")
	}

	return nil
}

// ######################## AUTHORIZE ##########################

type LocAccAuthorize struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (l *LocAccAuthorize) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s/authorize/%s/%s", Server,
		l.Args.Login, l.Args.Rule, l.Args.Direction)

	return authorize("local account", l.Args.Login, l.Args.Rule, l.Args.Direction)
}

// ######################## REVOKE ##########################

type LocAccRevoke struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (l *LocAccRevoke) Execute([]string) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s/revoke/%s/%s", Server,
		l.Args.Login, l.Args.Rule, l.Args.Direction)

	return revoke("local account", l.Args.Login, l.Args.Rule, l.Args.Direction)
}
