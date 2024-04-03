package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func checkRuleDir(direction string) error {
	if direction != directionSend && direction != directionRecv {
		return fmt.Errorf("invalid rule direction '%s': %w", direction, errBadArgs)
	}

	return nil
}

func direction(isSend bool) string {
	if isSend {
		return directionSend
	}

	return directionRecv
}

func DisplayRule(w io.Writer, rule *api.OutRule) {
	f := NewFormatter(w)
	defer f.Render()

	displayRule(f, rule)
}

func displayRule(f *Formatter, rule *api.OutRule) {
	f.Title("Rule %q (%s)", rule.Name, direction(rule.IsSend))
	f.Indent()

	defer f.UnIndent()

	f.ValueCond("Comment", rule.Comment)
	f.Value("Path", rule.Path)
	f.ValueCond("Local directory", rule.LocalDir)
	f.ValueCond("Remote directory", rule.RemoteDir)
	f.ValueCond("Temp receive directory", rule.TmpLocalRcvDir)
	displayTaskChain(f, "Pre tasks", rule.PreTasks)
	displayTaskChain(f, "Post tasks", rule.PostTasks)
	displayTaskChain(f, "Error tasks", rule.ErrorTasks)
	displayRuleAccess(f, rule.Authorized)
}

func warnRuleInPathDeprecated(w io.Writer) {
	fmt.Fprintln(w, "[WARNING] The '-i' ('--in_path') option is deprecated. "+
		"Use '--local-dir' and '--remote-dir' instead.")
}

func warnRuleOutPathDeprecated(w io.Writer) {
	fmt.Fprintln(w, "[WARNING] The '-o' ('--out_path') option is deprecated. "+
		"Use '--local-dir' and '--remote-dir' instead.")
}

func warnRuleWorkPathDeprecated(w io.Writer) {
	fmt.Fprintln(w, "[WARNING] The '-w' ('--work_path') option is deprecated. "+
		"Use '--tmp-dir' instead.")
}

// ######################## GET ##########################

//nolint:lll // struct tags for command line arguments can be long
type RuleGet struct {
	Args struct {
		Name      string `required:"yes" positional-arg-name:"name" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction" choice:"send" choice:"receive"`
	} `positional-args:"yes"`
}

func (r *RuleGet) Execute([]string) error { return r.execute(stdOutput) }
func (r *RuleGet) execute(w io.Writer) error {
	if err := checkRuleDir(r.Args.Direction); err != nil {
		return err
	}

	addr.Path = path.Join("/api/rules", r.Args.Name, strings.ToLower(r.Args.Direction))

	rule := &api.OutRule{}
	if err := get(rule); err != nil {
		return err
	}

	DisplayRule(w, rule)

	return nil
}

// ######################## ADD ##########################

//nolint:lll,tagliatelle // struct tags for command line arguments can be long
type RuleAdd struct {
	Name          string       `required:"true" short:"n" long:"name" description:"The rule's name" json:"name,omitempty"`
	Comment       string       `short:"c" long:"comment" description:"A short comment describing the rule" json:"comment,omitempty"`
	Direction     string       `required:"true" short:"d" long:"direction" description:"The direction of the file transfer" choice:"send" choice:"receive" json:"-"`
	IsSend        bool         `json:"isSend,omitempty"`
	Path          string       `short:"p" long:"path" description:"The path used to identify the rule, by default, the rule's name is used" json:"path,omitempty"`
	LocalDir      string       `long:"local-dir" description:"The directory for files on the local disk" json:"localDir,omitempty"`
	RemoteDir     string       `long:"remote-dir" description:"The directory for files on the remote host" json:"remoteDir,omitempty"`
	TmpReceiveDir string       `long:"tmp-dir" description:"The local temp directory for partially received files" json:"tmpLocalRcvDir,omitempty"`
	PreTasks      []jsonObject `short:"r" long:"pre" description:"A pre-transfer task in JSON format, can be repeated" json:"preTasks,omitempty"`
	PostTasks     []jsonObject `short:"s" long:"post" description:"A post-transfer task in JSON format, can be repeated" json:"postTasks,omitempty"`
	ErrorTasks    []jsonObject `short:"e" long:"err" description:"A transfer error task in JSON format, can be repeated" json:"errorTasks,omitempty"`

	// Deprecated options
	InPath   string `short:"i" long:"in_path" description:"[DEPRECATED] The path to the destination of the file" json:"inPath,omitempty"` // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  string `short:"o" long:"out_path" description:"[DEPRECATED] The path to the source of the file" json:"outPath,omitempty"`    // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath string `short:"w" long:"work_path" description:"[DEPRECATED] The path to write the received file" json:"workPath,omitempty"` // Deprecated: replaced by TmpReceiveDir
}

func (r *RuleAdd) Execute([]string) error { return r.execute(stdOutput) }
func (r *RuleAdd) execute(w io.Writer) error {
	r.IsSend = r.Direction == directionSend

	if r.InPath != "" {
		warnRuleInPathDeprecated(w)
	}

	if r.OutPath != "" {
		warnRuleOutPathDeprecated(w)
	}

	if r.WorkPath != "" {
		warnRuleWorkPathDeprecated(w)
	}

	addr.Path = "/api/rules"

	if _, err := add(w, r); err != nil {
		return err
	}

	fmt.Fprintf(w, "The rule %q was successfully added.\n", r.Name)

	return nil
}

// ######################## DELETE ##########################

//nolint:lll // struct tags for command line arguments can be long
type RuleDelete struct {
	Args struct {
		Name      string `required:"yes" positional-arg-name:"name" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction"  choice:"send" choice:"receive"`
	} `positional-args:"yes"`
}

func (r *RuleDelete) Execute([]string) error { return r.execute(stdOutput) }
func (r *RuleDelete) execute(w io.Writer) error {
	if err := checkRuleDir(r.Args.Direction); err != nil {
		return err
	}

	addr.Path = path.Join("/api/rules", r.Args.Name, r.Args.Direction)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The rule %q was successfully deleted.\n", r.Args.Name)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type RuleList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+"`
}

func (r *RuleList) Execute([]string) error { return r.execute(stdOutput) }

//nolint:dupl //duplicate is for a completely different type, keep separate
func (r *RuleList) execute(w io.Writer) error {
	addr.Path = "/api/rules"

	listURL(&r.ListOptions, r.SortBy)

	body := map[string][]*api.OutRule{}
	if err := list(&body); err != nil {
		return err
	}

	if rules := body["rules"]; len(rules) > 0 {
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("Rules:")

		for _, rule := range rules {
			displayRule(f, rule)
		}
	} else {
		fmt.Fprintln(w, "No rules found.")
	}

	return nil
}

// ######################## UPDATE ##########################

//nolint:lll,tagliatelle // struct tags for command line arguments can be long
type RuleUpdate struct {
	Args struct {
		Name      string `required:"yes" positional-arg-name:"name" description:"The server's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction" choice:"send" choice:"receive"`
	} `positional-args:"yes" json:"-"`

	Name          string       `short:"n" long:"name" description:"The rule's name" json:"name,omitempty"`
	Comment       string       `short:"c" long:"comment" description:"A short comment describing the rule" json:"comment,omitempty"`
	Path          string       `short:"p" long:"path" description:"The path used to identify the rule" json:"path,omitempty"`
	LocalDir      string       `long:"local-dir" description:"The directory for files on the local disk" json:"localDir,omitempty"`
	RemoteDir     string       `long:"remote-dir" description:"The directory for files on the remote host" json:"remoteDir,omitempty"`
	TmpReceiveDir string       `long:"tmp-dir" description:"The local temp directory for partially received files" json:"tmpLocalRcvDir,omitempty"`
	PreTasks      []jsonObject `short:"r" long:"pre" description:"A pre-transfer task in JSON format, can be repeated" json:"preTasks,omitempty"`
	PostTasks     []jsonObject `short:"s" long:"post" description:"A post-transfer task in JSON format, can be repeated" json:"postTasks,omitempty"`
	ErrorTasks    []jsonObject `short:"e" long:"err" description:"A transfer error task in JSON format, can be repeated" json:"errorTasks,omitempty"`

	// Deprecated options
	InPath   string `short:"i" long:"in_path" description:"[DEPRECATED] The path to the destination of the file" json:"inPath,omitempty"` // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  string `short:"o" long:"out_path" description:"[DEPRECATED] The path to the source of the file" json:"outPath,omitempty"`    // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath string `short:"w" long:"work_path" description:"[DEPRECATED] The path to write the received file" json:"workPath,omitempty"` // Deprecated: replaced by TmpReceiveDir
}

func (r *RuleUpdate) Execute([]string) error { return r.execute(stdOutput) }
func (r *RuleUpdate) execute(w io.Writer) error {
	if err := checkRuleDir(r.Args.Direction); err != nil {
		return err
	}

	addr.Path = path.Join("/api/rules", r.Args.Name, strings.ToLower(r.Args.Direction))

	if r.InPath != "" {
		warnRuleInPathDeprecated(w)
	}

	if r.OutPath != "" {
		warnRuleOutPathDeprecated(w)
	}

	if r.WorkPath != "" {
		warnRuleWorkPathDeprecated(w)
	}

	if err := update(w, r); err != nil {
		return err
	}

	name := r.Args.Name
	if r.Name != "" {
		name = r.Name
	}

	fmt.Fprintf(w, "The rule %q was successfully updated.\n", name)

	return nil
}

// ######################## RESTRICT ##########################

//nolint:lll // struct tags for command line arguments can be long
type RuleAllowAll struct {
	Args struct {
		Name      string `required:"yes" positional-arg-name:"name" description:"The rule's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction" choice:"send" choice:"receive"`
	} `positional-args:"yes"`
}

func (r *RuleAllowAll) Execute([]string) error { return r.execute(stdOutput) }
func (r *RuleAllowAll) execute(w io.Writer) error {
	if err := checkRuleDir(r.Args.Direction); err != nil {
		return err
	}

	addr.Path = fmt.Sprintf("/api/rules/%s/%s/allow_all", r.Args.Name,
		strings.ToLower(r.Args.Direction))

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, err := sendRequest(ctx, nil, http.MethodPut)
	if err != nil {
		return err
	}

	defer resp.Body.Close() //nolint:errcheck,gosec // error is irrelevant

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Fprintf(w, "The use of the %s rule %q is now unrestricted.\n",
			r.Args.Direction, r.Args.Name)

		return nil

	case http.StatusNotFound:
		return getResponseErrorMessage(resp)

	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status,
			getResponseErrorMessage(resp))
	}
}
