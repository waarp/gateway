package rules

import (
	"slices"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The objects of the gateway the arguments of a task name (keys, SMTP credentials, templates, partners,
// accounts, clients, servers, rules): the lists of the form, and what the Tasks tab checks a name against.

// namedObjects are the objects of the gateway the arguments of a task name,
// read for the kinds the form needs, as the groups of their select.
type namedObjects map[refKind][]taskGroup

// namesFor reads the objects the arguments of these types name, each kind
// once.
func (h *Handlers) namesFor(specs ...*taskSpec) (namedObjects, error) {
	out := namedObjects{}

	for _, spec := range specs {
		for i := range spec.Args {
			arg := &spec.Args[i]
			if _, read := out[arg.Ref]; arg.Kind != argRef || read {
				continue
			}

			groups, err := h.namesOf(arg.Ref)
			if err != nil {
				return nil, err
			}

			out[arg.Ref] = groups
		}
	}

	return out, nil
}

// has says whether an object of the kind bears the name.
func (n namedObjects) has(kind refKind, name string) bool { return offers(n[kind], name) }

// offers says whether the groups of a select offer the value.
func offers(groups []taskGroup, value string) bool {
	return slices.ContainsFunc(groups, func(g taskGroup) bool {
		return slices.ContainsFunc(g.Options, func(o taskOption) bool { return o.Value == value })
	})
}

// namesOf reads the names of the objects of a kind (those of this gateway:
// the database keeps to them), sorted; the keys with their type, the
// accounts in groups by server or partner.
//
//nolint:cyclop // one case per kind of object
func (h *Handlers) namesOf(kind refKind) ([]taskGroup, error) {
	var (
		names     []string
		protocols map[string]string // the protocols of the clients
		err       error
	)

	switch kind {
	case refKey:
		return h.keyNames()
	case refRemoteAccount, refLocalAccount:
		return h.accountNames(kind)
	case refSMTP:
		var list model.SMTPCredentials
		err = h.db.Select(&list).Run()
		names = collect(list, func(c *model.SMTPCredential) string { return c.EmailAddress })
	case refTemplate:
		var list model.EmailTemplates
		err = h.db.Select(&list).Run()
		names = collect(list, func(t *model.EmailTemplate) string { return t.Name })
	case refPartner:
		var list model.RemoteAgents
		err = h.db.Select(&list).Run()
		names = collect(list, func(p *model.RemoteAgent) string { return p.Name })
	case refClient:
		var list model.Clients
		err = h.db.Select(&list).Run()
		names = collect(list, func(c *model.Client) string { return c.Name })
		protocols = map[string]string{}

		for _, c := range list {
			protocols[c.Name] = protoform.ProtocolOf(c.Protocol).Label
		}
	case refServer:
		var list model.LocalAgents
		err = h.db.Select(&list).Run()
		names = collect(list, func(s *model.LocalAgent) string { return s.Name })
	case refRule:
		var list model.Rules
		err = h.db.Select(&list).Run()
		names = collect(list, func(r *model.Rule) string { return r.Name })
	case refNone:
	}

	if err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	slices.Sort(names)
	names = slices.Compact(names) // a rule name is shared by its two directions

	group := taskGroup{}
	for _, name := range names {
		option := taskOption{Value: name, Label: name}
		if protocol := protocols[name]; protocol != "" {
			option.Label += " · " + protocol // a client with its protocol, as the page that adds a transfer says it
		}

		group.Options = append(group.Options, option)
	}

	return []taskGroup{group}, nil
}

// ruleNames are the names of the rules by direction ("send", "receive"),
// sorted: the exchange of a TRANSFER task offers those of its direction.
func (h *Handlers) ruleNames() (map[string][]string, error) {
	var list model.Rules
	if err := h.db.Select(&list).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	out := map[string][]string{}
	for _, rule := range list {
		dir := directionOf(rule.IsSend)
		out[dir] = append(out[dir], rule.Name)
	}

	for dir := range out {
		slices.Sort(out[dir])
	}

	return out, nil
}

func collect[T any](list []T, name func(T) string) []string {
	out := make([]string, 0, len(list))
	for _, item := range list {
		out = append(out, name(item))
	}

	return out
}

// keyNames: the cryptographic keys, each with its type.
func (h *Handlers) keyNames() ([]taskGroup, error) {
	var list model.CryptoKeys
	if err := h.db.Select(&list).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	group := taskGroup{}
	for _, k := range list {
		group.Options = append(group.Options, taskOption{Value: k.Name, Label: k.Name + " (" + k.Type + ")"})
	}

	slices.SortFunc(group.Options, func(a, b taskOption) int { return strings.Compare(a.Value, b.Value) })

	return []taskGroup{group}, nil
}

// accountNames: the logins of the accounts, grouped by their server or their
// partner.
func (h *Handlers) accountNames(kind refKind) ([]taskGroup, error) {
	t, err := h.targets()
	if err != nil {
		return nil, err
	}

	byOwner := map[string][]taskOption{}

	if kind == refLocalAccount {
		for _, a := range t.locals {
			owner := t.servers[a.LocalAgentID].Name
			byOwner[owner] = append(byOwner[owner], taskOption{Value: a.Login, Label: a.Login})
		}
	} else {
		for _, a := range t.remotes {
			owner := t.partners[a.RemoteAgentID].Name
			byOwner[owner] = append(byOwner[owner], taskOption{Value: a.Login, Label: a.Login})
		}
	}

	owners := make([]string, 0, len(byOwner))
	for owner := range byOwner {
		owners = append(owners, owner)
	}

	slices.Sort(owners)

	groups := make([]taskGroup, 0, len(owners))
	for _, owner := range owners {
		opts := byOwner[owner]
		slices.SortFunc(opts, func(a, b taskOption) int { return strings.Compare(a.Value, b.Value) })
		groups = append(groups, taskGroup{Label: owner, Options: opts})
	}

	return groups, nil
}
