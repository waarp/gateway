package wg

import (
	"fmt"
	"io"

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

func (r *RemAccGet) Execute([]string) error { return execute(r) }
func (r *RemAccGet) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, r.Args.Login)

	account := &api.OutAccount{}
	if err := get(account); err != nil {
		return err
	}

	displayAccount(w, account)

	return nil
}

// ######################## ADD ##########################

//nolint:lll //tags are long
type RemAccAdd struct {
	Login    string `required:"true" short:"l" long:"login" description:"The account's login" json:"login,omitempty"`
	Password string `short:"p" long:"password" description:"The account's password" json:"password,omitempty"`
}

func (r *RemAccAdd) Execute([]string) error { return execute(r) }
func (r *RemAccAdd) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts", Partner)

	if _, err := add(w, r); err != nil {
		return err
	}

	fmt.Fprintf(w, "The account %q was successfully added.\n", r.Login)

	return nil
}

// ######################## DELETE ##########################

type RemAccDelete struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (r *RemAccDelete) Execute([]string) error { return execute(r) }
func (r *RemAccDelete) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, r.Args.Login)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The account %q was successfully deleted.\n", r.Args.Login)

	return nil
}

// ######################## UPDATE ##########################

type RemAccUpdate struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"old-login" description:"The account's login"`
	} `positional-args:"yes" json:"-"`
	Login    string `short:"l" long:"login" description:"The account's login" json:"login,omitempty"`
	Password string `short:"p" long:"password" description:"The account's password" json:"password,omitempty"`
}

func (r *RemAccUpdate) Execute([]string) error { return execute(r) }
func (r *RemAccUpdate) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, r.Args.Login)

	if err := update(w, r); err != nil {
		return err
	}

	login := r.Args.Login
	if r.Login != "" {
		login = r.Login
	}

	fmt.Fprintf(w, "The account %q was successfully updated.\n", login)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type RemAccList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login+" choice:"login-" default:"login+"`
}

func (r *RemAccList) Execute([]string) error { return execute(r) }

//nolint:dupl //duplicate is for a different command, best keep separate
func (r *RemAccList) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts", Partner)

	listURL(&r.ListOptions, r.SortBy)

	body := map[string][]*api.OutAccount{}
	if err := list(&body); err != nil {
		return err
	}

	if accounts := body["remoteAccounts"]; len(accounts) > 0 {
		style0.printf(w, "=== Accounts of partner %q ===", Partner)

		for _, account := range accounts {
			displayAccount(w, account)
		}
	} else {
		fmt.Fprintf(w, "Partner %q has no accounts.\n", Partner)
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

func (r *RemAccAuthorize) Execute([]string) error { return execute(r) }
func (r *RemAccAuthorize) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s/authorize/%s/%s", Partner,
		r.Args.Login, r.Args.Rule, r.Args.Direction)

	return authorize(w, "remote account", r.Args.Login, r.Args.Rule, r.Args.Direction)
}

// ######################## REVOKE ##########################

type RemAccRevoke struct {
	Args struct {
		Login     string `required:"yes" positional-arg-name:"login" description:"The account's login"`
		Rule      string `required:"yes" positional-arg-name:"rule" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"`
	} `positional-args:"yes"`
}

func (r *RemAccRevoke) Execute([]string) error { return execute(r) }
func (r *RemAccRevoke) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s/revoke/%s/%s", Partner,
		r.Args.Login, r.Args.Rule, r.Args.Direction)

	return revoke(w, "remote account", r.Args.Login, r.Args.Rule, r.Args.Direction)
}
