package wg

import (
	"fmt"
	"io"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

const cloudsAPIPath = "/api/clouds"

type cloudObject = api.GetCloudRespObject

func displayCloud(w io.Writer, cloud *cloudObject) {
	style1.printf(w, "Cloud instance %q (%s)", cloud.Name, cloud.Type)
	style22.printL(w, "Key", withDefault(cloud.Key, none))

	if len(cloud.Options) == 0 {
		style22.printL(w, "Options", none)
	} else {
		style22.printf(w, "Options:")
		displayMap(w, style333, cloud.Options)
	}
}

type CloudGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The name of the cloud instance"`
	} `positional-args:"yes"`
}

func (c *CloudGet) Execute([]string) error { return execute(c) }
func (c *CloudGet) execute(w io.Writer) error {
	addr.Path = path.Join(cloudsAPIPath, c.Args.Name)

	cloud := &cloudObject{}
	if err := get(cloud); err != nil {
		return err
	}

	displayCloud(w, cloud)

	return nil
}

//nolint:lll //tags can be long for flags
type CloudAdd struct {
	Name    string             `required:"yes" short:"n" long:"name" description:"The name of the cloud instance" json:"name,omitempty"`
	Type    string             `required:"yes" short:"t" long:"type" description:"The type of the cloud instance" json:"type,omitempty"`
	Key     string             `short:"k" long:"key" description:"The authentication key of the cloud instance" json:"key,omitempty"`
	Secret  string             `short:"s" long:"secret" description:"The authentication secret of the cloud instance" json:"secret,omitempty"`
	Options map[string]confVal `short:"o" long:"options" description:"The options of the cloud instance, in key:val format. Can be repeated." json:"options,omitempty"`
}

func (c *CloudAdd) Execute([]string) error { return execute(c) }
func (c *CloudAdd) execute(w io.Writer) error {
	addr.Path = cloudsAPIPath

	if _, err := add(w, c); err != nil {
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

func (c *CloudDelete) Execute([]string) error { return execute(c) }
func (c *CloudDelete) execute(w io.Writer) error {
	addr.Path = path.Join(cloudsAPIPath, c.Args.Name)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The cloud instance %q was successfully deleted.\n", c.Args.Name)

	return nil
}

//nolint:lll //tags can be long for flags
type CloudUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The name of the cloud instance"`
	} `positional-args:"yes" json:"-"`

	Name    *string             `required:"yes" short:"n" long:"name" description:"The name of the cloud instance" json:"name,omitempty"`
	Type    *string             `required:"yes" short:"t" long:"type" description:"The type of the cloud instance" json:"type,omitempty"`
	Key     *string             `short:"k" long:"key" description:"The authentication key of the cloud instance" json:"key,omitempty"`
	Secret  *string             `short:"s" long:"secret" description:"The authentication secret of the cloud instance" json:"secret,omitempty"`
	Options *map[string]confVal `short:"o" long:"options" description:"The options of the cloud instance, in key:val format. Can be repeated." json:"options,omitempty"`
}

func (c *CloudUpdate) Execute([]string) error { return execute(c) }
func (c *CloudUpdate) execute(w io.Writer) error {
	addr.Path = path.Join(cloudsAPIPath, c.Args.Name)

	if err := update(w, c); err != nil {
		return err
	}

	finalName := c.Args.Name
	if c.Name != nil && *c.Name != "" {
		finalName = *c.Name
	}

	fmt.Fprintf(w, "The cloud instance %q was successfully updated.\n", finalName)

	return nil
}

//nolint:lll //tags can be long for flags
type CloudList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"The property to sort by." choice:"name+" choice:"name-" choice:"type+" choice:"type-" default:"name+"`
}

func (c *CloudList) Execute([]string) error { return execute(c) }
func (c *CloudList) execute(w io.Writer) error {
	addr.Path = cloudsAPIPath

	listURL(&c.ListOptions, c.SortBy)

	body := map[string][]*cloudObject{}
	if err := list(&body); err != nil {
		return err
	}

	if clouds := body["clouds"]; clouds != nil {
		style0.printf(w, "=== Cloud instances ===")

		for _, cloud := range clouds {
			displayCloud(w, cloud)
		}
	} else {
		fmt.Fprintf(w, "No cloud instances found.\n")
	}

	return nil
}
