package rules

import (
	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The forms of the tasks are built from one typed description per task type:
// the arguments the task reads, how the form asks for each one, whether the
// task requires it, and the cards that group them on the page. The texts come
// from the catalogue (task_types.<slug>, task_types.group). The description
// takes over the definitions of the former tasks page (its fields, lists,
// labels and help texts) and a test checks it against the arguments of the
// tasks in pkg/tasks: none missing, none extra; another one, that every
// argument has its place in one card.

// argKind is how the form asks for an argument of a task.
type argKind int

const (
	argText     argKind = iota // a line of text: a path, a pattern, a variable
	argBool                    // a box: "true" when ticked, nothing otherwise
	argInt                     // a whole number, between Min and Max
	argDecimal                 // a decimal number
	argDuration                // a duration: 30s, 5m, 1h30m
	argOctal                   // permissions in octal: 640
	argChoice                  // one of the values of Choices
	argRef                     // the name of an object of the gateway (Ref)
	argJSON                    // a JSON object
	argICAP                    // the address of an ICAP server, "icap://" optional (the gateway adds it)
	argPairs                   // free keys and values, one "key=value" a line (SETINFO)
)

// refKind is the kind of object an argument names.
type refKind int

const (
	refNone          refKind = iota
	refKey                   // a cryptographic key
	refSMTP                  // an SMTP credential, by its address
	refTemplate              // an email template
	refPartner               // a remote partner
	refRemoteAccount         // the login of a remote account, grouped by partner
	refClient                // a local client
	refServer                // a local server
	refLocalAccount          // the login of a local account, grouped by server
	refRule                  // a transfer rule
)

// choiceValue is a value of a choice, with the catalogue key of its label ("" for
// the value itself).
type choiceValue struct {
	Value string
	Key   string
}

func choiceValues(list []string) []choiceValue {
	out := make([]choiceValue, 0, len(list))
	for _, v := range list {
		out = append(out, choiceValue{Value: v})
	}

	return out
}

// argWhen: an argument that only matters when another one has a value (the
// path of a refused file when it is moved, the time limit of a synchronous
// transfer); it is shown, read and, when Required, required only then.
type argWhen struct {
	Arg   string
	Value string
}

// argSpec is an argument of a task. Seconds: a duration the form asks in
// seconds, stored as "30s". Millis: a duration the form asks as "30s, 5m,
// 1h30m", stored as the whole number of milliseconds the product reads; empty,
// no limit. Summary: shown on the folded line of the task, with its type.
// Modes: a box asked as two buttons, the catalogue keys of the labels of ""
// and of "true". Wide: over the whole width of its card. Check: the type the
// kit checks a text against (the durations and the permissions take theirs
// from their kind).
type argSpec struct {
	Name     string
	Kind     argKind
	Required bool
	Min, Max int // bounds of a number; Max 0: none
	Choices  func() []choiceValue
	None     string // the catalogue key of the empty choice of an optional argument; "": "None"
	Ref      refKind
	Seconds  bool
	Millis   bool
	Summary  bool
	Modes    [2]string
	When     *argWhen
	Wide     bool
	Check    gui.Check
}

// groupKind is what a card of the form of a task holds.
type groupKind int

const (
	groupArgs     groupKind = iota // the arguments of Args, side by side
	groupInfos                     // the transfer information: its copy and its rows (copyInfo, info)
	groupExchange                  // TRANSFER: the direction, the rule, the partner and its account, the client
)

// argGroup is a card of the form of a task: its title (a key under
// task_types.group, "" for none), what it holds, a note under its title (a
// catalogue key) and its columns (two, or three short fields).
type argGroup struct {
	Title   string
	Kind    groupKind
	Args    []string
	Note    string
	Columns int
}

func group(title string, args ...string) argGroup { return argGroup{Title: title, Args: args} }

// covers are the arguments a card asks for.
func (g *argGroup) covers() []string {
	switch g.Kind {
	case groupInfos:
		return []string{"copyInfo", "info"}
	case groupExchange:
		return []string{"to", "from", "as", "rule", "using"}
	case groupArgs:
	}

	return g.Args
}

// taskSpec is a type of task: its name in the product, its key in the
// catalogue, its family, its arguments, its cards. Mapped is a task that reads
// its arguments from a map rather than from a structure (COPY, EXEC…): the
// test checks those by the validation of the product.
type taskSpec struct {
	Type   string
	Slug   string
	Family string
	Args   []argSpec
	Groups []argGroup
	Mapped bool
}

// has says whether one of the cards of the type is of a kind.
func (s *taskSpec) has(kind groupKind) bool {
	for i := range s.Groups {
		if s.Groups[i].Kind == kind {
			return true
		}
	}

	return false
}

// special are the arguments its cards of information and of exchange ask
// for, read by those cards.
func (s *taskSpec) special() map[string]bool {
	out := map[string]bool{}

	for i := range s.Groups {
		if s.Groups[i].Kind != groupArgs {
			for _, name := range s.Groups[i].covers() {
				out[name] = true
			}
		}
	}

	return out
}

// arg is the index of an argument in the description.
func (s *taskSpec) arg(name string) int {
	for i := range s.Args {
		if s.Args[i].Name == name {
			return i
		}
	}

	return -1
}

func pathArg(required bool) argSpec { return argSpec{Name: "path", Kind: argText, Required: required} }

func textArg(name string, required bool) argSpec {
	return argSpec{Name: name, Kind: argText, Required: required}
}

func wideArg(name string, required bool) argSpec {
	return argSpec{Name: name, Kind: argText, Required: required, Wide: true}
}

func boolArg(name string) argSpec { return argSpec{Name: name, Kind: argBool} }

func durationArg(name string) argSpec { return argSpec{Name: name, Kind: argDuration} }

func refArg(name string, kind refKind, required bool) argSpec {
	return argSpec{Name: name, Kind: argRef, Ref: kind, Required: required}
}

// choiceArg is a required choice; an optional one sets its empty choice (None).
func choiceArg(name string, list func() []choiceValue) argSpec {
	return argSpec{Name: name, Kind: argChoice, Required: true, Choices: list}
}

func keysOf(keys func() []string) func() []choiceValue {
	return func() []choiceValue { return choiceValues(keys()) }
}

// execArgs: the program, its arguments, its time limit; the program and the
// limit make the folded line of the task (#784).
func execArgs() []argSpec {
	program := pathArg(true)
	program.Summary = true

	return []argSpec{program, wideArg("args", false), {Name: "delay", Kind: argDuration, Millis: true, Summary: true}}
}

func execGroups() []argGroup {
	return []argGroup{group("program", "path", "args"), group("execution", "delay")}
}

// The families of the former tasks page, in its order.
const (
	familyFiles      = "fileOperations"
	familyExec       = "executables"
	familyTransfer   = "transfer"
	familyAlteration = "fileAlterations"
	familyArchive    = "archiving"
	familyNetwork    = "network"
	familyCrypto     = "crypto"

	maxCompression = 9
	maxAttempts    = 127 // the attempts of a transfer are an int8

	// The columns of a card: two fields side by side, or three short ones.
	twoColumns   = 2
	threeColumns = 3
)

//nolint:gochecknoglobals // reference data of the screen: the order of the families
var families = []string{
	familyFiles, familyExec, familyTransfer, familyAlteration, familyArchive, familyNetwork, familyCrypto,
}

// taskSpecs are the 28 types of task, in the order of the former tasks page.
// Their cards follow the prototype validated by Driss on 2026-10-23; those of
// TRANSFER follow the page that adds a transfer.
//
//nolint:gochecknoglobals,funlen // reference data of the screen
var taskSpecs = []taskSpec{
	{
		Type: tasks.Copy, Slug: "copy", Family: familyFiles, Args: []argSpec{pathArg(true)},
		Groups: []argGroup{group("", "path")}, Mapped: true,
	},
	{
		Type: tasks.CopyRename, Slug: "copyrename", Family: familyFiles, Args: []argSpec{pathArg(true)},
		Groups: []argGroup{group("", "path")}, Mapped: true,
	},
	{
		Type: tasks.Move, Slug: "move", Family: familyFiles, Args: []argSpec{pathArg(true)},
		Groups: []argGroup{group("", "path")}, Mapped: true,
	},
	{
		Type: tasks.MoveRename, Slug: "moverename", Family: familyFiles, Args: []argSpec{pathArg(true)},
		Groups: []argGroup{group("", "path")}, Mapped: true,
	},
	{Type: tasks.Delete, Slug: "delete", Family: familyFiles, Mapped: true},
	{Type: tasks.RemoteDelete, Slug: "remotedelete", Family: familyFiles, Args: []argSpec{
		pathArg(false), boolArg("recursive"), durationArg("timeout"),
	}, Groups: []argGroup{group("target", "path", "recursive"), group("execution", "timeout")}},
	{Type: tasks.Chmod, Slug: "chmod", Family: familyFiles, Args: []argSpec{
		{Name: "perms", Kind: argOctal, Required: true},
	}, Groups: []argGroup{group("", "perms")}},
	{Type: tasks.Exec, Slug: "exec", Family: familyExec, Args: execArgs(), Groups: execGroups(), Mapped: true},
	{Type: tasks.ExecMove, Slug: "execmove", Family: familyExec, Args: execArgs(), Groups: execGroups(), Mapped: true},
	{Type: tasks.ExecOutput, Slug: "execoutput", Family: familyExec, Args: execArgs(), Groups: execGroups(), Mapped: true},
	{
		Type: tasks.UpdateConf, Slug: "updateconf", Family: familyExec, Args: []argSpec{textArg("zipFile", false)},
		Groups: []argGroup{group("", "zipFile")},
	},
	{
		Type: tasks.Rename, Slug: "rename", Family: familyTransfer, Args: []argSpec{pathArg(true)},
		Groups: []argGroup{group("", "path")}, Mapped: true,
	},
	{Type: tasks.Transfer, Slug: "transfer", Family: familyTransfer, Args: []argSpec{
		{Name: "synchronous", Kind: argBool, Modes: [2]string{"task_types.choice.async", "task_types.choice.sync"}},
		textArg("file", true), textArg("output", false),
		refArg("to", refPartner, false), refArg("from", refPartner, false), refArg("as", refRemoteAccount, true),
		{Name: "using", Kind: argRef, Ref: refClient, None: "transfers.form.client_default"},
		refArg("rule", refRule, true),
		{Name: "info", Kind: argJSON},
		boolArg("copyInfo"),
		{Name: "nbOfAttempts", Kind: argInt, Min: 0, Max: maxAttempts},
		{Name: "firstRetryDelay", Kind: argDuration, Seconds: true},
		{Name: "retryIncrementFactor", Kind: argDecimal},
		{Name: "timeout", Kind: argDuration, Seconds: true, When: &argWhen{Arg: "synchronous", Value: trueValue}},
	}, Groups: []argGroup{
		group("file", "file", "output"),
		{Title: "exchange", Kind: groupExchange},
		{Title: "infos", Kind: groupInfos},
		{
			Title: "retries", Args: []string{"nbOfAttempts", "firstRetryDelay", "retryIncrementFactor"},
			Note: "transfers.form.retries_hint", Columns: threeColumns,
		},
		group("execution", "synchronous", "timeout"),
	}},
	{Type: tasks.Preregister, Slug: "preregister", Family: familyTransfer, Args: []argSpec{
		textArg("file", true),
		choiceArg("isSend", func() []choiceValue {
			return []choiceValue{{"true", "task_types.choice.send"}, {"false", "task_types.choice.receive"}}
		}),
		refArg("rule", refRule, true), refArg("server", refServer, true), refArg("account", refLocalAccount, true),
		durationArg("validFor"),
		{Name: "info", Kind: argJSON},
		boolArg("copyInfo"),
	}, Groups: []argGroup{
		group("file", "file", "isSend"), group("rule_account", "rule", "server", "account"), group("validity", "validFor"),
		{Title: "infos", Kind: groupInfos},
	}},
	{Type: tasks.SetInfo, Slug: "setinfo", Family: familyTransfer, Args: []argSpec{
		{Name: "pairs", Kind: argPairs},
	}, Groups: []argGroup{group("", "pairs")}, Mapped: true},
	{Type: tasks.Transcode, Slug: "transcode", Family: familyAlteration, Args: []argSpec{
		choiceArg("fromCharset", keysOf(tasks.TranscodeFormats.Keys)),
		choiceArg("toCharset", keysOf(tasks.TranscodeFormats.Keys)),
	}, Groups: []argGroup{group("", "fromCharset", "toCharset")}},
	{Type: tasks.ChangeNewline, Slug: "chnewline", Family: familyAlteration, Args: []argSpec{
		textArg("from", true), textArg("to", true),
	}, Groups: []argGroup{group("", "from", "to")}},
	{Type: tasks.Archive, Slug: "archive", Family: familyArchive, Args: []argSpec{
		wideArg("files", true),
		{Name: "compressionLevel", Kind: argInt, Min: 0, Max: maxCompression},
		textArg("outputPath", true),
	}, Groups: []argGroup{group("files", "files"), group("archive", "outputPath", "compressionLevel")}},
	{Type: tasks.Extract, Slug: "extract", Family: familyArchive, Args: []argSpec{
		textArg("archive", false), textArg("outputDir", true),
	}, Groups: []argGroup{group("", "archive", "outputDir")}},
	{Type: tasks.Icap, Slug: "icap", Family: familyNetwork, Args: []argSpec{
		{Name: "uploadURL", Kind: argICAP, Required: true, Wide: true},
		boolArg("useTLS"), durationArg("timeout"), boolArg("allowFileModifications"),
		{Name: "onError", Kind: argChoice, None: "task_types.choice.icap_nothing", Choices: func() []choiceValue {
			return []choiceValue{
				{tasks.IcapOnErrorDelete, "task_types.choice.icap_delete"},
				{tasks.IcapOnErrorMove, "task_types.choice.icap_move"},
			}
		}},
		{
			Name: "onErrorMovePath", Kind: argText, Required: true,
			When: &argWhen{Arg: "onError", Value: tasks.IcapOnErrorMove},
		},
	}, Groups: []argGroup{
		group("icap_server", "uploadURL", "timeout", "useTLS"), group("file", "allowFileModifications"),
		group("refused", "onError", "onErrorMovePath"),
	}},
	{Type: tasks.Email, Slug: "email", Family: familyNetwork, Args: []argSpec{
		refArg("sender", refSMTP, true),
		{Name: "recipients", Kind: argText, Required: true, Wide: true, Check: gui.CheckEmails},
		refArg("template", refTemplate, true),
	}, Groups: []argGroup{group("sending", "sender", "template"), group("recipients", "recipients")}},
	{Type: tasks.SendMessage, Slug: "sendmessage", Family: familyNetwork, Args: []argSpec{
		refArg("partner", refPartner, true), refArg("account", refRemoteAccount, true), refArg("client", refClient, true),
		wideArg("message", false),
	}, Groups: []argGroup{group("recipient", "partner", "account", "client"), group("message", "message")}},
	{Type: tasks.Encrypt, Slug: "encrypt", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), boolArg("keepOriginal"),
		choiceArg("method", keysOf(tasks.EncryptMethods.Keys)), refArg("keyName", refKey, true),
	}, Groups: []argGroup{group("encryption", "method", "keyName"), group("produced", "outputFile", "keepOriginal")}},
	{Type: tasks.Decrypt, Slug: "decrypt", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), boolArg("keepOriginal"),
		choiceArg("method", keysOf(tasks.DecryptMethods.Keys)), refArg("keyName", refKey, true),
	}, Groups: []argGroup{group("decryption", "method", "keyName"), group("produced", "outputFile", "keepOriginal")}},
	{Type: tasks.Sign, Slug: "sign", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), choiceArg("method", keysOf(tasks.SignMethods.Keys)),
		refArg("keyName", refKey, true),
	}, Groups: []argGroup{group("signature", "method", "keyName"), group("produced", "outputFile")}},
	{Type: tasks.Verify, Slug: "verify", Family: familyCrypto, Args: []argSpec{
		textArg("signatureFile", true), choiceArg("method", keysOf(tasks.VerifyMethods.Keys)),
		refArg("keyName", refKey, true),
	}, Groups: []argGroup{group("verification", "method", "keyName", "signatureFile")}},
	{Type: tasks.EncryptAndSign, Slug: "encrypt_sign", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), boolArg("keepOriginal"),
		choiceArg("method", keysOf(tasks.EncryptSignMethods.Keys)),
		refArg("encryptKeyName", refKey, true), refArg("signKeyName", refKey, true),
	}, Groups: []argGroup{
		group("method_keys", "method", "encryptKeyName", "signKeyName"), group("produced", "outputFile", "keepOriginal"),
	}},
	{Type: tasks.DecryptAndVerify, Slug: "decrypt_verify", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), boolArg("keepOriginal"),
		choiceArg("method", keysOf(tasks.DecryptVerifyMethods.Keys)),
		refArg("decryptKeyName", refKey, true), refArg("verifyKeyName", refKey, true),
	}, Groups: []argGroup{
		group("method_keys", "method", "decryptKeyName", "verifyKeyName"), group("produced", "outputFile", "keepOriginal"),
	}},
}

// check is the type the kit checks an argument against: its own, or the one
// of its kind (a duration not asked in seconds, permissions).
func (a *argSpec) check() gui.Check {
	switch {
	case a.Check != "":
		return a.Check
	case a.Kind == argDuration && !a.Seconds:
		return gui.CheckDuration
	case a.Kind == argOctal:
		return gui.CheckOctal
	}

	return ""
}

// specOf is the description of a type of task, nil for a type the screen
// does not know (a task saved by a newer version, say).
func specOf(typ string) *taskSpec {
	for i := range taskSpecs {
		if taskSpecs[i].Type == typ {
			return &taskSpecs[i]
		}
	}

	return nil
}
