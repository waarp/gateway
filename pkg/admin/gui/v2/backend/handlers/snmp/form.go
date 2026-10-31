package snmp

import (
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	gwsnmp "code.waarp.fr/apps/gateway/gateway/pkg/snmp"
)

// Names of the fields of the forms of a monitor and of the agent.
const (
	fieldName          = "name"
	fieldVersion       = "version"
	fieldAddress       = "address"
	fieldCommunity     = "community"
	fieldSecurity      = "security"
	fieldUser          = "username"
	fieldEngine        = "engine"
	fieldAuthProtocol  = "auth_protocol"
	fieldAuthPass      = "auth_pass"
	fieldPrivProtocol  = "priv_protocol"
	fieldPrivPass      = "priv_pass"
	fieldContextName   = "context_name"
	fieldContextEngine = "context_engine"
	fieldV3Only        = "v3_only"

	// The causes the model gives when it refuses a monitor or the agent.
	causeTaken       = "already exists"
	causeAddress     = "invalid UDP address"
	causeUser        = "missing authentication username"
	causeEngine      = "missing authentication engine ID"
	causeAuthPass    = "missing authentication passphrase"
	causePrivPass    = "missing privacy passphrase"
	causeAuthProto   = "invalid authentication protocol"
	causePrivProto   = "invalid privacy protocol"
	causeVersion     = "SNMP version"
	causeSNMPv1      = "SNMPv1"
	maxPort          = 65535
	errorKeyPrefix   = "snmp.form.error."
	requiredErrorKey = errorKeyPrefix + "required"
)

// The text fields of the forms, trimmed; the secrets are kept as typed.
//
//nolint:gochecknoglobals // the fields of the forms
var (
	textFields = []string{
		fieldName, fieldVersion, fieldAddress, fieldSecurity, fieldUser, fieldEngine,
		fieldAuthProtocol, fieldPrivProtocol, fieldContextName, fieldContextEngine,
	}
	secretFields = []string{fieldCommunity, fieldAuthPass, fieldPrivPass}

	// An authentication engine ID as the product reads it: hexadecimal
	// prefixed with 0x, or decimal; any other text would be sent as is.
	engineIDPattern = regexp.MustCompile(`^(0[xX][0-9A-Fa-f]+|\d+)$`)
)

// fields are the posted values of a form, with their errors.
type fields map[string]field

func readFields(values url.Values) fields {
	f := fields{}
	for _, id := range textFields {
		f[id] = field{Value: strings.TrimSpace(values.Get(id))}
	}

	for _, id := range secretFields {
		f[id] = field{Value: values.Get(id)}
	}

	return f
}

func (f fields) value(id string) string { return f[id].Value }

// fail marks a field with its error.
func (f fields) fail(id, msg string) {
	v := f[id]
	v.Error = msg
	f[id] = v
}

func (f fields) hasErrors() bool {
	for _, v := range f {
		if v.Error != "" {
			return true
		}
	}

	return false
}

// checkAddress checks the form "host:port" the product asks; the model then
// resolves the host.
func checkAddress(translate gui.Translator, f fields) {
	addr := f.value(fieldAddress)
	if addr == "" {
		f.fail(fieldAddress, translate(requiredErrorKey))

		return
	}

	host, port, err := net.SplitHostPort(addr)
	if n, convErr := strconv.Atoi(port); err != nil || host == "" || convErr != nil || n < 1 || n > maxPort {
		f.fail(fieldAddress, translate(errorKeyPrefix+"address_form"))
	}
}

// protocolChoices: the protocols of a list, the historic ones said so.
func protocolChoices(translate gui.Translator, list []string, selected string) []choice {
	out := make([]choice, 0, len(list))
	for _, p := range list {
		out = append(out, choice{Value: p, Label: protocolLabel(translate, p), Selected: p == selected})
	}

	return out
}

// -------------------------------------------------------------- monitor ----

// monitorForm is the form of a monitor: SNMPv2c (a community) or SNMPv3
// (a security level, a user, and the protocols and passphrases the level
// asks); the secrets are never rendered back, an empty one keeps the stored
// one.
type monitorForm struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	Refused   string // a refusal of the model no field carries

	Fields        fields
	VersionErr    string
	Versions      []choice
	Levels        []choice
	AuthProtocols []choice
	PrivProtocols []choice
	CommunitySet  bool
	AuthSet       bool
	PrivSet       bool
	ContextOpen   bool
	EngineReq     bool // the stored monitor uses informs: the product exempts it

	// The monitor before the edition, nil on creation: its secrets, its
	// spelling of the version and its notification type are kept.
	stored *gwsnmp.MonitorConfig
}

func newMonitorForm(translate gui.Translator, monitor *gwsnmp.MonitorConfig, carried string) *monitorForm {
	form := &monitorForm{
		Title:     translate("snmp.new.title"),
		Action:    withRet(common.WebuiURL(monitorsPath+"/new"), carried),
		CancelURL: listURL(carried),
		Fields:    fields{fieldSecurity: {Value: gwsnmp.V3SecurityNoAuthNoPriv}},
	}

	if monitor == nil {
		return form
	}

	form.Title = translate("snmp.edit.title", "name", monitor.Name)
	form.Action = withRet(monitorURL(monitor.ID, "/edit"), carried)
	form.CancelURL = withRet(monitorURL(monitor.ID, ""), carried)
	form.Edit = true
	form.stored = monitor

	version := ""

	switch {
	case isV2(monitor.Version):
		version = versionV2
	case isV3(monitor.Version):
		version = versionV3
	}

	form.Fields = fields{
		fieldName:          {Value: monitor.Name},
		fieldVersion:       {Value: version},
		fieldAddress:       {Value: monitor.UDPAddress},
		fieldSecurity:      {Value: levelOf(monitor)},
		fieldUser:          {Value: monitor.AuthUsername},
		fieldEngine:        {Value: monitor.AuthEngineID},
		fieldAuthProtocol:  {Value: monitor.AuthProtocol},
		fieldPrivProtocol:  {Value: monitor.PrivProtocol},
		fieldContextName:   {Value: monitor.ContextName},
		fieldContextEngine: {Value: monitor.ContextEngineID},
	}

	return form
}

// read takes the posted values; an unknown version or level reads as none.
func (f *monitorForm) read(values url.Values) {
	f.Fields = readFields(values)

	if v := f.Fields.value(fieldVersion); v != versionV2 && v != versionV3 {
		f.Fields[fieldVersion] = field{}
	}
}

// storedSecret tells whether the edited monitor holds a value of a secret.
func (f *monitorForm) storedSecret(id string) bool {
	if f.stored == nil {
		return false
	}

	switch id {
	case fieldAuthPass:
		return f.stored.AuthPassphrase != ""
	case fieldPrivPass:
		return f.stored.PrivPassphrase != ""
	default:
		return f.stored.Community != ""
	}
}

// validate checks what the screen can check before the model: the name,
// the version, the form of the address and, in SNMPv3, the user, the
// authentication engine ID (unless the stored monitor uses informs, which
// the product exempts) and the protocols and passphrases the level asks.
func (f *monitorForm) validate(translate gui.Translator) bool {
	if f.Fields.value(fieldName) == "" {
		f.Fields.fail(fieldName, translate(requiredErrorKey))
	}

	checkAddress(translate, f.Fields)

	switch f.Fields.value(fieldVersion) {
	case "":
		f.VersionErr = translate(errorKeyPrefix + "version")
	case versionV3:
		f.validateV3(translate)
	}

	f.HasErrors = f.VersionErr != "" || f.Fields.hasErrors()

	return !f.HasErrors
}

func (f *monitorForm) validateV3(translate gui.Translator) {
	level := f.Fields.value(fieldSecurity)
	if !slices.Contains(securityLevels, level) {
		f.Fields.fail(fieldSecurity, translate(errorKeyPrefix+"security"))

		return
	}

	if f.Fields.value(fieldUser) == "" {
		f.Fields.fail(fieldUser, translate(errorKeyPrefix+"user"))
	}

	if !usesAuth(level) {
		return
	}

	switch engine := f.Fields.value(fieldEngine); {
	case engine == "" && (f.stored == nil || !f.stored.UseInforms):
		f.Fields.fail(fieldEngine, translate(requiredErrorKey))
	case engine != "" && !engineIDPattern.MatchString(engine):
		f.Fields.fail(fieldEngine, translate(errorKeyPrefix+"engine"))
	}

	f.checkProtocol(translate, fieldAuthProtocol, fieldAuthPass, authProtocols)

	if usesPriv(level) {
		f.checkProtocol(translate, fieldPrivProtocol, fieldPrivPass, privProtocols)
	}
}

// checkProtocol: a protocol of the list is chosen, and its passphrase is
// given or already stored.
func (f *monitorForm) checkProtocol(translate gui.Translator, protoID, passID string, list []string) {
	if !slices.Contains(list, f.Fields.value(protoID)) {
		f.Fields.fail(protoID, translate(errorKeyPrefix+"protocol"))
	}

	if f.Fields.value(passID) == "" && !f.storedSecret(passID) {
		f.Fields.fail(passID, translate(requiredErrorKey))
	}
}

// apply writes the validated form into the monitor (a copy of the stored
// one on edition): an empty secret keeps the stored one; the product clears
// what the version or the level does not use.
func (f *monitorForm) apply(monitor *gwsnmp.MonitorConfig) {
	version := f.Fields.value(fieldVersion)

	monitor.Name = f.Fields.value(fieldName)
	monitor.UDPAddress = f.Fields.value(fieldAddress)

	// The stored spelling of the version is kept while it does not change.
	if versionLabel(monitor.Version) != versionLabel(version) {
		monitor.Version = version
	}

	keep := func(dst *string, id string) {
		if v := f.Fields.value(id); v != "" {
			*dst = v
		}
	}

	if version == versionV2 {
		keep(&monitor.Community, fieldCommunity)

		return
	}

	level := f.Fields.value(fieldSecurity)
	monitor.SNMPv3Security = level
	monitor.AuthUsername = f.Fields.value(fieldUser)
	monitor.ContextName = f.Fields.value(fieldContextName)
	monitor.ContextEngineID = f.Fields.value(fieldContextEngine)

	if usesAuth(level) {
		monitor.AuthEngineID = f.Fields.value(fieldEngine)
		monitor.AuthProtocol = f.Fields.value(fieldAuthProtocol)
		keep(&monitor.AuthPassphrase, fieldAuthPass)
	}

	if usesPriv(level) {
		monitor.PrivProtocol = f.Fields.value(fieldPrivProtocol)
		keep(&monitor.PrivPassphrase, fieldPrivPass)
	}
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *monitorForm) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, causeTaken):
		f.Fields.fail(fieldName, translate(errorKeyPrefix+"name_taken"))
	case strings.Contains(msg, causeAddress):
		f.Fields.fail(fieldAddress, translate(errorKeyPrefix+"address", "cause", msg))
	case strings.Contains(msg, causeUser):
		f.Fields.fail(fieldUser, translate(errorKeyPrefix+"user"))
	case strings.Contains(msg, causeEngine):
		f.Fields.fail(fieldEngine, translate(requiredErrorKey))
	case strings.Contains(msg, causeAuthPass):
		f.Fields.fail(fieldAuthPass, translate(requiredErrorKey))
	case strings.Contains(msg, causePrivPass):
		f.Fields.fail(fieldPrivPass, translate(requiredErrorKey))
	case strings.Contains(msg, causeAuthProto):
		f.Fields.fail(fieldAuthProtocol, translate(errorKeyPrefix+"protocol"))
	case strings.Contains(msg, causePrivProto):
		f.Fields.fail(fieldPrivProtocol, translate(errorKeyPrefix+"protocol"))
	case strings.Contains(msg, causeVersion), strings.Contains(msg, causeSNMPv1):
		f.VersionErr = translate(errorKeyPrefix + "version")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// prepare builds what the templates render: the choices with the current
// values, what secrets are stored, whether the context card opens.
func (f *monitorForm) prepare(translate gui.Translator) {
	version := f.Fields.value(fieldVersion)
	f.Versions = []choice{
		{Value: versionV2, Label: versionV2c, Selected: version == versionV2},
		{Value: versionV3, Label: versionV3, Selected: version == versionV3},
	}

	level := f.Fields.value(fieldSecurity)
	f.Levels = make([]choice, 0, len(securityLevels))

	for _, l := range securityLevels {
		f.Levels = append(f.Levels, choice{Value: l, Label: translate(securityKeyPrefix + l), Selected: l == level})
	}

	f.AuthProtocols = protocolChoices(translate, authProtocols, f.Fields.value(fieldAuthProtocol))
	f.PrivProtocols = protocolChoices(translate, privProtocols, f.Fields.value(fieldPrivProtocol))
	f.CommunitySet = f.storedSecret(fieldCommunity)
	f.AuthSet = f.storedSecret(fieldAuthPass)
	f.PrivSet = f.storedSecret(fieldPrivPass)
	f.ContextOpen = f.Fields.value(fieldContextName) != "" || f.Fields.value(fieldContextEngine) != ""
	f.EngineReq = f.stored == nil || !f.stored.UseInforms
}

func (h *Handlers) renderMonitor(w http.ResponseWriter, r *http.Request, form *monitorForm, status int) {
	translate := h.shell.Translator(r)
	form.prepare(translate)
	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, ret(r), status), monitorFormView(form))
}

// NewMonitorForm serves GET /webui/snmp/monitors/new.
func (h *Handlers) NewMonitorForm(w http.ResponseWriter, r *http.Request) {
	h.renderMonitor(w, r, newMonitorForm(h.shell.Translator(r), nil, ret(r)), 0)
}

// CreateMonitor serves POST /webui/snmp/monitors/new.
func (h *Handlers) CreateMonitor(w http.ResponseWriter, r *http.Request) {
	h.saveMonitor(w, r, nil)
}

// EditMonitorForm serves GET /webui/snmp/monitors/{id}/edit.
func (h *Handlers) EditMonitorForm(w http.ResponseWriter, r *http.Request) {
	m, ok := h.monitor(w, r)
	if !ok {
		return
	}

	h.renderMonitor(w, r, newMonitorForm(h.shell.Translator(r), m, ret(r)), 0)
}

// UpdateMonitor serves POST /webui/snmp/monitors/{id}/edit.
func (h *Handlers) UpdateMonitor(w http.ResponseWriter, r *http.Request) {
	m, ok := h.monitor(w, r)
	if !ok {
		return
	}

	h.saveMonitor(w, r, m)
}

// saveMonitor reads, validates and stores the posted form: a new monitor
// when stored is nil, the given one otherwise. A refused form, by the screen
// or by the model, is rendered again with its errors (422); a saved one
// makes the SNMP service read the monitors again and redirects with a
// notification: a new monitor shows on the screen, an edited one comes back
// to its sheet.
func (h *Handlers) saveMonitor(w http.ResponseWriter, r *http.Request, stored *gwsnmp.MonitorConfig) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := newMonitorForm(translate, stored, carried)
	form.read(r.PostForm)

	if form.validate(translate) {
		// A copy: a refused edition leaves the monitor as it was read.
		monitor := &gwsnmp.MonitorConfig{}
		if stored != nil {
			stored := *stored
			monitor = &stored
		}

		form.apply(monitor)

		var err error
		if monitor.ID == 0 {
			err = h.db.Insert(monitor).Run()
		} else {
			err = h.db.Update(monitor).Run()
		}

		switch {
		case err == nil:
			target, done := listURL(carried), doneCreated
			if stored != nil {
				target, done = withRet(monitorURL(monitor.ID, ""), carried), doneUpdated
			}

			h.reloaded(w, r, target, done, monitor.Name, h.reloadMonitors())

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the SNMP monitor %q: %v", monitor.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	h.renderMonitor(w, r, form, http.StatusUnprocessableEntity)
}

// ---------------------------------------------------------------- agent ----

// agentForm is the form of the agent: where it listens, its community, the
// SNMPv3 user and its protocols; the secrets are never rendered back, an
// empty one keeps the stored one.
type agentForm struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	Refused   string

	Fields        fields
	V3Only        bool
	AuthProtocols []choice
	PrivProtocols []choice
	CommunitySet  bool
	AuthSet       bool
	PrivSet       bool

	stored *gwsnmp.ServerConfig
}

func newAgentForm(translate gui.Translator, conf *gwsnmp.ServerConfig, carried string) *agentForm {
	form := &agentForm{
		Title:     translate("snmp.agent.form.new_title"),
		Action:    withRet(common.WebuiURL(agentPath+"/edit"), carried),
		CancelURL: listURL(carried),
		Fields:    fields{},
	}

	if conf == nil {
		return form
	}

	form.Title = translate("snmp.agent.form.edit_title")
	form.Edit = true
	form.stored = conf
	form.V3Only = conf.SNMPv3Only
	form.Fields = fields{
		fieldAddress:      {Value: conf.LocalUDPAddress},
		fieldUser:         {Value: conf.SNMPv3Username},
		fieldAuthProtocol: {Value: conf.SNMPv3AuthProtocol},
		fieldPrivProtocol: {Value: conf.SNMPv3PrivProtocol},
	}

	return form
}

func (f *agentForm) read(values url.Values) {
	f.Fields = readFields(values)
	f.V3Only = values.Get(fieldV3Only) != ""
}

func (f *agentForm) storedSecret(id string) bool {
	if f.stored == nil {
		return false
	}

	switch id {
	case fieldAuthPass:
		return f.stored.SNMPv3AuthPassphrase != ""
	case fieldPrivPass:
		return f.stored.SNMPv3PrivPassphrase != ""
	default:
		return f.stored.Community != ""
	}
}

// validate checks the form of the address, the protocols (none, or one of
// the list; privacy asks authentication, as SNMPv3 does), the user an
// SNMPv3 setting needs, and the passphrase of each chosen protocol, given
// or already stored.
func (f *agentForm) validate(translate gui.Translator) bool {
	checkAddress(translate, f.Fields)

	auth, priv := f.Fields.value(fieldAuthProtocol), f.Fields.value(fieldPrivProtocol)

	switch {
	case auth != "" && !slices.Contains(authProtocols, auth):
		f.Fields.fail(fieldAuthProtocol, translate(errorKeyPrefix+"protocol"))
	case auth != "" && f.Fields.value(fieldAuthPass) == "" && !f.storedSecret(fieldAuthPass):
		f.Fields.fail(fieldAuthPass, translate(requiredErrorKey))
	}

	switch {
	case priv != "" && !slices.Contains(privProtocols, priv):
		f.Fields.fail(fieldPrivProtocol, translate(errorKeyPrefix+"protocol"))
	case priv != "" && auth == "":
		f.Fields.fail(fieldPrivProtocol, translate(errorKeyPrefix+"priv_without_auth"))
	case priv != "" && f.Fields.value(fieldPrivPass) == "" && !f.storedSecret(fieldPrivPass):
		f.Fields.fail(fieldPrivPass, translate(requiredErrorKey))
	}

	if (f.V3Only || auth != "") && f.Fields.value(fieldUser) == "" {
		f.Fields.fail(fieldUser, translate(errorKeyPrefix+"user"))
	}

	f.HasErrors = f.Fields.hasErrors()

	return !f.HasErrors
}

// apply writes the validated form into the configuration (a copy of the
// stored one on edition): an empty secret keeps the stored one; the product
// clears the passphrase of a protocol left empty.
func (f *agentForm) apply(c *gwsnmp.ServerConfig) {
	c.LocalUDPAddress = f.Fields.value(fieldAddress)
	c.SNMPv3Only = f.V3Only
	c.SNMPv3Username = f.Fields.value(fieldUser)
	c.SNMPv3AuthProtocol = f.Fields.value(fieldAuthProtocol)
	c.SNMPv3PrivProtocol = f.Fields.value(fieldPrivProtocol)

	for id, dst := range map[string]*string{
		fieldCommunity: &c.Community,
		fieldAuthPass:  &c.SNMPv3AuthPassphrase,
		fieldPrivPass:  &c.SNMPv3PrivPassphrase,
	} {
		if v := f.Fields.value(id); v != "" {
			*dst = v
		}
	}
}

func (f *agentForm) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, causeAddress):
		f.Fields.fail(fieldAddress, translate(errorKeyPrefix+"address", "cause", msg))
	case strings.Contains(msg, causeAuthProto):
		f.Fields.fail(fieldAuthProtocol, translate(errorKeyPrefix+"protocol"))
	case strings.Contains(msg, causePrivProto):
		f.Fields.fail(fieldPrivProtocol, translate(errorKeyPrefix+"protocol"))
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

func (f *agentForm) prepare(translate gui.Translator) {
	f.AuthProtocols = protocolChoices(translate, authProtocols, f.Fields.value(fieldAuthProtocol))
	f.PrivProtocols = protocolChoices(translate, privProtocols, f.Fields.value(fieldPrivProtocol))
	f.CommunitySet = f.storedSecret(fieldCommunity)
	f.AuthSet = f.storedSecret(fieldAuthPass)
	f.PrivSet = f.storedSecret(fieldPrivPass)
}

func (h *Handlers) renderAgent(w http.ResponseWriter, r *http.Request, form *agentForm, status int) {
	translate := h.shell.Translator(r)
	form.prepare(translate)
	h.shell.Render(w, h.logger, r, h.screen(translate, form.Title, ret(r), status), agentFormView(form))
}

// AgentForm serves GET /webui/snmp/agent/edit: the configuration of the
// agent, new or in place.
func (h *Handlers) AgentForm(w http.ResponseWriter, r *http.Request) {
	conf, err := h.agent()
	if err != nil {
		h.logger.Errorf("failed to read the SNMP agent configuration: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.renderAgent(w, r, newAgentForm(h.shell.Translator(r), conf, ret(r)), 0)
}

// SaveAgent serves POST /webui/snmp/agent/edit: the configuration is
// created or replaced, then the SNMP service restarts on it, as with the
// REST API.
func (h *Handlers) SaveAgent(w http.ResponseWriter, r *http.Request) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	stored, err := h.agent()
	if err == nil {
		err = r.ParseForm()
	}

	if err != nil {
		h.logger.Errorf("failed to read the SNMP agent configuration: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := newAgentForm(translate, stored, carried)
	form.read(r.PostForm)

	if form.validate(translate) {
		conf := &gwsnmp.ServerConfig{}
		if stored != nil {
			copied := *stored
			conf = &copied
		}

		form.apply(conf)

		if stored == nil {
			err = h.db.Insert(conf).Run()
		} else {
			err = h.db.Update(conf).Run()
		}

		switch {
		case err == nil:
			h.reloaded(w, r, listURL(carried), doneAgentSaved, "", h.reloadAgent(r.Context()))

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the SNMP agent configuration: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	h.renderAgent(w, r, form, http.StatusUnprocessableEntity)
}

// v3Touched says whether the SNMPv3 card of the agent form has something to
// show, a user, a protocol, a stored passphrase or an error: the card opens
// then, folded otherwise (#810).
func (f *agentForm) v3Touched() bool {
	for _, id := range []string{fieldUser, fieldAuthProtocol, fieldAuthPass, fieldPrivProtocol, fieldPrivPass} {
		if f.Fields[id].Value != "" || f.Fields[id].Error != "" {
			return true
		}
	}

	return f.AuthSet || f.PrivSet
}
