package wg

import (
	"fmt"
	"io"
	"path"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

const filewatchersAPIPath = "/api/filewatchers"

func displayFilewatchers(w io.Writer, fws []*api.OutFilewatcher) error {
	Style0.Printf(w, "=== Filewatchers ===")
	for _, fw := range fws {
		if err := displayFilewatcher(w, fw); err != nil {
			return err
		}
	}

	return nil
}

func displayFilewatcher(w io.Writer, fw *api.OutFilewatcher) error {
	Style1.Printf(w, "Filewatcher %q", fw.Flow)
	Style22.PrintL(w, "Disabled", fmt.Sprintf("%v", fw.Disabled))
	Style22.PrintL(w, "Interval", time.Duration(fw.Interval).String())
	Style22.PrintL(w, "Pattern", fw.Pattern)
	Style22.PrintL(w, "No duplicate check", fmt.Sprintf("%v", fw.NoDuplicateCheck))
	Style22.PrintL(w, "Partner", fw.Partner)
	Style22.PrintL(w, "Account", fw.Account)
	Style22.PrintL(w, "Client", fw.Client)
	Style22.PrintL(w, "Rule", fw.Rule)

	return nil
}

//nolint:lll //struct tags can be long
type FilewatcherAdd struct {
	Flow             string `required:"yes" short:"f" long:"flow" description:"The name of the filewatcher flow" json:"flow,omitempty"`
	Disabled         bool   `long:"disabled" description:"Create the filewatcher in disabled state" json:"disabled,omitempty"`
	Interval         string `required:"yes" short:"i" long:"interval" description:"The polling interval (e.g. 5m, 30s)" json:"interval,omitempty"`
	Pattern          string `required:"yes" short:"p" long:"pattern" description:"The file pattern to watch (e.g. *.txt)" json:"pattern,omitempty"`
	NoDuplicateCheck bool   `long:"no-duplicate-check" description:"Disable duplicate file detection" json:"noDuplicateCheck,omitempty"`
	Partner          string `required:"yes" long:"partner" description:"The name of the remote partner" json:"partner,omitempty"`
	Account          string `required:"yes" short:"a" long:"account" description:"The login of the remote account" json:"account,omitempty"`
	Client           string `required:"yes" short:"c" long:"client" description:"The name of the client to use" json:"client,omitempty"`
	Rule             string `required:"yes" short:"r" long:"rule" description:"The name of the receive rule to use" json:"rule,omitempty"`
}

func (fw *FilewatcherAdd) Execute([]string) error { return fw.execute(stdOutput) }
func (fw *FilewatcherAdd) execute(w io.Writer) error {
	addr.Path = filewatchersAPIPath

	if _, err := add(w, fw); err != nil {
		return err
	}

	fmt.Fprintf(w, "The filewatcher %q was successfully added.\n", fw.Flow)

	return nil
}

type FilewatcherGet struct {
	OutputFormat

	Args struct {
		Flow string `required:"yes" positional-arg-name:"flow" description:"The name of the filewatcher flow"`
	} `positional-args:"yes"`
}

func (fw *FilewatcherGet) Execute([]string) error { return fw.execute(stdOutput) }
func (fw *FilewatcherGet) execute(w io.Writer) error {
	addr.Path = path.Join(filewatchersAPIPath, fw.Args.Flow)

	var filewatcher api.OutFilewatcher
	if err := get(&filewatcher); err != nil {
		return err
	}

	return outputObject(w, &filewatcher, &fw.OutputFormat, displayFilewatcher)
}

//nolint:lll //struct tags can be long
type FilewatcherList struct {
	ListOptions

	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"flow+" choice:"flow-" default:"flow+"`
}

func (fw *FilewatcherList) Execute([]string) error { return fw.execute(stdOutput) }
func (fw *FilewatcherList) execute(w io.Writer) error {
	addr.Path = filewatchersAPIPath

	listURL(&fw.ListOptions, fw.SortBy)

	body := map[string][]*api.OutFilewatcher{}
	if err := list(&body); err != nil {
		return err
	}

	if fws := body["filewatchers"]; len(fws) > 0 {
		return outputObject(w, fws, &fw.OutputFormat, displayFilewatchers)
	}

	fmt.Fprintln(w, "No filewatchers found.")

	return nil
}

//nolint:lll //struct tags can be long
type FilewatcherUpdate struct {
	Args struct {
		Flow string `required:"yes" positional-arg-name:"flow" description:"The name of the filewatcher flow"`
	} `positional-args:"yes" json:"-"`

	Flow             *string `short:"f" long:"flow" description:"The new name of the filewatcher flow" json:"flow,omitempty"`
	Disabled         *bool   `long:"disabled" description:"Enable or disable the filewatcher" json:"disabled,omitempty"`
	Interval         *string `short:"i" long:"interval" description:"The new polling interval (e.g. 5m, 30s)" json:"interval,omitempty"`
	Pattern          *string `short:"p" long:"pattern" description:"The new file pattern to watch (e.g. *.txt)" json:"pattern,omitempty"`
	NoDuplicateCheck *bool   `long:"no-duplicate-check" description:"Disable or enable duplicate file detection" json:"noDuplicateCheck,omitempty"`
	Partner          *string `long:"partner" description:"The name of the remote partner" json:"partner,omitempty"`
	Account          *string `short:"a" long:"account" description:"The login of the remote account" json:"account,omitempty"`
	Client           *string `short:"c" long:"client" description:"The name of the client to use" json:"client,omitempty"`
	Rule             *string `short:"r" long:"rule" description:"The name of the receive rule to use" json:"rule,omitempty"`
}

func (fw *FilewatcherUpdate) Execute([]string) error { return fw.execute(stdOutput) }
func (fw *FilewatcherUpdate) execute(w io.Writer) error {
	addr.Path = path.Join(filewatchersAPIPath, fw.Args.Flow)

	if err := update(w, fw); err != nil {
		return err
	}

	displayFlow := fw.Args.Flow
	if fw.Flow != nil && *fw.Flow != "" {
		displayFlow = *fw.Flow
	}

	fmt.Fprintf(w, "The filewatcher %q was successfully updated.\n", displayFlow)

	return nil
}

type FilewatcherDelete struct {
	Args struct {
		Flow string `required:"yes" positional-arg-name:"flow" description:"The name of the filewatcher flow"`
	} `positional-args:"yes"`
}

func (fw *FilewatcherDelete) Execute([]string) error { return fw.execute(stdOutput) }
func (fw *FilewatcherDelete) execute(w io.Writer) error {
	addr.Path = path.Join(filewatchersAPIPath, fw.Args.Flow)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The filewatcher %q was successfully deleted.\n", fw.Args.Flow)

	return nil
}

type FilewatcherStart struct {
	Args struct {
		Flow string `required:"yes" positional-arg-name:"flow" description:"The name of the filewatcher flow"`
	} `positional-args:"yes"`
}

func (fw *FilewatcherStart) Execute([]string) error { return fw.execute(stdOutput) }
func (fw *FilewatcherStart) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("%s/%s/start", filewatchersAPIPath, fw.Args.Flow)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The filewatcher %q was successfully started.\n", fw.Args.Flow)

	return nil
}

type FilewatcherStop struct {
	Args struct {
		Flow string `required:"yes" positional-arg-name:"flow" description:"The name of the filewatcher flow"`
	} `positional-args:"yes"`
}

func (fw *FilewatcherStop) Execute([]string) error { return fw.execute(stdOutput) }
func (fw *FilewatcherStop) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("%s/%s/stop", filewatchersAPIPath, fw.Args.Flow)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The filewatcher %q was successfully stopped.\n", fw.Args.Flow)

	return nil
}

type FilewatcherFire struct {
	Args struct {
		Flow string `required:"yes" positional-arg-name:"flow" description:"The name of the filewatcher flow"`
	} `positional-args:"yes"`
}

func (fw *FilewatcherFire) Execute([]string) error { return fw.execute(stdOutput) }
func (fw *FilewatcherFire) execute(w io.Writer) error {
	if err := exec(w, fmt.Sprintf("%s/%s/fire", filewatchersAPIPath, fw.Args.Flow)); err != nil {
		return err
	}

	fmt.Fprintf(w, "The filewatcher %q was successfully fired.\n", fw.Args.Flow)

	return nil
}
