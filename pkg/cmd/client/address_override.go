package wg

import (
	"fmt"
	"io"
)

func displayAddressOverride(w io.Writer, target, redirect string) {
	fmt.Fprintln(w, "● Address", bold(target), "redirects to", bold(redirect))
}

type OverrideAddressSet struct {
	Target    string `required:"true" short:"t" long:"target" description:"The target address to be replaced"`
	ReplaceBy string `required:"true" short:"r" long:"replace-by" description:"The real address to replace with"`
}

func (o *OverrideAddressSet) Execute([]string) error {
	override := map[string]string{o.Target: o.ReplaceBy}
	addr.Path = "/api/override/addresses"

	if err := add(&override); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The indirection for address", bold(o.Target),
		"was successfully set.")

	return nil
}

type OverrideAddressList struct{}

func (o *OverrideAddressList) Execute([]string) error {
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

type OverrideAddressDelete struct {
	Args struct {
		Target string `required:"yes" positional-arg-name:"target" description:"The target address"`
	} `positional-args:"yes"`
}

func (o *OverrideAddressDelete) Execute([]string) error {
	addr.Path = "/api/override/addresses/" + o.Args.Target

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The indirection for address", o.Args.Target,
		"was successfully deleted.")

	return nil
}
