package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type userCommand struct {
	Get    userGet    `command:"get" description:"Retrieve a user's information"`
	Add    userAdd    `command:"add" description:"Add a new user"`
	Delete userDelete `command:"delete" description:"Delete a user"`
	Update userUpdate `command:"update" description:"Update an existing user"`
	List   userList   `command:"list" description:"List the known users"`
}

func displayUser(w io.Writer, user *rest.OutUser) {
	fmt.Fprintln(w, whiteBold("● User ")+whiteBoldUL(user.Username))
}

// ######################## ADD ##########################

type userAdd struct {
	Username string `required:"true" short:"u" long:"username" description:"The user's name"`
	Password string `required:"true" short:"p" long:"password" description:"The user's password"`
}

func (u *userAdd) Execute([]string) error {
	newUser := rest.InUser{
		Username: u.Username,
		Password: []byte(u.Password),
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.UsersPath

	resp, err := sendRequest(conn, newUser, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, whiteBold("The user '")+whiteBoldUL(newUser.Username)+
			whiteBold("' was successfully added."))
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## GET ##########################

type userGet struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The user's name"`
	} `positional-args:"yes"`
}

func (u *userGet) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.UsersPath + "/" + u.Args.Username

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		user := &rest.OutUser{}
		if err := unmarshalBody(resp.Body, user); err != nil {
			return err
		}
		displayUser(w, user)
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## UPDATE ##########################

type userUpdate struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The old username"`
	} `positional-args:"yes"`
	Username string `short:"u" long:"username" description:"The new username"`
	Password string `short:"p" long:"password" description:"The new password"`
}

func (u *userUpdate) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.UsersPath + "/" + u.Args.Username

	update := rest.InUser{
		Username: u.Username,
		Password: []byte(u.Password),
	}
	resp, err := sendRequest(conn, update, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, whiteBold("The user '")+whiteBoldUL(update.Username)+
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

// ######################## DELETE ##########################

type userDelete struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The old username"`
	} `positional-args:"yes"`
}

func (u *userDelete) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.UsersPath + "/" + u.Args.Username

	resp, err := sendRequest(conn, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Fprintln(w, whiteBold("The user '")+whiteBoldUL(u.Args.Username)+
			whiteBold("' was successfully deleted."))
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## LIST ##########################

type userList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"username+" choice:"username-" default:"username+"`
}

func (u *userList) Execute([]string) error {
	conn, err := listURL(rest.UsersPath, &u.listOptions, u.SortBy)
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
		body := map[string][]rest.OutUser{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}
		users := body["users"]
		if len(users) > 0 {
			fmt.Fprintln(w, yellowBold("Users:"))
			for _, u := range users {
				user := u
				displayUser(w, &user)
			}
		} else {
			fmt.Fprintln(w, yellow("No users found."))
		}
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}
