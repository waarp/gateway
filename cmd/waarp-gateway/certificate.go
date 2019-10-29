package main

import (
	"fmt"
	"io/ioutil"
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type certificateCommand struct {
	Get    certGetCommand    `command:"get" description:"Retrieve a certificate's information"`
	Add    certAddCommand    `command:"add" description:"Add a new certificate"`
	Delete certDeleteCommand `command:"delete" description:"Delete a certificate"`
	List   certListCommand   `command:"list" description:"List the known certificates"`
	Update certUpdateCommand `command:"update" description:"Update an existing certificate"`
}

func displayCertificate(cert model.Cert) {
	w := getColorable()

	fmt.Fprintf(w, "\033[37;1;4mCertificate n°%v:\033[0m\n", cert.ID)
	fmt.Fprintf(w, "        \033[37mName:\033[0m \033[37;1m%s\033[0m\n", cert.Name)
	fmt.Fprintf(w, "        \033[37mType:\033[0m \033[33m%s\033[0m\n", cert.OwnerType)
	fmt.Fprintf(w, "       \033[37mOwner:\033[0m \033[33m%v\033[0m\n", cert.OwnerID)
	fmt.Fprintf(w, " \033[37mPrivate key:\033[0m \033[90m%s\033[0m\n", string(cert.PrivateKey))
	fmt.Fprintf(w, "  \033[37mPublic key:\033[0m \033[90m%s\033[0m\n", string(cert.PublicKey))
	fmt.Fprintf(w, "     \033[37mContent:\033[0m \033[90m%v\033[0m\n", cert.Certificate)
}

// ######################## GET ##########################

type certGetCommand struct{}

func (c *certGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing certificate ID")
	}

	res := model.Cert{}
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.CertificatesPath + "/" + args[0]

	if err := getCommand(&res, conn); err != nil {
		return err
	}

	displayCertificate(res)

	return nil
}

// ######################## ADD ##########################

type certAddCommand struct {
	Name        string `required:"true" short:"n" long:"name" description:"The certificate's name"`
	Type        string `required:"true" short:"t" long:"type" description:"The type of the certificates's owner" choice:"local_agents" choice:"remote_agents" choice:"local_accounts" choice:"remote_accounts"`
	Owner       uint64 `required:"true" short:"o" long:"owner" description:"The ID of the certificate's owner"`
	PrivateKey  string `long:"private_key" description:"The path to the certificate's private key file"`
	PublicKey   string `long:"public_key" description:"The path to the certificate's public key file"`
	Certificate string `long:"certificate" description:"The path to the certificate file"`
}

func (c *certAddCommand) Execute(_ []string) error {
	var prK, puK, crt []byte
	var err error

	if c.PrivateKey != "" {
		prK, err = ioutil.ReadFile(c.PrivateKey)
		if err != nil {
			return err
		}
	}
	if c.PublicKey != "" {
		puK, err = ioutil.ReadFile(c.PublicKey)
		if err != nil {
			return err
		}
	}
	if c.Certificate != "" {
		crt, err = ioutil.ReadFile(c.Certificate)
		if err != nil {
			return err
		}
	}

	newCert := model.Cert{
		OwnerType:   c.Type,
		OwnerID:     c.Owner,
		Name:        c.Name,
		PrivateKey:  prK,
		PublicKey:   puK,
		Certificate: crt,
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.CertificatesPath

	loc, err := addCommand(newCert, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The certificate \033[33m'%s'\033[0m was successfully added. "+
		"It can be consulted at the address: \033[37m%s\033[0m\n", newCert.Name, loc)

	return nil
}

// ######################## DELETE ##########################

type certDeleteCommand struct{}

func (c *certDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing certificate ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.CertificatesPath + "/" + args[0]

	if err := deleteCommand(conn); err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The certificate n°\033[33m%s\033[0m was successfully deleted from "+
		"the database\n", args[0])

	return nil
}

// ######################## LIST ##########################

type certListCommand struct {
	listOptions
	SortBy  string   `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name" default:"name"`
	Access  []uint64 `long:"access" description:"Filter the certificates based on the ID of the local account they are attached to. Can be repeated multiple times to filter multiple accounts."`
	Account []uint64 `long:"account" description:"Filter the certificates based on the ID of the remote account they are attached to. Can be repeated multiple times to filter multiple accounts."`
	Partner []uint64 `long:"partner" description:"Filter the certificates based on the ID of the distant partner they are attached to. Can be repeated multiple times to filter multiple partners."`
	Server  []uint64 `long:"server" description:"Filter the certificates based on the ID of the local server they are attached to. Can be repeated multiple times to filter multiple servers."`
}

func (c *certListCommand) Execute(_ []string) error {
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}

	conn.Path = admin.APIPath + admin.CertificatesPath
	query := url.Values{}
	query.Set("limit", fmt.Sprint(c.Limit))
	query.Set("offset", fmt.Sprint(c.Offset))
	query.Set("sortby", c.SortBy)
	if c.DescOrder {
		query.Set("order", "desc")
	}
	for _, acc := range c.Access {
		query.Add("local_accounts", fmt.Sprint(acc))
	}
	for _, acc := range c.Access {
		query.Add("remote_accounts", fmt.Sprint(acc))
	}
	for _, par := range c.Partner {
		query.Add("remote_agents", fmt.Sprint(par))
	}
	for _, ser := range c.Server {
		query.Add("local_agents", fmt.Sprint(ser))
	}
	conn.RawQuery = query.Encode()

	res := map[string][]model.Cert{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	certs := res["certificates"]
	if len(certs) > 0 {
		fmt.Fprintf(w, "\033[33mCertificates:\033[0m\n")
		for _, cert := range certs {
			displayCertificate(cert)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo certificates found\033[0m")
	}

	return nil
}

// ######################## UPDATE ##########################

type certUpdateCommand struct {
	Name        string `short:"n" long:"name" description:"The certificate's name"`
	Type        string `short:"t" long:"type" description:"The type of the certificates's owner" choice:"local_agents" choice:"remote_agents" choice:"local_accounts" choice:"remote_accounts"`
	Owner       uint64 `short:"o" long:"owner" description:"The ID of the certificate's owner"`
	PrivateKey  string `long:"private_key" description:"The path to the certificate's private key file"`
	PublicKey   string `long:"public_key" description:"The path to the certificate's public key file"`
	Certificate string `long:"certificate" description:"The path to the certificate file"`
}

func (c *certUpdateCommand) Execute(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("missing certificate ID")
	}

	newCert := map[string]interface{}{}
	if c.Name != "" {
		newCert["name"] = c.Name
	}
	if c.Type != "" {
		newCert["ownerType"] = c.Type
	}
	if c.Owner != 0 {
		newCert["ownerID"] = c.Owner
	}
	if c.PrivateKey != "" {
		prK, err := ioutil.ReadFile(c.PrivateKey)
		if err != nil {
			return err
		}
		newCert["privateKey"] = prK
	}
	if c.PublicKey != "" {
		puK, err := ioutil.ReadFile(c.PublicKey)
		if err != nil {
			return err
		}
		newCert["publicKey"] = puK
	}
	if c.PrivateKey != "" {
		crt, err := ioutil.ReadFile(c.Certificate)
		if err != nil {
			return err
		}
		newCert["cert"] = crt
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.CertificatesPath + "/" + args[0]

	_, err = updateCommand(newCert, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The certificate n°\033[33m%s\033[0m was successfully updated\n", args[0])

	return nil
}
