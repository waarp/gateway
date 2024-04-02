package wg

import (
	"fmt"
	"sort"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

const NotApplicable = "N/A"

func displayMap[T any](f *Formatter, m map[string]T) {
	pairs := make([]pair, 0, len(m))

	for key, val := range m {
		pairs = append(pairs, pair{key: key, val: val})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})

	for i := range pairs {
		f.Value(pairs[i].key, pairs[i].val)
	}
}

func displayProtoConfig(f *Formatter, cfg map[string]any) {
	if len(cfg) == 0 {
		f.Empty("Configuration", "<empty>")

		return
	}

	f.Title("Configuration")
	f.Indent()

	defer f.UnIndent()

	displayMap(f, cfg)
}

func displayAuthorizedRules(f *Formatter, auth api.AuthorizedRules) {
	f.Title("Authorized rules")
	f.Indent()

	defer f.UnIndent()

	if len(auth.Sending) == 0 {
		f.Empty("Send", "<none>")
	} else {
		f.Value("Send", strings.Join(auth.Sending, ", "))
	}

	if len(auth.Sending) == 0 {
		f.Empty("Receive", "<none>")
	} else {
		f.Value("Receive", strings.Join(auth.Reception, ", "))
	}
}

//nolint:varnamelen //formatter name is kept short for readability
func displayRuleAccess(f *Formatter, access api.RuleAccess) {
	if len(access.LocalServers) == 0 && len(access.RemotePartners) == 0 &&
		len(access.LocalAccounts) == 0 && len(access.RemoteAccounts) == 0 {
		f.Empty("Rule access", "<unrestricted>")

		return
	}

	f.Title("Rule access")
	f.Indent()

	defer f.UnIndent()

	// Servers
	if len(access.LocalServers) == 0 {
		f.Empty("Local servers", "<none>")
	} else {
		f.Value("Local servers", strings.Join(access.LocalServers, ", "))
	}

	// Partners
	if len(access.RemotePartners) == 0 {
		f.Empty("Remote partners", "<none>")
	} else {
		f.Value("Remote partners", strings.Join(access.RemotePartners, ", "))
	}

	// Local accounts
	if len(access.LocalAccounts) == 0 {
		f.Empty("Local accounts", "<none>")
	} else {
		var fullAccounts []string

		for partner, accounts := range access.LocalAccounts {
			for _, account := range accounts {
				fullAccounts = append(fullAccounts, fmt.Sprintf("%s.%s", partner, account))
			}
		}

		sort.Strings(fullAccounts)
		f.Value("Local accounts", strings.Join(fullAccounts, ", "))
	}

	// Remote accounts
	if len(access.RemoteAccounts) == 0 {
		f.Empty("Remote accounts", "<none>")
	} else {
		var fullAccounts []string

		for partner, accounts := range access.RemoteAccounts {
			for _, account := range accounts {
				fullAccounts = append(fullAccounts, fmt.Sprintf("%s.%s", partner, account))
			}
		}

		sort.Strings(fullAccounts)
		f.Value("Remote accounts", strings.Join(fullAccounts, ", "))
	}
}

func displayTask(f *Formatter, task *api.Task) {
	if len(task.Args) == 0 {
		f.Title("Command %q", task.Type)
	} else {
		f.Title("Command %q with args", task.Type)
		f.Indent()
		defer f.UnIndent()

		displayMap(f, task.Args)
	}
}

func displayTaskChain(f *Formatter, title string, chain []*api.Task) {
	if len(chain) == 0 {
		f.Empty(title, "<none>")

		return
	}

	f.Title(title)
	f.Indent()

	defer f.UnIndent()

	for _, task := range chain {
		displayTask(f, task)
	}
}

func displayPermissions(f *Formatter, perms *api.Perms) {
	f.Title("Permissions")
	f.Indent()

	defer f.UnIndent()

	f.ValueWithDefault("Transfers", perms.Transfers, "---")
	f.ValueWithDefault("Servers", perms.Servers, "---")
	f.ValueWithDefault("Partners", perms.Partners, "---")
	f.ValueWithDefault("Rules", perms.Rules, "---")
	f.ValueWithDefault("Users", perms.Users, "---")
	f.ValueWithDefault("Administration", perms.Administration, "---")
}
