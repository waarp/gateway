package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func DisplayUser(w io.Writer, user *api.OutUser) {
	f := NewFormatter(w)
	defer f.Render()

	displayUser(f, user)
}

func displayUser(f *Formatter, user *api.OutUser) {
	f.Title("User %q", user.Username)
	f.Indent()

	defer f.UnIndent()

	displayPermissions(f, &user.Perms)
}

// ######################## ADD ##########################

type UserAdd struct {
	Username string `required:"true" short:"u" long:"username" description:"The user's name"`
	Password string `required:"true" short:"p" long:"password" description:"The user's password"`
	Perms    string `required:"true" short:"r" long:"rights" description:"The user's rights in chmod symbolic format"`
}

func (u *UserAdd) Execute([]string) error { return u.execute(stdOutput) }
func (u *UserAdd) execute(w io.Writer) error {
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

	if _, err := add(w, newUser); err != nil {
		return err
	}

	fmt.Fprintf(w, "The user %q was successfully added.\n", u.Username)

	return nil
}

// ######################## GET ##########################

type UserGet struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The user's name"`
	} `positional-args:"yes"`
}

func (u *UserGet) Execute([]string) error { return u.execute(stdOutput) }
func (u *UserGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/users", u.Args.Username)

	user := &api.OutUser{}
	if err := get(user); err != nil {
		return err
	}

	DisplayUser(w, user)

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

func (u *UserUpdate) Execute([]string) error { return u.execute(stdOutput) }
func (u *UserUpdate) execute(w io.Writer) error {
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
	addr.Path = path.Join("/api/users", u.Args.Username)

	if err := update(w, user); err != nil {
		return err
	}

	username := u.Args.Username
	if user.Username != nil && *user.Username != "" {
		username = *user.Username
	}

	fmt.Fprintf(w, "The user %q was successfully updated.\n", username)

	return nil
}

// ######################## DELETE ##########################

type UserDelete struct {
	Args struct {
		Username string `required:"yes" positional-arg-name:"username" description:"The old username"`
	} `positional-args:"yes"`
}

func (u *UserDelete) Execute([]string) error { return u.execute(stdOutput) }
func (u *UserDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/users", u.Args.Username)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The user %q was successfully deleted.\n", u.Args.Username)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // tags can be long for flags
type UserList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"username+" choice:"username-" default:"username+"`
}

func (u *UserList) Execute([]string) error { return u.execute(stdOutput) }

//nolint:dupl //duplicate is for a completely different command
func (u *UserList) execute(w io.Writer) error {
	addr.Path = "/api/users"

	listURL(&u.ListOptions, u.SortBy)

	body := map[string][]*api.OutUser{}
	if err := list(&body); err != nil {
		return err
	}

	if users := body["users"]; len(users) > 0 {
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("Users:")

		for _, user := range users {
			displayUser(f, user)
		}
	} else {
		fmt.Fprintln(w, "No users found.")
	}

	return nil
}
