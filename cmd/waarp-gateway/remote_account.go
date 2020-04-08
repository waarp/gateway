package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type remoteAccountCommand struct {
	Args struct {
		Partner string `required:"yes" positional-arg-name:"partner" description:"The partner's name"`
	} `positional-args:"yes"`
	Get    remAccGet    `command:"get" description:"Retrieve a remote account's information"`
	Add    remAccAdd    `command:"add" description:"Add a new remote account"`
	Delete remAccDelete `command:"delete" description:"Delete a remote account"`
	Update remAccUpdate `command:"update" description:"Update an existing remote account"`
	List   remAccList   `command:"list" description:"List the known remote accounts"`
}

// ######################## GET ##########################

type remAccGet struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (r *remAccGet) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + partner +
		rest.RemoteAccountsPath + "/" + r.Args.Login
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

// ######################## ADD ##########################

type remAccAdd struct {
	Login    string `required:"true" short:"l" long:"login" description:"The account's login"`
	Password string `required:"true" short:"p" long:"password" description:"The account's password"`
}

func (r *remAccAdd) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + partner + rest.RemoteAccountsPath

	newAccount := rest.InAccount{
		Login:    r.Login,
		Password: []byte(r.Password),
	}
	resp, err := sendRequest(conn, newAccount, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, whiteBold("The account '")+whiteBoldUL(newAccount.Login)+
			whiteBold("' was successfully added."))
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp).Error())
	}
}

// ######################## DELETE ##########################

type remAccDelete struct {
	Args struct {
		Login string `required:"yes" positional-arg-name:"login" description:"The account's login"`
	} `positional-args:"yes"`
}

func (r *remAccDelete) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + partner +
		rest.RemoteAccountsPath + "/" + r.Args.Login

	resp, err := sendRequest(conn, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Fprintln(w, whiteBold("The account '")+whiteBoldUL(r.Args.Login)+
			whiteBold("' was successfully deleted."))
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
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
	partner := commandLine.Account.Remote.Args.Partner
	update := rest.InAccount{
		Login:    r.Login,
		Password: []byte(r.Password),
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RemoteAgentsPath + "/" + partner +
		rest.RemoteAccountsPath + "/" + r.Args.Login

	resp, err := sendRequest(conn, update, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, whiteBold("The account '")+whiteBoldUL(update.Login)+
			whiteBold("' was successfully updated."))
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

// ######################## LIST ##########################

type remAccList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"login+" choice:"login-" default:"login+"`
}

func (r *remAccList) Execute([]string) error {
	partner := commandLine.Account.Remote.Args.Partner
	path := rest.RemoteAgentsPath + "/" + partner + rest.RemoteAccountsPath
	conn, err := accountListURL(path, &r.listOptions, r.SortBy)
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
		accounts := body["remoteAccounts"]
		if len(accounts) > 0 {
			fmt.Fprintln(w, yellowBold("Accounts of partner ")+yellowBoldUL(partner)+
				yellow(":"))
			for _, a := range accounts {
				account := a
				displayAccount(w, &account)
			}
		} else {
			fmt.Fprintln(w, yellow("No accounts found on partner ")+yellowBoldUL(
				partner)+yellowBoldUL("."))
		}
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp).Error())
	}
}
