package wg

import (
	"fmt"
	"io"
	"sort"
)

func displayAddressOverride(f *Formatter, target, redirect string) {
	f.Title("Address %q redirects to %q", target, redirect)
}

type OverrideAddressSet struct {
	Target    string `required:"true" short:"t" long:"target" description:"The target address to be replaced"`
	ReplaceBy string `required:"true" short:"r" long:"replace-by" description:"The real address to replace with"`
}

func (o *OverrideAddressSet) Execute([]string) error { return o.execute(stdOutput) }
func (o *OverrideAddressSet) execute(w io.Writer) error {
	override := map[string]string{o.Target: o.ReplaceBy}
	addr.Path = "/api/override/addresses"

	if _, err := add(w, &override); err != nil {
		return err
	}

	fmt.Fprintf(w, "The indirection for address %q was successfully set to %q.\n",
		o.Target, o.ReplaceBy)

	return nil
}

type OverrideAddressList struct{}

func (o *OverrideAddressList) Execute([]string) error { return o.execute(stdOutput) }
func (o *OverrideAddressList) execute(w io.Writer) error {
	overrides := map[string]string{}
	addr.Path = "/api/override/addresses"

	if err := list(&overrides); err != nil {
		return err
	}

	f := NewFormatter(w)
	defer f.Render()

	if len(overrides) != 0 {
		f.MainTitle("Address indirections:")
		f.Indent()

		type redirection struct{ target, real string }

		redirections := make([]redirection, 0, len(overrides))

		for target, redirectTo := range overrides {
			redirections = append(redirections, redirection{target: target, real: redirectTo})
		}

		sort.Slice(redirections, func(i, j int) bool {
			return redirections[i].target < redirections[j].target
		})

		for _, redirect := range redirections {
			displayAddressOverride(f, redirect.target, redirect.real)
		}
	} else {
		f.Empty("No overrides found.", nil)
	}

	return nil
}

type OverrideAddressDelete struct {
	Args struct {
		Target string `required:"yes" positional-arg-name:"target" description:"The target address"`
	} `positional-args:"yes"`
}

func (o *OverrideAddressDelete) Execute([]string) error { return o.execute(stdOutput) }
func (o *OverrideAddressDelete) execute(w io.Writer) error {
	addr.Path = "/api/override/addresses/" + o.Args.Target

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The indirection for address %q was successfully deleted.\n", o.Args.Target)

	return nil
}
