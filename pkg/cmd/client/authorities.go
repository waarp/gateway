package wg

import (
	"fmt"
	"io"
	"path"

	"github.com/gookit/color"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

func displayAuthority(w io.Writer, authority *api.OutAuthority) {
	Style1.Printf(w, "Authority %q", authority.Name)
	Style22.PrintL(w, "Type", authority.Type)
	Style22.PrintL(w, "Valid for hosts", withDefault(join(authority.ValidHosts), "<all>"))

	var err error

	switch authority.Type {
	case auth.AuthorityTLS:
		err = displayTLSInfo(w, Style22, authority.Name, authority.PublicIdentity)
	case sftp.AuthoritySSHCert:
		err = displaySSHKeyInfo(w, Style22, authority.Name, authority.PublicIdentity)
	}

	if err != nil {
		fmt.Fprintln(w, color.Red.Sprint(err))
	}
}

//nolint:lll,tagliatelle //flag tags are long
type AuthorityAdd struct {
	Name         string   `required:"yes" short:"n" long:"name" description:"The authority's name" json:"name,omitempty"`
	Type         string   `required:"yes" short:"t" long:"type" description:"The type of authority" choice:"tls_authority" choice:"ssh_cert_authority" json:"type,omitempty"`
	IdentityFile file     `required:"yes" short:"i" long:"identity-file" description:"The authority's public identity file" json:"publicIdentity,omitempty"`
	ValidHosts   []string `short:"h" long:"host" description:"The hosts on which the authority is valid. Can be repeated." json:"validHosts,omitempty"`
}

func (a *AuthorityAdd) Execute([]string) error { return a.execute(stdOutput) }
func (a *AuthorityAdd) execute(w io.Writer) error {
	addr.Path = "/api/authorities"

	if _, err := add(w, a); err != nil {
		return err
	}

	fmt.Fprintf(w, "The authority %q was successfully added.\n", a.Name)

	return nil
}

type AuthorityGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The authority's name"`
	} `positional-args:"yes"`
}

func (a *AuthorityGet) Execute([]string) error { return a.execute(stdOutput) }
func (a *AuthorityGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/authorities", a.Args.Name)

	authority := &api.OutAuthority{}
	if err := get(authority); err != nil {
		return err
	}

	displayAuthority(w, authority)

	return nil
}

//nolint:lll // struct tags can be long for command line args
type AuthorityList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+" `
}

func (a *AuthorityList) Execute([]string) error { return a.execute(stdOutput) }

//nolint:dupl //duplicate is for a completely different command, keep separate
func (a *AuthorityList) execute(w io.Writer) error {
	addr.Path = "/api/authorities"

	listURL(&a.ListOptions, a.SortBy)

	body := map[string][]*api.OutAuthority{}
	if err := list(&body); err != nil {
		return err
	}

	if authorities := body["authorities"]; len(authorities) > 0 {
		Style0.Printf(w, "=== Authentication authorities ===")

		for _, authority := range authorities {
			displayAuthority(w, authority)
		}
	} else {
		fmt.Fprintln(w, "No authorities found.")
	}

	return nil
}

//nolint:lll,tagliatelle //flag tags are long
type AuthorityUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The authority's name"`
	} `positional-args:"yes" json:"-"`

	Name         string   `short:"n" long:"name" description:"The new authority name" json:"name,omitempty"`
	Type         string   `short:"t" long:"type" description:"The type of authority" choice:"tls_authority" choice:"ssh_cert_authority" json:"type,omitempty"`
	IdentityFile file     `short:"i" long:"identity-file" description:"The authority's public identity file" json:"publicIdentity,omitempty"`
	ValidHosts   []string `short:"h" long:"host" description:"The hosts on which the authority is valid. Can be repeated. Will replace the existing list. Can be called with an empty host to delete all existing hosts." json:"validHosts,omitempty"`
}

func (a *AuthorityUpdate) Execute([]string) error { return a.execute(stdOutput) }
func (a *AuthorityUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/authorities", a.Args.Name)

	if err := update(w, a); err != nil {
		return err
	}

	name := a.Args.Name
	if a.Name != "" {
		name = a.Name
	}

	fmt.Fprintf(w, "The authority %q was successfully updated.\n", name)

	return nil
}

type AuthorityDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The authority's name"`
	} `positional-args:"yes"`
}

func (a *AuthorityDelete) Execute([]string) error { return a.execute(stdOutput) }
func (a *AuthorityDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/authorities", a.Args.Name)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The authority %q was successfully deleted.\n", a.Args.Name)

	return nil
}
