package wg

import (
	"fmt"
	"io"
	"os"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

const cloudsAPIPath = "/api/clouds"

type cloudObject = api.GetCloudRespObject

func DisplayCloud(w io.Writer, cloud *cloudObject) {
	f := NewFormatter(w)
	defer f.Render()

	displayCloud(f, cloud)
}

func displayCloud(f *Formatter, cloud *cloudObject) {
	f.Title("Cloud instance %q (%s)", cloud.Name, cloud.Type)
	f.Indent()

	defer f.UnIndent()

	f.ValueWithDefault("Key", cloud.Key, "<none>")
	displayMap(f, "Options", "<none>", cloud.Options)
}

type CloudGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The name of the cloud instance"`
	} `positional-args:"yes"`
}

func (c *CloudGet) Execute([]string) error { return c.execute(os.Stdout) }
func (c *CloudGet) execute(w io.Writer) error {
	addr.Path = path.Join(cloudsAPIPath, c.Args.Name)

	cloud := &cloudObject{}
	if err := get(cloud); err != nil {
		return err
	}

	DisplayCloud(w, cloud)

	return nil
}

//nolint:lll //tags can be long for flags
type CloudAdd struct {
	Name    string             `required:"yes" short:"n" long:"name" description:"The name of the cloud instance"`
	Type    string             `required:"yes" short:"t" long:"type" description:"The type of the cloud instance"`
	Key     string             `short:"k" long:"key" description:"The authentication key of the cloud instance"`
	Secret  string             `short:"s" long:"secret" description:"The authentication secret of the cloud instance"`
	Options map[string]confVal `short:"o" long:"options" description:"The options of the cloud instance, in key:val format. Can be repeated."`
}

func (c *CloudAdd) Execute([]string) error { return c.execute(os.Stdout) }
func (c *CloudAdd) execute(w io.Writer) error {
	addr.Path = cloudsAPIPath

	newCloud := map[string]any{}

	addIfNotZero(newCloud, "name", c.Name)
	addIfNotZero(newCloud, "type", c.Type)
	addIfNotZero(newCloud, "key", c.Key)
	addIfNotZero(newCloud, "secret", c.Secret)
	addIfNotZero(newCloud, "options", c.Options)

	if err := add(newCloud); err != nil {
		return err
	}

	fmt.Fprintf(w, "The cloud instance %q was successfully added.\n", c.Name)

	return nil
}

type CloudDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The name of the cloud instance"`
	} `positional-args:"yes"`
}

func (c *CloudDelete) Execute([]string) error { return c.execute(os.Stdout) }
func (c *CloudDelete) execute(w io.Writer) error {
	addr.Path = path.Join(cloudsAPIPath, c.Args.Name)

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintf(w, "The cloud instance %q was successfully deleted.\n", c.Args.Name)

	return nil
}

//nolint:lll //tags can be long for flags
type CloudUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The name of the cloud instance"`
	} `positional-args:"yes"`

	Name    string             `required:"yes" short:"n" long:"name" description:"The name of the cloud instance"`
	Type    string             `required:"yes" short:"t" long:"type" description:"The type of the cloud instance"`
	Key     string             `short:"k" long:"key" description:"The authentication key of the cloud instance"`
	Secret  string             `short:"s" long:"secret" description:"The authentication secret of the cloud instance"`
	Options map[string]confVal `short:"o" long:"options" description:"The options of the cloud instance, in key:val format. Can be repeated."`
}

func (c *CloudUpdate) Execute([]string) error { return c.execute(os.Stdout) }
func (c *CloudUpdate) execute(w io.Writer) error {
	addr.Path = path.Join(cloudsAPIPath, c.Args.Name)

	newCloud := map[string]any{}

	addIfNotZero(newCloud, "name", c.Name)
	addIfNotZero(newCloud, "type", c.Type)
	addIfNotZero(newCloud, "key", c.Key)
	addIfNotZero(newCloud, "secret", c.Secret)
	addIfNotZero(newCloud, "options", c.Options)

	if err := update(newCloud); err != nil {
		return err
	}

	finalName := c.Args.Name
	if c.Name != "" {
		finalName = c.Name
	}

	fmt.Fprintf(w, "The cloud instance %q was successfully updated.\n", finalName)

	return nil
}

//nolint:lll //tags can be long for flags
type CloudList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"The property to sort by." choice:"name+" choice:"name-" choice:"type+" choice:"type-" default:"name+"`
}

func (c *CloudList) Execute([]string) error { return c.execute(os.Stdout) }
func (c *CloudList) execute(w io.Writer) error {
	addr.Path = cloudsAPIPath

	listURL(&c.ListOptions, c.SortBy)

	body := map[string][]*cloudObject{}
	if err := list(&body); err != nil {
		return err
	}

	if clouds := body["clouds"]; clouds != nil {
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("Cloud instances:")

		for _, cloud := range clouds {
			displayCloud(f, cloud)
		}
	} else {
		fmt.Fprintf(w, "No cloud instances found.\n")
	}

	return nil
}
