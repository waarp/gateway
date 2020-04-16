package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
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

	fmt.Fprintln(w, bold("● Account", account.Login))
	fmt.Fprintln(w, orange("   Authorized rules"))
	fmt.Fprintln(w, orange("   ├─  Sending:"), send)
	fmt.Fprintln(w, orange("   └─Reception:"), recv)
}

// ######################## ADD ##########################

type locAccAdd struct {
	Login    string `required:"yes" short:"l" long:"login" description:"The account's login"`
	Password string `required:"yes" short:"p" long:"password" description:"The account's password"`
}

func (l *locAccAdd) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + server + rest.LocalAccountsPath

	newAccount := rest.InAccount{
		Login:    l.Login,
		Password: []byte(l.Password),
	}
	resp, err := sendRequest(conn, newAccount, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, "The account", bold(newAccount.Login), "was successfully added.")
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## GET ##########################

type locAccGet struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (l *locAccGet) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + server +
		rest.LocalAccountsPath + "/" + l.Args.Login
	log.Println(conn.String())

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		account := &rest.OutAccount{}
		if err := unmarshalBody(resp.Body, account); err != nil {
			return err
		}
		displayAccount(w, account)
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
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
	server := commandLine.Account.Local.Args.Server
	update := rest.InAccount{
		Login:    l.Login,
		Password: []byte(l.Password),
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + server +
		rest.LocalAccountsPath + "/" + l.Args.Login

	resp, err := sendRequest(conn, update, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, "The account", bold(update.Login), "was successfully updated.")
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %v - %s", resp.StatusCode,
			getResponseMessage(resp).Error())
	}
}

// ######################## DELETE ##########################

type locAccDelete struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (l *locAccDelete) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.LocalAgentsPath + "/" + server +
		rest.LocalAccountsPath + "/" + l.Args.Login

	resp, err := sendRequest(conn, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Fprintln(w, "The account", bold(l.Args.Login), "was successfully deleted.")
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## LIST ##########################

type locAccList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login+" choice:"login-" default:"login+"`
}

func (l *locAccList) Execute([]string) error {
	server := commandLine.Account.Local.Args.Server
	path := rest.LocalAgentsPath + "/" + server + rest.LocalAccountsPath
	conn, err := accountListURL(path, &l.listOptions, l.SortBy)
	if err != nil {
		return err
	}

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		body := map[string][]rest.OutAccount{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}
		accounts := body["localAccounts"]
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
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
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
