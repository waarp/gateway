package wg

import (
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

//nolint:gochecknoglobals //a global var is required here
var LocalAccount string

type LocAccArg struct{}

func (*LocAccArg) UnmarshalFlag(value string) error {
	LocalAccount = value

	return nil
}

func displayAccount(w io.Writer, account *api.OutAccount) {
	style1.printf(w, "Account %q", account.Login)
	style22.printL(w, "Credentials", withDefault(join(account.Credentials), none))
	displayAuthorizedRules(w, account.AuthorizedRules)
}

// ######################## ADD ##########################

//nolint:lll //tags are long
type LocAccAdd struct {
	Login    string `required:"yes" short:"l" long:"login" description:"The account's login" json:"login,omitempty"`
	Password string `short:"p" long:"password" description:"The account's password" json:"password,omitempty"`
}

func (l *LocAccAdd) Execute([]string) error { return execute(l) }
func (l *LocAccAdd) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts", Server)

	if _, err := add(w, l); err != nil {
		return err
	}

	fmt.Fprintf(w, "The account %q was successfully added.\n", l.Login)

	return nil
}

// ######################## GET ##########################

type LocAccGet struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (l *LocAccGet) Execute([]string) error { return execute(l) }
func (l *LocAccGet) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s", Server, l.Args.Login)

	account := &api.OutAccount{}
	if err := get(account); err != nil {
		return err
	}

	displayAccount(w, account)

	return nil
}

// ######################## UPDATE ##########################

type LocAccUpdate struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"old-login" description:"The account's login"`
	} `positional-args:"yes" json:"-"`
	Login    string `short:"l" long:"login" description:"The account's login" json:"login,omitempty"`
	Password string `short:"p" long:"password" description:"The account's password" json:"password,omitempty"`
}

func (l *LocAccUpdate) Execute([]string) error { return execute(l) }
func (l *LocAccUpdate) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s", Server, l.Args.Login)

	if err := update(w, l); err != nil {
		return err
	}

	login := l.Args.Login
	if l.Login != "" {
		login = l.Login
	}

	fmt.Fprintf(w, "The account %q was successfully updated.\n", login)

	return nil
}

// ######################## DELETE ##########################

type LocAccDelete struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (l *LocAccDelete) Execute([]string) error { return execute(l) }
func (l *LocAccDelete) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s", Server, l.Args.Login)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The account %q was successfully deleted.\n", l.Args.Login)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type LocAccList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login+" choice:"login-" default:"login+"`
}

func (l *LocAccList) Execute([]string) error { return execute(l) }

//nolint:dupl //duplicate is for a different command, best keep separate
func (l *LocAccList) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts", Server)

	listURL(&l.ListOptions, l.SortBy)

	body := map[string][]*api.OutAccount{}
	if err := list(&body); err != nil {
		return err
	}

	if accounts := body["localAccounts"]; len(accounts) > 0 {
		style0.printf(w, "=== Accounts of server %q ===", Server)

		for _, account := range accounts {
			displayAccount(w, account)
		}
	} else {
		fmt.Fprintf(w, "Server %q has no accounts.\n", Server)
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

func (l *LocAccAuthorize) Execute([]string) error { return execute(l) }
func (l *LocAccAuthorize) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s/authorize/%s/%s", Server,
		l.Args.Login, l.Args.Rule, l.Args.Direction)

	return authorize(w, "local account", l.Args.Login, l.Args.Rule, l.Args.Direction)
}

// ######################## REVOKE ##########################

type LocAccRevoke struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (l *LocAccRevoke) Execute([]string) error { return execute(l) }
func (l *LocAccRevoke) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/servers/%s/accounts/%s/revoke/%s/%s", Server,
		l.Args.Login, l.Args.Rule, l.Args.Direction)

	return revoke(w, "local account", l.Args.Login, l.Args.Rule, l.Args.Direction)
}
