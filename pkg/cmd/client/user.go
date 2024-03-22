package wg

import (
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func displayUser(w io.Writer, user *api.OutUser) {
	fmt.Fprintln(w, bold("● User %q", user.Username))
	fmt.Fprintln(w, orange("    Permissions:"))
	fmt.Fprintln(w, bold("    ├─Transfers:"), user.Perms.Transfers)
	fmt.Fprintln(w, bold("    ├─Servers:  "), user.Perms.Servers)
	fmt.Fprintln(w, bold("    ├─Partners: "), user.Perms.Partners)
	fmt.Fprintln(w, bold("    ├─Rules:    "), user.Perms.Rules)
	fmt.Fprintln(w, bold("    └─Users:    "), user.Perms.Users)
}

// ######################## ADD ##########################

type UserAdd struct {
	Username string `required:"true" short:"u" long:"username" description:"The user's name"`
	Password string `required:"true" short:"p" long:"password" description:"The user's password"`
	Perms    string `required:"true" short:"r" long:"rights" description:"The user's rights in chmod symbolic format"`
}

func (u *UserAdd) Execute([]string) error {
	perms, err := parsePerms(u.Perms)
	if err != nil {
		return err
	}

	newUser := &api.InUser{
		Username: &u.Username,
		Password: &u.Password,
		Perms:    perms,
	}
	addr.Path = "/api/users"

	if err := add(newUser); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The user", bold(u.Username), "was successfully added.")

	return nil
}

// ######################## GET ##########################

type UserGet struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The user's name"`
	} `positional-args:"yes"`
}

func (u *UserGet) Execute([]string) error {
	addr.Path = "/api/users/" + u.Args.Username

	user := &api.OutUser{}
	if err := get(user); err != nil {
		return err
	}

	displayUser(getColorable(), user)

	return nil
}

// ######################## UPDATE ##########################

type UserUpdate struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The old username"`
	} `positional-args:"yes"`
	Username *string `short:"u" long:"username" description:"The new username"`
	Password *string `short:"p" long:"password" description:"The new password"`
	Perms    *string `short:"r" long:"rights" description:"The user's rights in chmod symbolic format"`
}

func (u *UserUpdate) Execute([]string) error {
	var perms *api.Perms

	if u.Perms != nil {
		var err error
		if perms, err = parsePerms(*u.Perms); err != nil {
			return err
		}
	}

	user := &api.InUser{
		Username: u.Username,
		Password: u.Password,
		Perms:    perms,
	}
	addr.Path = "/api/users/" + u.Args.Username

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

type UserDelete struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The old username"`
	} `positional-args:"yes"`
}

func (u *UserDelete) Execute([]string) error {
	addr.Path = "/api/users/" + u.Args.Username

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The user", bold(u.Args.Username), "was successfully deleted.")

	return nil
}

// ######################## LIST ##########################

//nolint:lll // tags can be long for flags
type UserList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"username+" choice:"username-" default:"username+"`
}

func (u *UserList) Execute([]string) error {
	addr.Path = "/api/users"

	listURL(&u.ListOptions, u.SortBy)

	body := map[string][]api.OutUser{}
	if err := list(&body); err != nil {
		return err
	}

	w := getColorable() //nolint:ifshort // false positive

	if users := body["users"]; len(users) > 0 {
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
