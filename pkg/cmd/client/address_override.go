package wg

import (
	"fmt"
	"io"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func displayAddressOverride(w io.Writer, target, redirect string) {
	style1.printf(w, "Address %q redirects to %q", target, redirect)
}

type OverrideAddressSet struct {
	Target    string `required:"true" short:"t" long:"target" description:"The target address to be replaced"`
	ReplaceBy string `required:"true" short:"r" long:"replace-by" description:"The real address to replace with"`
}

func (o *OverrideAddressSet) Execute([]string) error { return execute(o) }
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

func (o *OverrideAddressList) Execute([]string) error { return execute(o) }
func (o *OverrideAddressList) execute(w io.Writer) error {
	overrides := map[string]string{}
	addr.Path = "/api/override/addresses"

	if err := list(&overrides); err != nil {
		return err
	}

	if len(overrides) != 0 {
		style0.printf(w, "=== Address indirections ===")

		redirects := maps.Keys(overrides)
		slices.Sort(redirects)

		for _, redirect := range redirects {
			displayAddressOverride(w, redirect, overrides[redirect])
		}
	} else {
		fmt.Fprintln(w, "No overrides found.")
	}

	return nil
}

type OverrideAddressDelete struct {
	Args struct {
		Target string `required:"yes" positional-arg-name:"target" description:"The target address"`
	} `positional-args:"yes"`
}

func (o *OverrideAddressDelete) Execute([]string) error { return execute(o) }
func (o *OverrideAddressDelete) execute(w io.Writer) error {
	addr.Path = "/api/override/addresses/" + o.Args.Target

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The indirection for address %q was successfully deleted.\n", o.Args.Target)

	return nil
}
