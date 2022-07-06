package wg

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/jessevdk/go-flags"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func displayCertificate(w io.Writer, cert *api.OutCrypto) {
	fmt.Fprintln(w, orange(bold("● Certificate", cert.Name)))
	fmt.Fprintln(w, orange("    Private key:"), cert.PrivateKey)
	fmt.Fprintln(w, orange("    Public key: "), cert.PublicKey)
	fmt.Fprintln(w, orange("    Content:    "), cert.Certificate)
}

func getCertPath() string {
	switch {
	case LocalAccount != "":
		return fmt.Sprintf("/api/servers/%s/accounts/%s", Server, LocalAccount)
	case RemoteAccount != "":
		return fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, RemoteAccount)
	case Server != "":
		return fmt.Sprintf("/api/servers/%s", Server)
	case Partner != "":
		return fmt.Sprintf("/api/partners/%s", Partner)
	default:
		panic("unknown certificate recipient")
	}
}

// ######################## GET ##########################

type CertGet struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes"`
}

func (c *CertGet) Execute([]string) error {
	addr.Path = path.Join(getCertPath(), "certificates", c.Args.Cert)

	cert := &api.OutCrypto{}
	if err := get(cert); err != nil {
		return err
	}

	displayCertificate(getColorable(), cert)

	return nil
}

// ######################## ADD ##########################

type CertAdd struct {
	Name        string         `required:"true" short:"n" long:"name" description:"The certificate's name"`
	PrivateKey  flags.Filename `short:"p" long:"private_key" description:"The path to the certificate's private key file"`
	PublicKey   flags.Filename `short:"b" long:"public_key" description:"The path to the certificate's public key file"`
	Certificate flags.Filename `short:"c" long:"certificate" description:"The path to the certificate file"`
}

func (c *CertAdd) Execute([]string) error {
	inCrypto := &api.InCrypto{
		Name: &c.Name,
	}

	if c.PrivateKey != "" {
		pk, err := ioutil.ReadFile(string(c.PrivateKey))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.PrivateKey, err)
		}

		inCrypto.PrivateKey = utils.StringPtr(string(pk))
	}

	if c.PublicKey != "" {
		pbk, err := ioutil.ReadFile(string(c.PublicKey))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.PublicKey, err)
		}

		inCrypto.PublicKey = utils.StringPtr(string(pbk))
	}

	if c.Certificate != "" {
		cert, err := ioutil.ReadFile(string(c.Certificate))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.Certificate, err)
		}

		inCrypto.Certificate = utils.StringPtr(string(cert))
	}

	addr.Path = path.Join(getCertPath(), "certificates")

	if err := add(inCrypto); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The certificate", bold(c.Name), "was successfully added.")

	return nil
}

// ######################## DELETE ##########################

type CertDelete struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes"`
}

func (c *CertDelete) Execute([]string) error {
	addr.Path = path.Join(getCertPath(), "certificates", c.Args.Cert)

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The certificate", c.Args.Cert, "was successfully deleted.")

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type CertList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+"`
}

func (c *CertList) Execute([]string) error {
	addr.Path = path.Join(getCertPath(), "certificates")

	listURL(&c.ListOptions, c.SortBy)

	body := map[string][]api.OutCrypto{}
	if err := list(&body); err != nil {
		return err
	}

	w := getColorable() //nolint:ifshort // decrease readability

	if certs := body["certificates"]; len(certs) > 0 {
		fmt.Fprintln(w, bold("Certificates:"))

		for i := range certs {
			displayCertificate(w, &certs[i])
		}
	} else {
		fmt.Fprintln(w, "No certificates found.")
	}

	return nil
}

// ######################## UPDATE ##########################

type CertUpdate struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes"`
	Name        *string        `short:"n" long:"name" description:"The certificate's name"`
	PrivateKey  flags.Filename `short:"p" long:"private_key" description:"The path to the certificate's private key file"`
	PublicKey   flags.Filename `short:"b" long:"public_key" description:"The path to the certificate's public key file"`
	Certificate flags.Filename `short:"c" long:"certificate" description:"The path to the certificate file"`
}

func (c *CertUpdate) Execute([]string) error {
	inCrypto := &api.InCrypto{
		Name: c.Name,
	}

	if c.PrivateKey != "" {
		pk, err := ioutil.ReadFile(string(c.PrivateKey))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.PrivateKey, err)
		}

		inCrypto.PrivateKey = utils.StringPtr(string(pk))
	}

	if c.PublicKey != "" {
		pbk, err := ioutil.ReadFile(string(c.PublicKey))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.PublicKey, err)
		}

		inCrypto.PublicKey = utils.StringPtr(string(pbk))
	}

	if c.Certificate != "" {
		cert, err := ioutil.ReadFile(string(c.Certificate))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.Certificate, err)
		}

		inCrypto.Certificate = utils.StringPtr(string(cert))
	}

	addr.Path = path.Join(getCertPath(), "certificates", c.Args.Cert)

	if err := update(inCrypto); err != nil {
		return err
	}

	name := c.Args.Cert
	if inCrypto.Name != nil && *inCrypto.Name != "" {
		name = *inCrypto.Name
	}

	fmt.Fprintln(getColorable(), "The certificate", bold(name), "was successfully updated.")

	return nil
}
