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

func (r *RemAccGet) Execute([]string) error { return r.execute(stdOutput) }
func (r *RemAccGet) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, r.Args.Login)

	account := &api.OutAccount{}
	if err := get(account); err != nil {
		return err
	}

	DisplayAccount(w, account)

	return nil
}

// ######################## ADD ##########################

type RemAccAdd struct {
	Login    string `required:"true" short:"l" long:"login" description:"The account's login"`
	Password string `required:"true" short:"p" long:"password" description:"The account's password"`
}

func (r *RemAccAdd) Execute([]string) error { return r.execute(stdOutput) }
func (r *RemAccAdd) execute(w io.Writer) error {
	account := api.InAccount{
		Login:    &r.Login,
		Password: &r.Password,
	}
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts", Partner)

	if _, err := add(w, account); err != nil {
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

func (r *RemAccDelete) Execute([]string) error { return r.execute(stdOutput) }
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
	} `positional-args:"yes"`
	Login    *string `short:"l" long:"login" description:"The account's login"`
	Password *string `short:"p" long:"password" description:"The account's password"`
}

func (r *RemAccUpdate) Execute([]string) error { return r.execute(stdOutput) }

//nolint:dupl //duplicate is for a different command, better keep separate
func (r *RemAccUpdate) execute(w io.Writer) error {
	account := &api.InAccount{
		Login:    r.Login,
		Password: r.Password,
	}

	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, r.Args.Login)

	if err := update(w, account); err != nil {
		return err
	}

	login := r.Args.Login
	if account.Login != nil && *account.Login != "" {
		login = *account.Login
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

func (r *RemAccList) Execute([]string) error { return r.execute(stdOutput) }

//nolint:dupl //duplicate is for a different command, better keep separate
func (r *RemAccList) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts", Partner)

	listURL(&r.ListOptions, r.SortBy)

	body := map[string][]*api.OutAccount{}
	if err := list(&body); err != nil {
		return err
	}

	if accounts := body["remoteAccounts"]; len(accounts) > 0 {
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("Accounts of partner %q:", Partner)

		for _, account := range accounts {
			displayAccount(f, account)
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

func (r *RemAccAuthorize) Execute([]string) error { return r.execute(stdOutput) }
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

func (r *RemAccRevoke) Execute([]string) error { return r.execute(stdOutput) }
func (r *RemAccRevoke) execute(w io.Writer) error {
	addr.Path = fmt.Sprintf("/api/partners/%s/accounts/%s/revoke/%s/%s", Partner,
		r.Args.Login, r.Args.Rule, r.Args.Direction)

	return revoke(w, "remote account", r.Args.Login, r.Args.Rule, r.Args.Direction)
}
