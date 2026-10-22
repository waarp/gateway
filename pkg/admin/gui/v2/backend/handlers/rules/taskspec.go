package rules

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

// The forms of the tasks are built from one typed description per task type:
// the arguments the task reads, how the form asks for each one, whether the
// task requires it. The texts come from the catalogue (task_types.<slug>).
// The description takes over the definitions of the former tasks page (its
// fields, lists, labels and help texts) and a test checks it against the
// arguments of the tasks in pkg/tasks: none missing, none extra.

// argKind is how the form asks for an argument of a task.
type argKind int

const (
	argText     argKind = iota // a line of text: a path, a pattern, a variable
	argBool                    // a box: "true" when ticked, nothing otherwise
	argInt                     // a whole number, between Min and Max
	argDecimal                 // a decimal number
	argMillis                  // a whole number of milliseconds
	argDuration                // a duration: 30s, 5m, 1h30m
	argOctal                   // permissions in octal: 640
	argChoice                  // one of the values of Choices
	argRef                     // the name of an object of the gateway (Ref)
	argJSON                    // a JSON object
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

// argSpec is an argument of a task.
type argSpec struct {
	Name     string
	Kind     argKind
	Required bool
	Min, Max int // bounds of a number; Max 0: none
	Choices  func() []choiceValue
	None     string // the catalogue key of the empty choice of an optional argument; "": "None"
	Ref      refKind
}

// taskSpec is a type of task: its name in the product, its key in the
// catalogue, its family, its arguments. Mapped is a task that reads its
// arguments from a map rather than from a structure (COPY, EXEC…): the test
// checks those by the validation of the product.
type taskSpec struct {
	Type   string
	Slug   string
	Family string
	Args   []argSpec
	Mapped bool
}

func pathArg(required bool) argSpec { return argSpec{Name: "path", Kind: argText, Required: required} }

func textArg(name string, required bool) argSpec {
	return argSpec{Name: name, Kind: argText, Required: required}
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

func execArgs() []argSpec {
	return []argSpec{pathArg(true), textArg("args", false), {Name: "delay", Kind: argMillis, Min: 0, Max: 0}}
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
)

//nolint:gochecknoglobals // reference data of the screen: the order of the families
var families = []string{
	familyFiles, familyExec, familyTransfer, familyAlteration, familyArchive, familyNetwork, familyCrypto,
}

// taskSpecs are the 28 types of task, in the order of the former tasks page.
//
//nolint:gochecknoglobals,funlen // reference data of the screen
var taskSpecs = []taskSpec{
	{Type: tasks.Copy, Slug: "copy", Family: familyFiles, Args: []argSpec{pathArg(true)}, Mapped: true},
	{Type: tasks.CopyRename, Slug: "copyrename", Family: familyFiles, Args: []argSpec{pathArg(true)}, Mapped: true},
	{Type: tasks.Move, Slug: "move", Family: familyFiles, Args: []argSpec{pathArg(true)}, Mapped: true},
	{Type: tasks.MoveRename, Slug: "moverename", Family: familyFiles, Args: []argSpec{pathArg(true)}, Mapped: true},
	{Type: tasks.Delete, Slug: "delete", Family: familyFiles, Mapped: true},
	{Type: tasks.RemoteDelete, Slug: "remotedelete", Family: familyFiles, Args: []argSpec{
		pathArg(false), boolArg("recursive"), durationArg("timeout"),
	}},
	{Type: tasks.Chmod, Slug: "chmod", Family: familyFiles, Args: []argSpec{
		{Name: "perms", Kind: argOctal, Required: true},
	}},
	{Type: tasks.Exec, Slug: "exec", Family: familyExec, Args: execArgs(), Mapped: true},
	{Type: tasks.ExecMove, Slug: "execmove", Family: familyExec, Args: execArgs(), Mapped: true},
	{Type: tasks.ExecOutput, Slug: "execoutput", Family: familyExec, Args: execArgs(), Mapped: true},
	{Type: tasks.UpdateConf, Slug: "updateconf", Family: familyExec, Args: []argSpec{textArg("zipFile", false)}},
	{Type: tasks.Rename, Slug: "rename", Family: familyTransfer, Args: []argSpec{pathArg(true)}, Mapped: true},
	{Type: tasks.Transfer, Slug: "transfer", Family: familyTransfer, Args: []argSpec{
		boolArg("synchronous"), textArg("file", true), textArg("output", false),
		refArg("to", refPartner, false), refArg("from", refPartner, false), refArg("as", refRemoteAccount, true),
		refArg("using", refClient, false), refArg("rule", refRule, true),
		{Name: "info", Kind: argJSON},
		boolArg("copyInfo"),
		{Name: "nbOfAttempts", Kind: argInt, Min: 0, Max: 0},
		durationArg("firstRetryDelay"),
		{Name: "retryIncrementFactor", Kind: argDecimal},
		durationArg("timeout"),
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
	}},
	{Type: tasks.SetInfo, Slug: "setinfo", Family: familyTransfer, Args: []argSpec{
		{Name: "pairs", Kind: argPairs},
	}, Mapped: true},
	{Type: tasks.Transcode, Slug: "transcode", Family: familyAlteration, Args: []argSpec{
		choiceArg("fromCharset", keysOf(tasks.TranscodeFormats.Keys)),
		choiceArg("toCharset", keysOf(tasks.TranscodeFormats.Keys)),
	}},
	{Type: tasks.ChangeNewline, Slug: "chnewline", Family: familyAlteration, Args: []argSpec{
		textArg("from", true), textArg("to", true),
	}},
	{Type: tasks.Archive, Slug: "archive", Family: familyArchive, Args: []argSpec{
		textArg("files", true),
		{Name: "compressionLevel", Kind: argInt, Min: 0, Max: maxCompression},
		textArg("outputPath", true),
	}},
	{Type: tasks.Extract, Slug: "extract", Family: familyArchive, Args: []argSpec{
		textArg("archive", false), textArg("outputDir", true),
	}},
	{Type: tasks.Icap, Slug: "icap", Family: familyNetwork, Args: []argSpec{
		textArg("uploadURL", true), boolArg("useTLS"), durationArg("timeout"), boolArg("allowFileModifications"),
		{Name: "onError", Kind: argChoice, None: "task_types.choice.icap_nothing", Choices: func() []choiceValue {
			return []choiceValue{
				{tasks.IcapOnErrorDelete, "task_types.choice.icap_delete"},
				{tasks.IcapOnErrorMove, "task_types.choice.icap_move"},
			}
		}},
		textArg("onErrorMovePath", false),
	}},
	{Type: tasks.Email, Slug: "email", Family: familyNetwork, Args: []argSpec{
		refArg("sender", refSMTP, true), textArg("recipients", true), refArg("template", refTemplate, true),
	}},
	{Type: tasks.SendMessage, Slug: "sendmessage", Family: familyNetwork, Args: []argSpec{
		refArg("partner", refPartner, true), refArg("account", refRemoteAccount, true), refArg("client", refClient, true),
		textArg("message", false),
	}},
	{Type: tasks.Encrypt, Slug: "encrypt", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), boolArg("keepOriginal"),
		choiceArg("method", keysOf(tasks.EncryptMethods.Keys)), refArg("keyName", refKey, true),
	}},
	{Type: tasks.Decrypt, Slug: "decrypt", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), boolArg("keepOriginal"),
		choiceArg("method", keysOf(tasks.DecryptMethods.Keys)), refArg("keyName", refKey, true),
	}},
	{Type: tasks.Sign, Slug: "sign", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), choiceArg("method", keysOf(tasks.SignMethods.Keys)),
		refArg("keyName", refKey, true),
	}},
	{Type: tasks.Verify, Slug: "verify", Family: familyCrypto, Args: []argSpec{
		textArg("signatureFile", true), choiceArg("method", keysOf(tasks.VerifyMethods.Keys)),
		refArg("keyName", refKey, true),
	}},
	{Type: tasks.EncryptAndSign, Slug: "encrypt_sign", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), boolArg("keepOriginal"),
		choiceArg("method", keysOf(tasks.EncryptSignMethods.Keys)),
		refArg("encryptKeyName", refKey, true), refArg("signKeyName", refKey, true),
	}},
	{Type: tasks.DecryptAndVerify, Slug: "decrypt_verify", Family: familyCrypto, Args: []argSpec{
		textArg("outputFile", false), boolArg("keepOriginal"),
		choiceArg("method", keysOf(tasks.DecryptVerifyMethods.Keys)),
		refArg("decryptKeyName", refKey, true), refArg("verifyKeyName", refKey, true),
	}},
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
