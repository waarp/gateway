package wg

import (
	"fmt"
	"io"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

func DisplayAuthority(w io.Writer, authority *api.OutAuthority) {
	f := NewFormatter(w)
	defer f.Render()

	displayAuthority(f, authority)
}

func displayAuthority(f *Formatter, authority *api.OutAuthority) {
	f.Title("Authority %q", authority.Name)
	f.Indent()

	defer f.UnIndent()

	f.Value("Type", authority.Type)
	f.ValueWithDefault("Valid Hosts", strings.Join(authority.ValidHosts, ", "), "<all>")

	switch authority.Type {
	case auth.AuthorityTLS:
		displayTLSInfo(f, authority.Name, authority.PublicIdentity)
	case sftp.AuthoritySSHCert:
		displaySSHKeyInfo(f, authority.Name, authority.PublicIdentity)
	}
}

//nolint:lll,tagliatelle //flag tags are long
type authorityAdd struct {
	Name         string   `required:"yes" short:"n" long:"name" description:"The authority's name" json:"name,omitempty"`
	Type         string   `required:"yes" short:"t" long:"type" description:"The type of authority" choice:"tls_authority" choice:"ssh_cert_authority" json:"type,omitempty"`
	IdentityFile file     `required:"yes" short:"i" long:"identity-file" description:"The authority's public identity file" json:"publicIdentity,omitempty"`
	ValidHosts   []string `short:"h" long:"host" description:"The hosts on which the authority is valid. Can be repeated." json:"validHosts,omitempty"`
}

func (a *authorityAdd) Execute([]string) error { return a.execute(stdOutput) }
func (a *authorityAdd) execute(w io.Writer) error {
	addr.Path = "/api/authorities"

	if _, err := add(w, a); err != nil {
		return err
	}

	fmt.Fprintf(w, "The authority %q was successfully added.\n", a.Name)

	return nil
}

type authorityGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The authority's name"`
	} `positional-args:"yes"`
}

func (a *authorityGet) Execute([]string) error { return a.execute(stdOutput) }
func (a *authorityGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/authorities", a.Args.Name)

	authority := &api.OutAuthority{}
	if err := get(authority); err != nil {
		return err
	}

	DisplayAuthority(w, authority)

	return nil
}

//nolint:lll // struct tags can be long for command line args
type authorityList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+" `
}

func (a *authorityList) Execute([]string) error { return a.execute(stdOutput) }

//nolint:dupl //duplicate is for a completely different command, keep separate
func (a *authorityList) execute(w io.Writer) error {
	addr.Path = "/api/authorities"

	listURL(&a.ListOptions, a.SortBy)

	body := map[string][]*api.OutAuthority{}
	if err := list(&body); err != nil {
		return err
	}

	if authorities := body["authorities"]; len(authorities) > 0 {
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("Authentication authorities:")

		for _, authority := range authorities {
			displayAuthority(f, authority)
		}
	} else {
		fmt.Fprintln(w, "No authorities found.")
	}

	return nil
}

//nolint:lll,tagliatelle //flag tags are long
type authorityUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The authority's name"`
	} `positional-args:"yes" json:"-"`

	Name         string   `short:"n" long:"name" description:"The new authority name" json:"name,omitempty"`
	Type         string   `short:"t" long:"type" description:"The type of authority" choice:"tls_authority" choice:"ssh_cert_authority" json:"type,omitempty"`
	IdentityFile file     `short:"i" long:"identity-file" description:"The authority's public identity file" json:"publicIdentity,omitempty"`
	ValidHosts   []string `short:"h" long:"host" description:"The hosts on which the authority is valid. Can be repeated. Will replace the existing list. Can be called with an empty host to delete all existing hosts." json:"validHosts,omitempty"`
}

func (a *authorityUpdate) Execute([]string) error { return a.execute(stdOutput) }
func (a *authorityUpdate) execute(w io.Writer) error {
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

type authorityDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The authority's name"`
	} `positional-args:"yes"`
}

func (a *authorityDelete) Execute([]string) error { return a.execute(stdOutput) }
func (a *authorityDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/authorities", a.Args.Name)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The authority %q was successfully deleted.\n", a.Args.Name)

	return nil
}
