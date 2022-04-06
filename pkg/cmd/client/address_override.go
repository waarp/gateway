package wg

import (
	"fmt"
	"io"
)

type overrideAddressCommand struct {
	Set    overrideAddressSet    `command:"set" description:"Create or update an address indirection"`
	List   overrideAddressList   `command:"list" description:"List the address indirections"`
	Delete overrideAddressDelete `command:"delete" description:"Delete an address indirection"`
}

func displayAddressOverride(w io.Writer, target, redirect string) {
	fmt.Fprintln(w, "● Address", bold(target), "redirects to", bold(redirect))
}

type overrideAddressSet struct {
	Target    string `required:"true" short:"t" long:"target" description:"The target address to be replaced"`
	ReplaceBy string `required:"true" short:"r" long:"replace-by" description:"The real address to replace with"`
}

func (o *overrideAddressSet) Execute([]string) error {
	override := map[string]string{o.Target: o.ReplaceBy}
	addr.Path = "/api/override/addresses"

	if err := add(&override); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The indirection for address", bold(o.Target),
		"was successfully set.")

	return nil
}

type overrideAddressList struct{}

func (o *overrideAddressList) Execute([]string) error {
	override := map[string]string{}
	addr.Path = "/api/override/addresses"

	if err := list(&override); err != nil {
		return err
	}

	for target, redirect := range override {
		displayAddressOverride(getColorable(), target, redirect)
	}

	return nil
}

type overrideAddressDelete struct {
	Args struct {
		Target string `required:"yes" positional-arg-name:"target" description:"The target address"`
	} `positional-args:"yes"`
}

func (o *overrideAddressDelete) Execute([]string) error {
	addr.Path = "/api/override/addresses/" + o.Args.Target

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The indirection for address", o.Args.Target,
		"was successfully deleted.")

	return nil
}
