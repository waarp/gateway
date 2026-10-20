package servers

import (
	"net/http"
	"net/url"
	"strings"

	"code.waarp.fr/lib/gui"
	"golang.org/x/exp/constraints"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

// Names of the form fields. The blocks of every protocol are posted together
// without JavaScript, so each block has its own names; the server reads the
// block of the chosen protocol only.
const (
	fieldName          = "name"
	fieldProtocol      = "protocol"
	fieldNoAutoStart   = "no_autostart"
	fieldHost          = "host"
	fieldPort          = "port"
	fieldRootDir       = "root_dir"
	fieldReceiveDir    = "receive_dir"
	fieldTmpReceiveDir = "tmp_receive_dir"
	fieldSendDir       = "send_dir"

	fieldR66Login          = "r66_login"
	fieldR66Block          = "r66_block"
	fieldR66NoFinalHash    = "r66_no_final_hash"
	fieldR66CheckBlockHash = "r66_check_block_hash"

	fieldFTPDisablePassive = "ftp_disable_passive"
	fieldFTPDisableActive  = "ftp_disable_active"
	fieldFTPPassiveMin     = "ftp_passive_min"
	fieldFTPPassiveMax     = "ftp_passive_max"
	fieldFTPSRequirement   = "ftps_tls_requirement"

	fieldPesitDisableRestart     = "pesit_disable_restart"
	fieldPesitDisableCheckpoints = "pesit_disable_checkpoints"
	fieldPesitCheckpointSize     = "pesit_checkpoint_size"
	fieldPesitCheckpointWindow   = "pesit_checkpoint_window"
	fieldPesitMaxMessage         = "pesit_max_message"
	fieldPesitMode               = "pesit_mode"
	fieldPesitDisablePreConn     = "pesit_disable_preconnection"

	fieldSFTPKex     = "sftp_kex"
	fieldSFTPCiphers = "sftp_ciphers"
	fieldSFTPMACs    = "sftp_macs"

	fieldAS2MaxFileSize = "as2_max_file_size"
	fieldAS2MDNAlgo     = "as2_mdn_algo"

	// Suffixes of the TLS fields, prefixed by the TLS prefix of the family
	// ("r66_min_tls", "ftps_tls_ciphers").
	suffixMinTLS     = "_min_tls"
	suffixTLSCiphers = "_tls_ciphers"
)

// tlsPrefixes gives the prefix of the TLS fields of each family.
//
//nolint:gochecknoglobals // reference data of the form
var tlsPrefixes = map[string]string{
	familyR66: "r66", familyFTP: "ftps", familyPeSIT: "pesit",
	familyHTTP: "https", familyWebDAV: "webdav", familyAS2: "as2",
}

// field is a text or number field of the form: its value, kept when the form
// is refused, and the message of its error.
type field struct {
	Value string
	Error string
}

// formChoices are the option lists of the selects, with the current values
// selected; the TLS lists are keyed by the TLS prefix of the family.
type formChoices struct {
	TLSVersions     map[string][]choice
	TLSCiphers      map[string][]choice
	TLSRequirements []choice
	PesitModes      []choice
	MDNAlgos        []choice
	KeyExchanges    []choice
	Ciphers         []choice
	MACs            []choice
}

type formData struct {
	Title     string
	Action    string
	CancelURL string
	Edit      bool
	HasErrors bool
	// Refused is the cause given by the model when it refuses the server for
	// another reason than a field shown with its own error.
	Refused string

	Fields        map[string]field // text and number fields, by name
	Checks        map[string]bool  // checkboxes, by name
	Protocol      string
	ProtocolError string
	Protocols     []protocol
	Choices       formChoices

	// The lists chosen in the dual lists and the selects, by name, and the
	// port once validated.
	lists   map[string][]string
	selects map[string]string
	port    uint16
}

// newForm builds the form of a new server, or the form of an existing one
// filled with its values.
func (h *Handlers) newForm(translate gui.Translator, agent *model.LocalAgent, carried string) *formData {
	form := &formData{
		Title:     translate("servers.new.title"),
		Action:    withRet(common.WebuiURL(pagePath+"/new"), carried),
		CancelURL: listURL(carried),
		Fields:    map[string]field{},
		Checks:    map[string]bool{},
		Protocols: protocolList(),
		lists:     map[string][]string{},
		selects: map[string]string{
			fieldFTPSRequirement: string(ftp.TLSOptional),
			fieldPesitMode:       pesit.CompatibilityModeStandard,
		},
	}

	for _, prefix := range tlsPrefixes {
		form.selects[prefix+suffixMinTLS] = protoutils.TLSv12
	}

	if agent != nil {
		form.Title = translate("servers.edit.title", "name", agent.Name)
		form.Action = withRet(serverURL(agent.ID, "/edit"), carried)
		form.CancelURL = withRet(serverURL(agent.ID, ""), carried)
		form.Edit = true
		form.fill(agent)
	}

	form.Choices = buildChoices(translate, form)

	return form
}

// fill reads the values of an existing server into the form.
func (f *formData) fill(agent *model.LocalAgent) {
	set := func(name, value string) { f.Fields[name] = field{Value: value} }

	set(fieldName, agent.Name)
	set(fieldHost, agent.Address.Host)
	set(fieldPort, cfgString(map[string]any{"proto": agent.Address.Port}, "proto"))
	set(fieldRootDir, agent.RootDir)
	set(fieldReceiveDir, agent.ReceiveDir)
	set(fieldTmpReceiveDir, agent.TmpReceiveDir)
	set(fieldSendDir, agent.SendDir)
	f.Protocol = agent.Protocol
	f.Checks[fieldNoAutoStart] = agent.Disabled

	proto := protocolOf(agent.Protocol)
	cfg := agent.ProtoConfig

	switch proto.Family {
	case familyR66:
		set(fieldR66Login, cfgString(cfg, cfgServerLogin))
		set(fieldR66Block, cfgString(cfg, cfgBlockSize))
		f.Checks[fieldR66NoFinalHash] = cfgBool(cfg, cfgNoFinalHash)
		f.Checks[fieldR66CheckBlockHash] = cfgBool(cfg, cfgCheckBlockHash)
	case familyFTP:
		f.Checks[fieldFTPDisablePassive] = cfgBool(cfg, cfgDisablePassiveMode)
		f.Checks[fieldFTPDisableActive] = cfgBool(cfg, cfgDisableActiveMode)
		set(fieldFTPPassiveMin, cfgString(cfg, cfgPassiveModeMinPort))
		set(fieldFTPPassiveMax, cfgString(cfg, cfgPassiveModeMaxPort))

		if v := cfgString(cfg, cfgTLSRequirement); v != "" {
			f.selects[fieldFTPSRequirement] = v
		}
	case familyPeSIT:
		f.Checks[fieldPesitDisableRestart] = cfgBool(cfg, cfgDisableRestart)
		f.Checks[fieldPesitDisableCheckpoints] = cfgBool(cfg, cfgDisableCheckpoints)
		set(fieldPesitCheckpointSize, cfgString(cfg, cfgCheckpointSize))
		set(fieldPesitCheckpointWindow, cfgString(cfg, cfgCheckpointWindow))
		set(fieldPesitMaxMessage, cfgString(cfg, cfgMaxMessageSize))
		f.Checks[fieldPesitDisablePreConn] = cfgBool(cfg, cfgDisablePreConnection)

		if v := cfgString(cfg, cfgCompatibilityMode); v != "" {
			f.selects[fieldPesitMode] = v
		}
	case familySFTP:
		f.lists[fieldSFTPKex] = cfgList(cfg, cfgKeyExchanges)
		f.lists[fieldSFTPCiphers] = cfgList(cfg, cfgCiphers)
		f.lists[fieldSFTPMACs] = cfgList(cfg, cfgMACs)
	case familyAS2:
		set(fieldAS2MaxFileSize, cfgString(cfg, cfgMaxFileSize))
		f.selects[fieldAS2MDNAlgo] = cfgString(cfg, cfgMDNSignatureAlgo)
	}

	if prefix, ok := tlsPrefixes[proto.Family]; ok && proto.TLS {
		if v := cfgString(cfg, cfgMinTLSVersion); v != "" {
			f.selects[prefix+suffixMinTLS] = v
		}

		f.lists[prefix+suffixTLSCiphers] = cfgList(cfg, cfgCipherSuites)
	}
}

// buildChoices builds the option lists of the selects and the dual lists,
// with the current values of the form selected.
func buildChoices(translate gui.Translator, f *formData) formChoices {
	c := formChoices{
		TLSVersions:     map[string][]choice{},
		TLSCiphers:      map[string][]choice{},
		TLSRequirements: tlsRequirementChoices(translate, f.selects[fieldFTPSRequirement]),
		PesitModes:      pesitModeChoices(translate, f.selects[fieldPesitMode]),
		MDNAlgos:        mdnAlgoChoices(translate, f.selects[fieldAS2MDNAlgo]),
		KeyExchanges:    sftpChoices(sftpKeyExchanges(), f.lists[fieldSFTPKex]),
		Ciphers:         sftpChoices(sftpCiphers(), f.lists[fieldSFTPCiphers]),
		MACs:            sftpChoices(sftpMACs(), f.lists[fieldSFTPMACs]),
	}

	for _, prefix := range tlsPrefixes {
		c.TLSVersions[prefix] = tlsVersionChoices(f.selects[prefix+suffixMinTLS])
		c.TLSCiphers[prefix] = tlsCipherChoices(translate, f.lists[prefix+suffixTLSCiphers])
	}

	return c
}

// read takes the posted values into the form: every text field, checkbox,
// select and list, whatever the protocol, so that a refused form comes back
// as the user filled it.
func (f *formData) read(values url.Values) {
	for _, name := range []string{
		fieldName, fieldHost, fieldPort, fieldRootDir, fieldReceiveDir, fieldTmpReceiveDir, fieldSendDir,
		fieldR66Login, fieldR66Block, fieldFTPPassiveMin, fieldFTPPassiveMax,
		fieldPesitCheckpointSize, fieldPesitCheckpointWindow, fieldPesitMaxMessage, fieldAS2MaxFileSize,
	} {
		f.Fields[name] = field{Value: strings.TrimSpace(values.Get(name))}
	}

	for _, name := range []string{
		fieldNoAutoStart, fieldR66NoFinalHash, fieldR66CheckBlockHash, fieldFTPDisablePassive, fieldFTPDisableActive,
		fieldPesitDisableRestart, fieldPesitDisableCheckpoints, fieldPesitDisablePreConn,
	} {
		f.Checks[name] = values.Get(name) != ""
	}

	f.Protocol = values.Get(fieldProtocol)

	for _, name := range []string{fieldFTPSRequirement, fieldPesitMode, fieldAS2MDNAlgo} {
		if values.Has(name) {
			f.selects[name] = values.Get(name)
		}
	}

	for _, name := range []string{fieldSFTPKex, fieldSFTPCiphers, fieldSFTPMACs} {
		f.lists[name] = nonEmpty(values[name])
	}

	for _, prefix := range tlsPrefixes {
		if values.Has(prefix + suffixMinTLS) {
			f.selects[prefix+suffixMinTLS] = values.Get(prefix + suffixMinTLS)
		}

		f.lists[prefix+suffixTLSCiphers] = nonEmpty(values[prefix+suffixTLSCiphers])
	}
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))

	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}

	return out
}

// validate checks the fields the screen owns (the model checks the rest):
// the required ones, the port, the protocol, the numbers of the options of
// the chosen protocol. It marks the errors on the form and returns whether
// it is valid.
func (f *formData) validate(translate gui.Translator) bool {
	fail := func(name, key string) {
		fld := f.Fields[name]
		fld.Error = translate(key)
		f.Fields[name] = fld
	}

	// The host is optional: empty, the server listens on every interface.
	if f.Fields[fieldName].Value == "" {
		fail(fieldName, "servers.form.error.required")
	}

	port, err := parsePort(f.Fields[fieldPort].Value)
	if err != nil {
		fail(fieldPort, "servers.form.error.port")
	}

	f.port = port

	if !isProtocol(f.Protocol) {
		f.Protocol = ""
		f.ProtocolError = translate("servers.form.protocol_error")
	}

	numbers := map[string]func(string) error{}

	switch protocolOf(f.Protocol).Family {
	case familyR66:
		numbers[fieldR66Block] = check(internal.ParseUint[uint32])
	case familyFTP:
		numbers[fieldFTPPassiveMin] = check(parsePort)
		numbers[fieldFTPPassiveMax] = check(parsePort)
	case familyPeSIT:
		numbers[fieldPesitCheckpointSize] = check(internal.ParseUint[uint32])
		numbers[fieldPesitCheckpointWindow] = check(internal.ParseUint[uint8])
		numbers[fieldPesitMaxMessage] = check(internal.ParseUint[uint16])
	case familyAS2:
		numbers[fieldAS2MaxFileSize] = check(internal.ParseInt[int64])
	}

	for name, parse := range numbers {
		if value := f.Fields[name].Value; value != "" && parse(value) != nil {
			fail(name, "servers.form.error.number")
		}
	}

	for _, fld := range f.Fields {
		if fld.Error != "" {
			f.HasErrors = true
		}
	}

	f.HasErrors = f.HasErrors || f.ProtocolError != ""

	return !f.HasErrors
}

// check adapts a typed parser to a validity check.
func check[T any](parse func(string) (T, error)) func(string) error {
	return func(s string) error {
		_, err := parse(s)

		return err
	}
}

func parsePort(s string) (uint16, error) {
	port, err := internal.ParseUint[uint16](s)
	if err != nil {
		return 0, err
	}

	if port == 0 {
		return 0, errPortOutOfRange
	}

	return port, nil
}

// apply writes the validated form into the server: its fields and the
// options of its protocol, with the keys the product reads.
func (f *formData) apply(agent *model.LocalAgent) {
	value := func(name string) string { return f.Fields[name].Value }

	agent.Name = value(fieldName)
	agent.Protocol = f.Protocol
	agent.Disabled = f.Checks[fieldNoAutoStart]
	agent.RootDir = value(fieldRootDir)
	agent.ReceiveDir = value(fieldReceiveDir)
	agent.TmpReceiveDir = value(fieldTmpReceiveDir)
	agent.SendDir = value(fieldSendDir)

	agent.Address = types.Addr(value(fieldHost), f.port)

	cfg := model.Map[any]{}
	proto := protocolOf(f.Protocol)

	switch proto.Family {
	case familyR66:
		setText(cfg, cfgServerLogin, value(fieldR66Login))
		setNumber(cfg, cfgBlockSize, value(fieldR66Block), internal.ParseUint[uint32])
		cfg[cfgNoFinalHash] = f.Checks[fieldR66NoFinalHash]
		cfg[cfgCheckBlockHash] = f.Checks[fieldR66CheckBlockHash]
	case familyFTP:
		cfg[cfgDisablePassiveMode] = f.Checks[fieldFTPDisablePassive]
		cfg[cfgDisableActiveMode] = f.Checks[fieldFTPDisableActive]
		setNumber(cfg, cfgPassiveModeMinPort, value(fieldFTPPassiveMin), parsePort)
		setNumber(cfg, cfgPassiveModeMaxPort, value(fieldFTPPassiveMax), parsePort)

		if proto.TLS {
			setText(cfg, cfgTLSRequirement, f.selects[fieldFTPSRequirement])
		}
	case familyPeSIT:
		cfg[cfgDisableRestart] = f.Checks[fieldPesitDisableRestart]
		cfg[cfgDisableCheckpoints] = f.Checks[fieldPesitDisableCheckpoints]
		setNumber(cfg, cfgCheckpointSize, value(fieldPesitCheckpointSize), internal.ParseUint[uint32])
		setNumber(cfg, cfgCheckpointWindow, value(fieldPesitCheckpointWindow), internal.ParseUint[uint8])
		setNumber(cfg, cfgMaxMessageSize, value(fieldPesitMaxMessage), internal.ParseUint[uint16])
		setText(cfg, cfgCompatibilityMode, f.selects[fieldPesitMode])
		cfg[cfgDisablePreConnection] = f.Checks[fieldPesitDisablePreConn]
	case familySFTP:
		setList(cfg, cfgKeyExchanges, f.lists[fieldSFTPKex])
		setList(cfg, cfgCiphers, f.lists[fieldSFTPCiphers])
		setList(cfg, cfgMACs, f.lists[fieldSFTPMACs])
	case familyAS2:
		setNumber(cfg, cfgMaxFileSize, value(fieldAS2MaxFileSize), internal.ParseInt[int64])
		setText(cfg, cfgMDNSignatureAlgo, f.selects[fieldAS2MDNAlgo])
	}

	if prefix, ok := tlsPrefixes[proto.Family]; ok && proto.TLS {
		setText(cfg, cfgMinTLSVersion, f.selects[prefix+suffixMinTLS])
		setList(cfg, cfgCipherSuites, f.lists[prefix+suffixTLSCiphers])
	}

	agent.ProtoConfig = cfg
}

func setText(cfg model.Map[any], key, value string) {
	if value != "" {
		cfg[key] = value
	}
}

func setList(cfg model.Map[any], key string, values []string) {
	if len(values) > 0 {
		cfg[key] = values
	}
}

// setNumber stores a number the form already validated; an empty field
// leaves the product default.
func setNumber[T constraints.Integer](cfg model.Map[any], key, value string, parse func(string) (T, error)) {
	if value == "" {
		return
	}

	if n, err := parse(value); err == nil {
		cfg[key] = n
	}
}

// refuse marks the form with the refusal of the model: the known causes go
// to their field, the others to the banner.
func (f *formData) refuse(translate gui.Translator, err error) {
	msg := err.Error()
	fail := func(name, key string) {
		fld := f.Fields[name]
		fld.Error = translate(key)
		f.Fields[name] = fld
	}

	switch {
	case strings.Contains(msg, "same name"):
		fail(fieldName, "servers.form.error.name_taken")
	case strings.Contains(msg, "same address"):
		fail(fieldHost, "servers.form.error.address_taken")
	case strings.Contains(msg, "address validation failed"):
		fail(fieldHost, "servers.form.error.address")
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// NewForm serves GET /webui/servers/new.
func (h *Handlers) NewForm(w http.ResponseWriter, r *http.Request) {
	translate := h.form.Translator(r)
	carried := ret(r)
	f := h.newForm(translate, nil, carried)

	h.form.Render(w, h.logger, r, h.screen(translate, f.Title, carried, 0, f))
}

// Create serves POST /webui/servers/new.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, nil)
}

// EditForm serves GET /webui/servers/{id}/edit.
func (h *Handlers) EditForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.server(w, r)
	if !ok {
		return
	}

	translate := h.form.Translator(r)
	carried := ret(r)
	f := h.newForm(translate, e.agent, carried)

	h.form.Render(w, h.logger, r, h.screen(translate, f.Title, carried, 0, f))
}

// Update serves POST /webui/servers/{id}/edit.
func (h *Handlers) Update(w http.ResponseWriter, r *http.Request) {
	e, ok := h.server(w, r)
	if !ok {
		return
	}

	h.save(w, r, e.agent)
}

// save reads, validates and stores the posted form: a new server when agent
// is nil, the given one otherwise. A refused form is rendered again with its
// errors (422); a saved one redirects to the list with a notification.
func (h *Handlers) save(w http.ResponseWriter, r *http.Request, agent *model.LocalAgent) {
	translate := h.form.Translator(r)
	carried := ret(r)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := h.newForm(translate, agent, carried)
	form.read(r.PostForm)

	if form.validate(translate) {
		done := doneUpdated

		if agent == nil {
			agent = &model.LocalAgent{}
			done = doneCreated
		}

		form.apply(agent)

		var err error
		if done == doneCreated {
			err = h.ctl.Add(h.db, agent)
		} else {
			err = internal.UpdateServer(h.db, agent)
		}

		switch {
		case err == nil:
			redirectList(w, r, carried, url.Values{doneParam: {done}, nameParam: {agent.Name}})

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the server %q: %v", agent.Name, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	form.Choices = buildChoices(translate, form)
	h.form.Render(w, h.logger, r, h.screen(translate, form.Title, carried, http.StatusUnprocessableEntity, form))
}
