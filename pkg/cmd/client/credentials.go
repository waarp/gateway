package wg

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
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

//nolint:lll //flags tags are long
type credentialAdd struct {
	Name   string      `short:"n" long:"name" description:"The credential's name" json:"name,omitempty"`
	Type   string      `required:"true" short:"t" long:"type" description:"The type of credential" json:"type,omitempty"`
	Value  fileOrValue `short:"v" long:"value" description:"The credential value. Can also be a file path." json:"value,omitempty"`
	Value2 fileOrValue `short:"s" long:"secondary-value" description:"The secondary credential value (when applicable). Can also be a filepath." json:"value2,omitempty"`
}

func (a *credentialAdd) Execute([]string) error { return a.execute(os.Stdout) }
func (a *credentialAdd) execute(w io.Writer) error {
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

type credentialDelete struct {
	Args struct {
		Credential string `required:"yes" positional-arg-name:"credential" description:"The credential's name"`
	} `positional-args:"yes"`
}

func (a *credentialDelete) Execute([]string) error { return a.execute(os.Stdout) }
func (a *credentialDelete) execute(w io.Writer) error {
	credPath, err := getCredentialPath()
	if err != nil {
		return err
	}

	addr.Path = path.Join(credPath, "credentials", a.Args.Credential)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The %q credential was successfully removed.\n", a.Args.Credential)

	return nil
}
