package wg

import (
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/jedib0t/go-pretty/v6/text"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

var errUnknownCredentialRecipient = errors.New("unknown credential recipient")

func getCredentialPath() (string, error) {
	switch {
	case Partner != "" && RemoteAccount != "":
		return fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, RemoteAccount), nil
	case Server != "" && LocalAccount != "":
		return fmt.Sprintf("/api/servers/%s/accounts/%s", Server, LocalAccount), nil
	case Partner != "":
		return fmt.Sprintf("/api/partners/%s", Partner), nil
	case Server != "":
		return fmt.Sprintf("/api/servers/%s", Server), nil
	default:
		return "", errUnknownCredentialRecipient
	}
}

func DisplayCredential(w io.Writer, cred *api.OutCred) {
	f := NewFormatter(w)
	defer f.Render()

	displayCredential(f, cred)
}

func displayCredential(f *Formatter, cred *api.OutCred) {
	switch cred.Type {
	case auth.PasswordHash:
		f.Title("Password hash %q: %s", cred.Name, cred.Value)
	case auth.Password:
		f.Title("Password %q: %s", cred.Name, cred.Value)
	case auth.TLSCertificate, auth.TLSTrustedCertificate:
		displayTLSInfo(f, cred.Name, cred.Value)
	case sftp.AuthSSHPublicKey:
		displaySSHKeyInfo(f, cred.Name, cred.Value)
	case sftp.AuthSSHPrivateKey:
		displayPrivateKeyInfo(f, cred.Name, cred.Value)
	case r66.AuthLegacyCertificate:
		f.Title("Legacy R66 certificate %q", cred.Name)
	default:
		f.Println(text.FgRed.Sprintf("Unknown credential type %q", cred.Type))
	}
}

//nolint:lll //flags tags are long
type CredentialAdd struct {
	Name   string      `short:"n" long:"name" description:"The credential's name" json:"name,omitempty"`
	Type   string      `required:"true" short:"t" long:"type" description:"The type of credential" json:"type,omitempty"`
	Value  fileOrValue `short:"v" long:"value" description:"The credential value. Can also be a file path." json:"value,omitempty"`
	Value2 fileOrValue `short:"s" long:"secondary-value" description:"The secondary credential value (when applicable). Can also be a filepath." json:"value2,omitempty"`
}

func (a *CredentialAdd) Execute([]string) error { return a.execute(stdOutput) }
func (a *CredentialAdd) execute(w io.Writer) error {
	credPath, pathErr := getCredentialPath()
	if pathErr != nil {
		return pathErr
	}

	addr.Path = path.Join(credPath, "credentials")

	if _, err := add(w, a); err != nil {
		return err
	}

	name := a.Name
	if name == "" {
		name = a.Type
	}

	fmt.Fprintf(w, "The %q credential was successfully added.\n", name)

	return nil
}

type CredentialGet struct {
	Args struct {
		Credential string `required:"yes" positional-arg-name:"credential" description:"The credential's name"`
	} `positional-args:"yes"`
}

func (a *CredentialGet) Execute([]string) error { return a.execute(stdOutput) }
func (a *CredentialGet) execute(w io.Writer) error {
	credPath, pathErr := getCredentialPath()
	if pathErr != nil {
		return pathErr
	}

	addr.Path = path.Join(credPath, "credentials", a.Args.Credential)

	var cred api.OutCred
	if err := get(&cred); err != nil {
		return err
	}

	DisplayCredential(w, &cred)

	return nil
}

type CredentialDelete struct {
	Args struct {
		Credential string `required:"yes" positional-arg-name:"credential" description:"The credential's name"`
	} `positional-args:"yes"`
}

func (a *CredentialDelete) Execute([]string) error { return a.execute(stdOutput) }
func (a *CredentialDelete) execute(w io.Writer) error {
	credPath, pathErr := getCredentialPath()
	if pathErr != nil {
		return pathErr
	}

	addr.Path = path.Join(credPath, "credentials", a.Args.Credential)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The %q credential was successfully removed.\n", a.Args.Credential)

	return nil
}
