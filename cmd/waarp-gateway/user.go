package main

import (
	"fmt"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/models"
)

type userCommand struct {
	Get    userGet    `command:"get" description:"Retrieve a user's information"`
	Add    userAdd    `command:"add" description:"Add a new user"`
	Delete userDelete `command:"delete" description:"Delete a user"`
	Update userUpdate `command:"update" description:"Update an existing user"`
	List   userList   `command:"list" description:"List the known users"`
}

func displayUser(w io.Writer, user *models.OutUser) {
	fmt.Fprintln(w, bold("● User", user.Username))
}

// ######################## ADD ##########################

type userAdd struct {
	Username string `required:"true" short:"u" long:"username" description:"The user's name"`
	Password string `required:"true" short:"p" long:"password" description:"The user's password"`
}

func (u *userAdd) Execute([]string) error {
	newUser := &models.InUser{
		Username: &u.Username,
		Password: []byte(u.Password),
	}
	addr.Path = admin.APIPath + rest.UsersPath

	if err := add(newUser); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The user", bold(u.Username), "was successfully added.")
	return nil
}

// ######################## GET ##########################

type userGet struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The user's name"`
	} `positional-args:"yes"`
}

func (u *userGet) Execute([]string) error {
	addr.Path = admin.APIPath + rest.UsersPath + "/" + u.Args.Username

	user := &models.OutUser{}
	if err := get(user); err != nil {
		return err
	}
	displayUser(getColorable(), user)
	return nil
}

// ######################## UPDATE ##########################

type userUpdate struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The old username"`
	} `positional-args:"yes"`
	Username *string `short:"u" long:"username" description:"The new username"`
	Password *string `short:"p" long:"password" description:"The new password"`
}

func (u *userUpdate) Execute([]string) error {
	user := &models.InUser{
		Username: u.Username,
		Password: parseOptBytes(u.Password),
	}
	addr.Path = admin.APIPath + rest.UsersPath + "/" + u.Args.Username

	if err := update(user); err != nil {
		return err
	}
	username := u.Args.Username
	if user.Username != nil && *user.Username != "" {
		username = *user.Username
	}
	fmt.Fprintln(getColorable(), "The user", bold(username), "was successfully updated.")
	return nil
}

// ######################## DELETE ##########################

type userDelete struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The old username"`
	} `positional-args:"yes"`
}

func (u *userDelete) Execute([]string) error {
	path := admin.APIPath + rest.UsersPath + "/" + u.Args.Username

	if err := remove(path); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The user", bold(u.Args.Username), "was successfully deleted.")
	return nil
}

// ######################## LIST ##########################

type userList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"username+" choice:"username-" default:"username+"`
}

func (u *userList) Execute([]string) error {
	addr.Path = rest.APIPath + rest.UsersPath
	listURL(&u.listOptions, u.SortBy)

	body := map[string][]models.OutUser{}
	if err := list(&body); err != nil {
		return err
	}

	users := body["users"]
	w := getColorable()
	if len(users) > 0 {
		fmt.Fprintln(w, bold("Users:"))
		for _, u := range users {
			user := u
			displayUser(w, &user)
		}
	} else {
		fmt.Fprintln(w, "No users found.")
	}
	return nil
}
