package wg

import (
	"fmt"
	"io"
	"path"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func displayCryptoKey(w io.Writer, key *api.GetCryptoKeyRespObject) error {
	switch key.Type {
	case model.CryptoKeyTypeAES, model.CryptoKeyTypeHMAC:
		return displayRawKey(w, key)
	case model.CryptoKeyTypePGPPrivate, model.CryptoKeyTypePGPPublic:
		return displayPGPKey(w, key)
	default:
		Style1.Printf(w, "Cryptographic key %q of unknown type %q", key.Name, key.Type)
		Style22.PrintL(w, "Base64 key", key.Key)

		return nil
	}
}

func displayPGPKey(w io.Writer, key *api.GetCryptoKeyRespObject) error {
	pgpKey, err := pgp.NewKeyFromArmored(key.Key)
	if err != nil {
		return fmt.Errorf("could not parse PGP key: %w", err)
	}

	kind := "public"
	if pgpKey.IsPrivate() {
		kind = "private"
	}

	Style1.Printf(w, "PGP %s key %q", kind, key.Name)
	Style22.Printf(w, "Entity:")

	for _, identity := range pgpKey.GetEntity().Identities {
		Style333.Printf(w, identity.Name)
	}

	Style22.PrintL(w, "Fingerprint", pgpKey.GetFingerprint())

	return nil
}

func displayRawKey(w io.Writer, key *api.GetCryptoKeyRespObject) error {
	Style1.Printf(w, "%q key %q", key.Type, key.Name)
	Style22.PrintL(w, "Base64 value", key.Key)

	return nil
}

//nolint:lll // command tags are long
type CryptoKeysAdd struct {
	Name string `short:"n" long:"name" required:"yes" description:"The name of the cryptographic key" json:"name,omitempty"`
	Type string `short:"t" long:"type" required:"yes" description:"The type of the cryptographic key." choice:"AES" choice:"HMAC" choice:"PGP-PUBLIC" choice:"PGP-PRIVATE" json:"type,omitempty"`
	Key  file   `short:"k" long:"key" description:"The file containing the cryptographic key" json:"key,omitempty"`
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
	Args struct {
		Key string `required:"yes" positional-arg-name:"key" description:"The name of the cryptographic key"`
	} `positional-args:"yes"`
}

func (p *CryptoKeysGet) Execute([]string) error { return p.execute(stdOutput) }
func (p *CryptoKeysGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/keys", p.Args.Key)

	key := &api.GetCryptoKeyRespObject{}
	if err := get(key); err != nil {
		return err
	}

	return displayCryptoKey(w, key)
}

//nolint:lll //struct tags can be long
type CryptographicKeysList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" choice:"type+" choice:"type-" default:"name+"`
}

func (p *CryptographicKeysList) Execute([]string) error { return p.execute(stdOutput) }
func (p *CryptographicKeysList) execute(w io.Writer) error {
	addr.Path = "/api/keys"

	listURL(&p.ListOptions, p.SortBy)

	body := map[string][]*api.GetCryptoKeyRespObject{}
	if err := list(&body); err != nil {
		return err
	}

	if keys := body["cryptoKeys"]; len(keys) > 0 {
		Style0.Printf(w, "=== Cryptographic keys ===")

		for _, key := range keys {
			if err := displayCryptoKey(w, key); err != nil {
				return err
			}
		}
	} else {
		fmt.Fprintln(w, "No cryptographic keys found.")
	}

	return nil
}

//nolint:lll //struct tags can be long
type CryptoKeysUpdate struct {
	Args struct {
		Key string `required:"yes" positional-arg-name:"key" description:"The name of the cryptographic key"`
	} `positional-args:"yes" json:"-"`
	Name string `short:"n" long:"name" description:"The name of the cryptographic key" json:"name,omitempty"`
	Type string `short:"t" long:"type" description:"The type of the cryptographic key." choice:"AES" choice:"HMAC" choice:"PGP-PUBLIC" choice:"PGP-PRIVATE"  json:"type,omitempty"`
	Key  file   `short:"k" long:"key" description:"The file containing the cryptographic key" json:"key,omitempty"`
}

func (p *CryptoKeysUpdate) Execute([]string) error { return p.execute(stdOutput) }
func (p *CryptoKeysUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/keys", p.Args.Key)

	if err := update(w, p); err != nil {
		return err
	}

	name := p.Args.Key
	if p.Name != "" {
		name = p.Name
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
	addr.Path = path.Join("/api/keys", p.Args.Key)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The cryptographic key %q was successfully deleted.\n", p.Args.Key)

	return nil
}
