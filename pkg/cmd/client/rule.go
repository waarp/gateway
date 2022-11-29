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

func displayTasks(w io.Writer, rule *api.OutRule) {
	chains := []struct {
		name  string
		tasks []*api.Task
	}{
		{"Pre tasks", rule.PreTasks},
		{"Post tasks", rule.PostTasks},
		{"Error tasks", rule.ErrorTasks},
	}

	for i := range chains {
		fmt.Fprintln(w, orange("    "+chains[i].name+":"))

		for i, t := range chains[i].tasks {
			prefix := "    ├─Command"
			if i == len(chains[i].tasks)-1 {
				prefix = "    └─Command"
			}

			fmt.Fprintln(w, bold(prefix), t.Type, bold("with args:"), string(t.Args))
		}
	}
}

func displayRule(w io.Writer, rule *api.OutRule) {
	way := directionRecv
	if rule.IsSend {
		way = directionSend
	}

	var servers, partners, locAcc, remAcc string

	if rule.Authorized != nil {
		servers = strings.Join(rule.Authorized.LocalServers, ", ")
		partners = strings.Join(rule.Authorized.RemotePartners, ", ")

		var la []string

		for server, accounts := range rule.Authorized.LocalAccounts {
			for _, account := range accounts {
				la = append(la, fmt.Sprint(server, ".", account))
			}
		}

		var ra []string

		for partner, accounts := range rule.Authorized.RemoteAccounts {
			for _, account := range accounts {
				ra = append(ra, fmt.Sprint(partner, ".", account))
			}
		}

		locAcc = strings.Join(la, ", ")
		remAcc = strings.Join(ra, ", ")
	}

	fmt.Fprintln(w, orange(bold("● Rule", rule.Name, "("+way+")")))
	fmt.Fprintln(w, orange("    Comment:               "), rule.Comment)
	fmt.Fprintln(w, orange("    Path:                  "), rule.Path)
	fmt.Fprintln(w, orange("    Local directory:       "), rule.LocalDir)
	fmt.Fprintln(w, orange("    Remote directory:      "), rule.RemoteDir)
	fmt.Fprintln(w, orange("    Temp receive directory:"), rule.TmpLocalRcvDir)
	displayTasks(w, rule)
	fmt.Fprintln(w, orange("    Authorized agents:"))
	fmt.Fprintln(w, bold("    ├─Servers:         "), servers)
	fmt.Fprintln(w, bold("    ├─Partners:        "), partners)
	fmt.Fprintln(w, bold("    ├─Server accounts: "), locAcc)
	fmt.Fprintln(w, bold("    └─Partner accounts:"), remAcc)
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

func (r *RuleGet) Execute([]string) error {
	if err := checkRuleDir(r.Args.Direction); err != nil {
		return err
	}

	addr.Path = path.Join("/api/rules", r.Args.Name, strings.ToLower(r.Args.Direction))

	rule := &api.OutRule{}
	if err := get(rule); err != nil {
		return err
	}

	displayRule(getColorable(), rule)

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

func (r *RuleAdd) Execute([]string) error {
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
		fmt.Fprintln(out, "[WARNING] The '-i' ('--in_path') option is deprecated. "+
			"Use '--local-dir' and '--remote-dir' instead.")

		rule.InPath = r.InPath
	}

	if r.OutPath != nil {
		fmt.Fprintln(out, "[WARNING] The '-o' ('--out_path') option is deprecated. "+
			"Use '--local-dir' and '--remote-dir' instead.")

		rule.OutPath = r.OutPath
	}

	if r.WorkPath != nil {
		fmt.Fprintln(out, "[WARNING] The '-w' ('--work_path') option is deprecated. "+
			"Use '--tmp-dir' instead.")

		rule.WorkPath = r.WorkPath
	}

	if err := parseTasks(rule.UptRule, r.PreTasks, r.PostTasks, r.ErrorTasks); err != nil {
		return err
	}

	addr.Path = "/api/rules"

	if err := add(rule); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The rule", bold(r.Name), "was successfully added.")

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

func (r *RuleDelete) Execute([]string) error {
	if err := checkRuleDir(r.Args.Direction); err != nil {
		return err
	}

	addr.Path = path.Join("/api/rules", r.Args.Name, r.Args.Direction)

	if err := remove(); err != nil {
		return err
	}

	fmt.Fprintln(getColorable(), "The rule", bold(r.Args.Name), "was successfully deleted.")

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type RuleList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+"`
}

func (r *RuleList) Execute([]string) error {
	addr.Path = "/api/rules"

	listURL(&r.ListOptions, r.SortBy)

	body := map[string][]api.OutRule{}
	if err := list(&body); err != nil {
		return err
	}

	w := getColorable() //nolint:ifshort // decrease readability

	if rules := body["rules"]; len(rules) > 0 {
		fmt.Fprintln(w, bold("Rules:"))

		for i := range rules {
			rule := rules[i]
			displayRule(w, &rule)
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

func (r *RuleUpdate) Execute([]string) error {
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
		fmt.Fprintln(out, "[WARNING] The '-i' ('--in_path') option is deprecated. "+
			"Use '--local-dir' and '--remote-dir' instead.")

		rule.InPath = r.InPath
	}

	if r.OutPath != nil {
		fmt.Fprintln(out, "[WARNING] The '-o' ('--out_path') option is deprecated. "+
			"Use '--local-dir' and '--remote-dir' instead.")

		rule.OutPath = r.OutPath
	}

	if r.WorkPath != nil {
		fmt.Fprintln(out, "[WARNING] The '-w' ('--work_path') option is deprecated. "+
			"Use '--tmp-dir' instead.")

		rule.WorkPath = r.WorkPath
	}

	if err := parseTasks(rule, r.PreTasks, r.PostTasks, r.ErrorTasks); err != nil {
		return err
	}

	if err := update(rule); err != nil {
		return err
	}

	name := r.Args.Name
	if rule.Name != nil && *rule.Name != "" {
		name = *rule.Name
	}

	fmt.Fprintln(getColorable(), "The rule", bold(name), "was successfully updated.")

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

func (r *RuleAllowAll) Execute([]string) error {
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

	defer resp.Body.Close() //nolint:errcheck // nothing to handle the error

	w := getColorable()

	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Fprintln(w, "The use of the", r.Args.Direction, "rule", bold(r.Args.Name),
			"is now unrestricted.")

		return nil

	case http.StatusNotFound:
		return getResponseErrorMessage(resp)

	default:
		return fmt.Errorf("unexpected error: %w", getResponseErrorMessage(resp))
	}
}
