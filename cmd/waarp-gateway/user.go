package main

import (
	"fmt"
	"io"
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type userCommand struct {
	Get    userGetCommand    `command:"get" description:"Retrieve a user's information"`
	Add    userAddCommand    `command:"add" description:"Add a new user"`
	Delete userDeleteCommand `command:"delete" description:"Delete a user"`
	Update userUpdateCommand `command:"update" description:"Update an existing user"`
	List   userListCommand   `command:"list" description:"List the known users"`
}

func displayUser(w io.Writer, user rest.OutUser) {
	fmt.Fprintf(w, "\033[97;1m● User %s\033[0m\n", user.Username)
}

// ######################## ADD ##########################

type userAddCommand struct {
	Username string `required:"true" short:"u" long:"username" description:"The user's name"`
	Password string `required:"true" short:"p" long:"password" description:"The user's password"`
}

func (u *userAddCommand) Execute([]string) error {
	newUser := rest.InUser{
		Username: u.Username,
		Password: []byte(u.Password),
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.UsersPath

	loc, err := addCommand(newUser, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The user \033[33m'%s'\033[0m was successfully added. "+
		"It can be consulted at the address: \033[37m%s\033[0m\n", newUser.Username, loc)

	return nil
}

// ######################## GET ##########################

type userGetCommand struct{}

func (u *userGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing user ID")
	}

	res := rest.OutUser{}
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.UsersPath + "/" + args[0]

	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	displayUser(w, res)

	return nil
}

// ######################## UPDATE ##########################

type userUpdateCommand struct {
	Username string `short:"u" long:"username" description:"The user's name"`
	Password string `short:"p" long:"password" description:"The user's password"`
}

func (u *userUpdateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing user ID")
	}

	newUser := rest.InUser{
		Username: u.Username,
		Password: []byte(u.Password),
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.UsersPath + "/" + args[0]

	_, err = updateCommand(newUser, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The user n°\033[33m%s\033[0m was successfully updated\n", args[0])

	return nil
}

// ######################## DELETE ##########################

type userDeleteCommand struct{}

func (u *userDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing user ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.UsersPath + "/" + args[0]

	if err := deleteCommand(conn); err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The user n°\033[33m%s\033[0m was successfully deleted from "+
		"the database\n", args[0])

	return nil
}

// ######################## LIST ##########################

type userListCommand struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"username" default:"username"`
}

func (u *userListCommand) Execute([]string) error {
	conn, err := listURL(rest.UsersPath, &u.listOptions, u.SortBy)
	if err != nil {
		return err
	}

	res := map[string][]rest.OutUser{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	users := res["users"]
	if len(users) > 0 {
		fmt.Fprintf(w, "\033[33;1mUsers:\033[0m\n")
		for _, user := range users {
			displayUser(w, user)
		}
	} else {
		fmt.Fprintln(w, "\033[31;1mNo users found\033[0m")
	}

	return nil
}
