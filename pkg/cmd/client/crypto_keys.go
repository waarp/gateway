package wg

import (
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func displayCryptoKeys(w io.Writer, keys []*api.GetCryptoKeyRespObject) error {
	Style0.Printf(w, "=== Cryptographic keys ===")
	for _, key := range keys {
		if err := displayCryptoKey(w, key); err != nil {
			return err
		}
	}

	return nil
}

func displayCryptoKey(w io.Writer, key *api.GetCryptoKeyRespObject) error {
	switch key.Type {
	case model.CryptoKeyTypeAES, model.CryptoKeyTypeHMAC:
		Style1.Printf(w, "%q key %q", key.Type, key.Name)
	case model.CryptoKeyTypePGPPrivate:
		displayPGPKey(w, "private", key)
	case model.CryptoKeyTypePGPPublic:
		displayPGPKey(w, "public", key)
	default:
		Style1.Printf(w, "Cryptographic key %q of unknown type %q", key.Name, key.Type)
	}

	// The API returns the value of a public key only: a key that comes without
	// its value has one, which is a secret.
	Style22.Secret(w, "Value", key.Key == "", false)

	return nil
}

// displayPGPKey describes a PGP key by what the API says of it: the value of a
// private key never reaches the client.
func displayPGPKey(w io.Writer, kind string, key *api.GetCryptoKeyRespObject) {
	Style1.Printf(w, "PGP %s key %q", kind, key.Name)
	Style22.Printf(w, "Entity:")

	for _, identity := range key.Identities {
		Style333.PrintV(w, identity)
	}

	Style22.PrintL(w, "Fingerprint", key.Fingerprint)
}

//nolint:lll // command tags are long
type CryptoKeysAdd struct {
	Name string   `short:"n" long:"name" required:"yes" description:"The name of the cryptographic key" json:"name,omitempty"`
	Type string   `short:"t" long:"type" required:"yes" description:"The type of the cryptographic key." choice:"AES" choice:"HMAC" choice:"PGP-PUBLIC" choice:"PGP-PRIVATE" json:"type,omitempty"`
	Key  textFile `short:"k" long:"key" description:"The file containing the cryptographic key" json:"key,omitzero"`
}

func (p *CryptoKeysAdd) Execute([]string) error { return p.execute(stdOutput) }
func (p *CryptoKeysAdd) execute(w io.Writer) error {
	addr.Path = "/api/keys"

	if _, err := add(w, p); err != nil {
		return err
	}

	fmt.Fprintf(w, "The cryptographic key %q was successfully added.\n", p.Name)

	return nil
}

type CryptoKeysGet struct {
	OutputFormat

	Args struct {
		Key string `required:"yes" positional-arg-name:"key" description:"The name of the cryptographic key"`
	} `positional-args:"yes"`
}

func (p *CryptoKeysGet) Execute([]string) error { return p.execute(stdOutput) }
func (p *CryptoKeysGet) execute(w io.Writer) error {
	SetAddrPath("/api/keys/%s", p.Args.Key)

	key := &api.GetCryptoKeyRespObject{}
	if err := get(key); err != nil {
		return err
	}

	return outputObject(w, key, &p.OutputFormat, displayCryptoKey)
}

//nolint:lll //struct tags can be long
type CryptoKeysList struct {
	ListOptions

	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"type+" choice:"type-" default:"name+"`
}

func (p *CryptoKeysList) Execute([]string) error { return p.execute(stdOutput) }
func (p *CryptoKeysList) execute(w io.Writer) error {
	addr.Path = "/api/keys"

	listURL(&p.ListOptions, p.SortBy)

	body := map[string][]*api.GetCryptoKeyRespObject{}
	if err := list(&body); err != nil {
		return err
	}

	if keys := body["cryptoKeys"]; len(keys) > 0 {
		return outputObject(w, keys, &p.OutputFormat, displayCryptoKeys)
	}

	fmt.Fprintln(w, "No cryptographic keys found.")

	return nil
}

//nolint:lll //struct tags can be long
type CryptoKeysUpdate struct {
	Args struct {
		Key string `required:"yes" positional-arg-name:"key" description:"The name of the cryptographic key"`
	} `positional-args:"yes" json:"-"`

	Name *string   `short:"n" long:"name" description:"The name of the cryptographic key" json:"name,omitempty"`
	Type *string   `short:"t" long:"type" description:"The type of the cryptographic key." choice:"AES" choice:"HMAC" choice:"PGP-PUBLIC" choice:"PGP-PRIVATE"  json:"type,omitempty"`
	Key  *textFile `short:"k" long:"key" description:"The file containing the cryptographic key" json:"key,omitempty"`
}

func (p *CryptoKeysUpdate) Execute([]string) error { return p.execute(stdOutput) }
func (p *CryptoKeysUpdate) execute(w io.Writer) error {
	SetAddrPath("/api/keys/%s", p.Args.Key)

	if err := update(w, p); err != nil {
		return err
	}

	name := p.Args.Key
	if p.Name != nil && *p.Name != "" {
		name = *p.Name
	}

	fmt.Fprintf(w, "The cryptographic key %q was successfully updated.\n", name)

	return nil
}

type CryptoKeysDelete struct {
	Args struct {
		Key string `required:"yes" positional-arg-name:"key" description:"The name of the cryptographic key"`
	} `positional-args:"yes"`
}

func (p *CryptoKeysDelete) Execute([]string) error { return p.execute(stdOutput) }
func (p *CryptoKeysDelete) execute(w io.Writer) error {
	SetAddrPath("/api/keys/%s", p.Args.Key)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The cryptographic key %q was successfully deleted.\n", p.Args.Key)

	return nil
}
