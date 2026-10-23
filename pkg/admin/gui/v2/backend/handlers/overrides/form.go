package overrides

import (
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

// Names of the fields of the form: the configured address (the key of the
// rule, on creation only) and the address on this node.
const (
	fieldConfigured  = "configured"
	fieldReplacement = "replacement"

	maxPort        = 65535
	errorKeyPrefix = "overrides.form.error."
)

// hostPattern is a host name as a configuration gives it: labels of letters,
// digits, hyphens and underscores, separated by dots. An address that holds
// anything else (a space, the "->" separator of the file of the node) would
// make the file of the node unreadable at the next start: the screen refuses
// it, before the product.
//
//nolint:gochecknoglobals // a compiled pattern
var hostPattern = regexp.MustCompile(`^` + hostLabel + `(\.` + hostLabel + `)*\.?$`)

// hostLabel is a label of a host name: up to 63 letters, digits, hyphens and
// underscores, neither starting nor ending with a hyphen.
const hostLabel = `[A-Za-z0-9_]([A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?`

type field = protoform.Field

// checkAddress reads an address as the product does (a host alone, or a host
// and its port, an IPv6 address in brackets) and checks its host and its
// port; it returns whether it carries a port, or the key of its error.
func checkAddress(addr string) (withPort bool, errKey string) {
	if addr == "" {
		return false, "required"
	}

	withPort, ok := hasPort(addr)
	if !ok {
		return false, "address"
	}

	host := addr
	if withPort {
		h, port, err := net.SplitHostPort(addr)
		if err != nil {
			return withPort, "address"
		}

		host = h
		if n, convErr := strconv.Atoi(port); convErr != nil || n < 1 || n > maxPort {
			return withPort, "port"
		}
	}

	// Brackets hold an IPv6 address, and only one.
	trimmed := strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	if strings.Contains(addr, "[") {
		if ip := net.ParseIP(trimmed); ip == nil || ip.To4() != nil {
			return withPort, "address"
		}

		return withPort, ""
	}

	if net.ParseIP(host) != nil && !strings.Contains(host, ":") {
		return withPort, ""
	}

	if !hostPattern.MatchString(host) {
		return withPort, "address"
	}

	return withPort, ""
}

// formData is the form of a rule: the configured address and the address on
// this node on creation; the address on this node alone on edition (the
// configured address is the key of the rule).
type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	Refused   string // a refusal of the product no field carries

	Configured  field
	Replacement field
	Uses        []use
	AdminWarn   bool
}

func newForm(translate gui.Translator, rl *rule, carried string) *formData {
	form := &formData{
		Title:     translate("overrides.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
	}

	if rl != nil {
		form.Title = translate("overrides.edit.title", "name", rl.Target)
		form.Action = ruleURL("edit", rl.Target, carried)
		form.Edit = true
		form.Configured = field{Value: rl.Target}
		form.Replacement = field{Value: rl.Real}
		form.Uses = usesOf(translate, rl)
		form.AdminWarn = rl.uses.admin
	}

	return form
}

// validate checks both addresses and that they have the same form, as the
// product asks; on creation, that the configured address is not replaced
// already (the product would overwrite its rule without a word).
func (f *formData) validate(translate gui.Translator, overrides *conf.ConfigOverride) bool {
	configuredPort, configuredErr := checkAddress(f.Configured.Value)
	replacementPort, replacementErr := checkAddress(f.Replacement.Value)

	if configuredErr != "" {
		f.Configured.Error = translate(errorKeyPrefix + configuredErr)
	}

	switch {
	case replacementErr != "":
		f.Replacement.Error = translate(errorKeyPrefix + replacementErr)
	case configuredErr == "" && configuredPort != replacementPort:
		f.Replacement.Error = translate(errorKeyPrefix + "form")
	}

	if !f.Edit && configuredErr == "" && overrides.GetIndirection(f.Configured.Value) != "" {
		f.Configured.Error = translate(errorKeyPrefix + "exists")
	}

	f.HasErrors = f.Configured.Error != "" || f.Replacement.Error != ""

	return !f.HasErrors
}

// refuse marks the form with the refusal of the product, in the banner with
// its cause: the form checks the two addresses and their form itself; the
// product refuses besides an address it cannot resolve (#731).
func (f *formData) refuse(err error) {
	f.Refused = err.Error()
	f.HasErrors = true
}

func (h *Handlers) render(w http.ResponseWriter, r *http.Request, form *formData, status int) {
	translate := h.shell.Translator(r)
	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, ret(r), status), formView(form))
}

// NewForm serves GET /webui/overrides/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	if overrides, _ := h.node(); overrides == nil {
		h.notFound(w, r)

		return
	}

	h.render(w, r, newForm(h.shell.Translator(r), nil, ret(r)), 0)
}

// Create serves POST /webui/overrides/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	overrides, _ := h.node()
	if overrides == nil {
		h.notFound(w, r)

		return
	}

	h.save(w, r, overrides, nil)
}

// EditForm serves GET /webui/overrides/edit?target=….
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	_, rl, ok := h.target(w, r)
	if !ok {
		return
	}

	h.concerns(rl)
	h.render(w, r, newForm(h.shell.Translator(r), rl, ret(r)), 0)
}

// Update serves POST /webui/overrides/edit?target=….
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	overrides, rl, ok := h.target(w, r)
	if !ok {
		return
	}

	h.concerns(rl)
	h.save(w, r, overrides, rl)
}

// save reads, validates and writes the posted rule into the file of the
// node: a new one when rl is nil, the address on this node of the given one
// otherwise. A refused form is rendered again with its errors (422); a saved
// one redirects to the list with a notification that says when it applies.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, overrides *conf.ConfigOverride, rl *rule) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := newForm(translate, rl, carried)
	form.Replacement = field{Value: strings.TrimSpace(r.PostFormValue(fieldReplacement))}

	if rl == nil {
		form.Configured = field{Value: strings.TrimSpace(r.PostFormValue(fieldConfigured))}
	}

	if form.validate(translate, overrides) {
		err := overrides.AddIndirection(form.Configured.Value, form.Replacement.Value)
		if err == nil {
			done := doneCreated
			if rl != nil {
				done = doneUpdated
			}

			listSpec.RedirectList(w, r, carried, url.Values{doneParam: {done}, nameParam: {form.Configured.Value}})

			return
		}

		h.logger.Warningf("failed to save the address indirection %q: %v", form.Configured.Value, err)
		form.refuse(err)
	}

	h.render(w, r, form, http.StatusUnprocessableEntity)
}
