package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func displayPGPKey(w io.Writer, key *api.GetPGPKeyRespObject) error {
	style1.printf(w, "PGP key %q", key.Name)

	if key.PrivateKey != "" {
		if err := displayPGPKeyInfo(w, style22, key.PrivateKey); err != nil {
			return err
		}
	}

	if key.PublicKey != "" {
		if err := displayPGPKeyInfo(w, style22, key.PublicKey); err != nil {
			return err
		}
	}

	return nil
}

type PGPKeysAdd struct {
	Name       string      `short:"n" long:"name" description:"The name of the PGP key" json:"name,omitempty"`
	PrivateKey fileOrValue `short:"p" long:"private-key" description:"The PGP private key" json:"privateKey,omitempty"`
	PublicKey  fileOrValue `short:"b" long:"public-key" description:"The PGP public key" json:"publicKey,omitempty"`
}

func (p *PGPKeysAdd) Execute([]string) error { return p.execute(stdOutput) }
func (p *PGPKeysAdd) execute(w io.Writer) error {
	addr.Path = "/api/pgp/keys"

	if _, err := add(w, p); err != nil {
		return err
	}

	fmt.Fprintf(w, "The PGP key %q was successfully added.\n", p.Name)

	return nil
}

type PGPKeysGet struct {
	Args struct {
		Key string `required:"yes" positional-arg-name:"key" description:"The name of the PGP key"`
	} `positional-args:"yes"`
}

func (p *PGPKeysGet) Execute([]string) error { return p.execute(stdOutput) }
func (p *PGPKeysGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/pgp/keys", p.Args.Key)

	key := &api.GetPGPKeyRespObject{}
	if err := get(key); err != nil {
		return err
	}

	return displayPGPKey(w, key)
}

//nolint:lll //struct tags can be long
type PGPKeysList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+"`
}

func (p *PGPKeysList) Execute([]string) error { return p.execute(stdOutput) }
func (p *PGPKeysList) execute(w io.Writer) error {
	addr.Path = "/api/pgp/keys"

	listURL(&p.ListOptions, p.SortBy)

	body := map[string][]*api.GetPGPKeyRespObject{}
	if err := list(&body); err != nil {
		return err
	}

	if keys := body["pgpKeys"]; len(keys) > 0 {
		style0.printf(w, "=== PGP keys ===")

		for _, key := range keys {
			if err := displayPGPKey(w, key); err != nil {
				return err
			}
		}
	} else {
		fmt.Fprintln(w, "No PGP keys found.")
	}

	return nil
}

type PGPKeysUpdate struct {
	Args struct {
		Key string `required:"yes" positional-arg-name:"key" description:"The name of the PGP key"`
	} `positional-args:"yes" json:"-"`
	Name       string      `short:"n" long:"name" description:"The name of the PGP key" json:"name,omitempty"`
	PrivateKey fileOrValue `short:"p" long:"private-key" description:"The PGP private key" json:"privateKey,omitempty"`
	PublicKey  fileOrValue `short:"b" long:"public-key" description:"The PGP public key" json:"publicKey,omitempty"`
}

func (p *PGPKeysUpdate) Execute([]string) error { return p.execute(stdOutput) }
func (p *PGPKeysUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/pgp/keys", p.Args.Key)

	if err := update(w, p); err != nil {
		return err
	}

	name := p.Args.Key
	if p.Name != "" {
		name = p.Name
	}

	fmt.Fprintf(w, "The PGP key %q was successfully updated.\n", name)

	return nil
}

type PGPKeysDelete struct {
	Args struct {
		Key string `required:"yes" positional-arg-name:"key" description:"The name of the PGP key"`
	} `positional-args:"yes"`
}

func (p *PGPKeysDelete) Execute([]string) error { return p.execute(stdOutput) }
func (p *PGPKeysDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/pgp/keys", p.Args.Key)

	if err := remove(w); err != nil {
		return err
	}

	return nil
}
