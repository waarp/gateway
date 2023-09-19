package wg

import (
	"context"
	"encoding/json"
	"errors"
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

func parseTasks(rule *api.UptRule, pre, post, errs []string) error {
	if len(pre) > 0 {
		preDecoder := json.NewDecoder(strings.NewReader("[" + strings.Join(pre, ",") + "]"))
		preDecoder.DisallowUnknownFields()

		if err := preDecoder.Decode(&rule.PreTasks); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("invalid pre task: %w", err)
		}
	}

	if len(post) > 0 {
		postDecoder := json.NewDecoder(strings.NewReader("[" + strings.Join(post, ",") + "]"))
		postDecoder.DisallowUnknownFields()

		if err := postDecoder.Decode(&rule.PostTasks); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("invalid post task: %w", err)
		}
	}

	if len(errs) > 0 {
		errDecoder := json.NewDecoder(strings.NewReader("[" + strings.Join(errs, ",") + "]"))
		errDecoder.DisallowUnknownFields()

		if err := errDecoder.Decode(&rule.ErrorTasks); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("invalid error task: %w", err)
		}
	}

	return nil
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

//nolint:lll // struct tags for command line arguments can be long
type RuleAdd struct {
	Name          string   `required:"true" short:"n" long:"name" description:"The rule's name"`
	Comment       *string  `short:"c" long:"comment" description:"A short comment describing the rule"`
	Direction     string   `required:"true" short:"d" long:"direction" description:"The direction of the file transfer" choice:"send" choice:"receive"`
	Path          string   `short:"p" long:"path" description:"The path used to identify the rule, by default, the rule's name is used"`
	LocalDir      *string  `long:"local-dir" description:"The directory for files on the local disk"`
	RemoteDir     *string  `long:"remote-dir" description:"The directory for files on the remote host"`
	TmpReceiveDir *string  `long:"tmp-dir" description:"The local temp directory for partially received files "`
	PreTasks      []string `short:"r" long:"pre" description:"A pre-transfer task in JSON format, can be repeated"`
	PostTasks     []string `short:"s" long:"post" description:"A post-transfer task in JSON format, can be repeated"`
	ErrorTasks    []string `short:"e" long:"err" description:"A transfer error task in JSON format, can be repeated"`

	// Deprecated options
	InPath   *string `short:"i" long:"in_path" description:"[DEPRECATED] The path to the destination of the file"` // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  *string `short:"o" long:"out_path" description:"[DEPRECATED] The path to the source of the file"`     // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath *string `short:"w" long:"work_path" description:"[DEPRECATED] The path to write the received file"`   // Deprecated: replaced by TmpReceiveDir
}

func (r *RuleAdd) Execute([]string) error { return r.execute(stdOutput) }
func (r *RuleAdd) execute(w io.Writer) error {
	isSend := r.Direction == directionSend
	rule := &api.InRule{
		UptRule: &api.UptRule{
			Name:           &r.Name,
			Comment:        r.Comment,
			Path:           &r.Path,
			LocalDir:       r.LocalDir,
			RemoteDir:      r.RemoteDir,
			TmpLocalRcvDir: r.TmpReceiveDir,
		},
		IsSend: &isSend,
	}

	if r.InPath != nil {
		fmt.Fprintln(w, "[WARNING] The '-i' ('--in_path') option is deprecated. "+
			"Use '--local-dir' and '--remote-dir' instead.")

		rule.InPath = r.InPath
	}

	if r.OutPath != nil {
		fmt.Fprintln(w, "[WARNING] The '-o' ('--out_path') option is deprecated. "+
			"Use '--local-dir' and '--remote-dir' instead.")

		rule.OutPath = r.OutPath
	}

	if r.WorkPath != nil {
		fmt.Fprintln(w, "[WARNING] The '-w' ('--work_path') option is deprecated. "+
			"Use '--tmp-dir' instead.")

		rule.WorkPath = r.WorkPath
	}

	if err := parseTasks(rule.UptRule, r.PreTasks, r.PostTasks, r.ErrorTasks); err != nil {
		return err
	}

	addr.Path = "/api/rules"

	if _, err := add(w, rule); err != nil {
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

//nolint:lll // struct tags for command line arguments can be long
type RuleUpdate struct {
	Args struct {
		Name      string `required:"yes" positional-arg-name:"name" description:"The server's name"`
		Direction string `required:"yes" positional-arg-name:"direction" description:"The rule's direction" choice:"send" choice:"receive"`
	} `positional-args:"yes"`
	Name          *string  `short:"n" long:"name" description:"The rule's name"`
	Comment       *string  `short:"c" long:"comment" description:"A short comment describing the rule"`
	Path          *string  `short:"p" long:"path" description:"The path used to identify the rule"`
	LocalDir      *string  `long:"local-dir" description:"The directory for files on the local disk"`
	RemoteDir     *string  `long:"remote-dir" description:"The directory for files on the remote host"`
	TmpReceiveDir *string  `long:"tmp-dir" description:"The local temp directory for partially received files "`
	PreTasks      []string `short:"r" long:"pre" description:"A pre-transfer task in JSON format, can be repeated"`
	PostTasks     []string `short:"s" long:"post" description:"A post-transfer task in JSON format, can be repeated"`
	ErrorTasks    []string `short:"e" long:"err" description:"A transfer error task in JSON format, can be repeated"`

	// Deprecated options
	InPath   *string `short:"i" long:"in_path" description:"[DEPRECATED] The path to the destination of the file"` // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  *string `short:"o" long:"out_path" description:"[DEPRECATED] The path to the source of the file"`     // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath *string `short:"w" long:"work_path" description:"[DEPRECATED] The path to write the received file"`   // Deprecated: replaced by TmpReceiveDir
}

func (r *RuleUpdate) Execute([]string) error { return r.execute(stdOutput) }
func (r *RuleUpdate) execute(w io.Writer) error {
	if err := checkRuleDir(r.Args.Direction); err != nil {
		return err
	}

	addr.Path = path.Join("/api/rules", r.Args.Name, strings.ToLower(r.Args.Direction))

	rule := &api.UptRule{
		Name:           r.Name,
		Comment:        r.Comment,
		Path:           r.Path,
		LocalDir:       r.LocalDir,
		RemoteDir:      r.RemoteDir,
		TmpLocalRcvDir: r.TmpReceiveDir,
	}

	if r.InPath != nil {
		fmt.Fprintln(w, "[WARNING] The '-i' ('--in_path') option is deprecated. "+
			"Use '--local-dir' and '--remote-dir' instead.")

		rule.InPath = r.InPath
	}

	if r.OutPath != nil {
		fmt.Fprintln(w, "[WARNING] The '-o' ('--out_path') option is deprecated. "+
			"Use '--local-dir' and '--remote-dir' instead.")

		rule.OutPath = r.OutPath
	}

	if r.WorkPath != nil {
		fmt.Fprintln(w, "[WARNING] The '-w' ('--work_path') option is deprecated. "+
			"Use '--tmp-dir' instead.")

		rule.WorkPath = r.WorkPath
	}

	if err := parseTasks(rule, r.PreTasks, r.PostTasks, r.ErrorTasks); err != nil {
		return err
	}

	if err := update(w, rule); err != nil {
		return err
	}

	name := r.Args.Name
	if rule.Name != nil && *rule.Name != "" {
		name = *rule.Name
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
