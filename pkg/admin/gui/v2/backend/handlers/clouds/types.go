package clouds

import (
	"slices"
	"strings"

	"code.waarp.fr/lib/gui"
)

// The storage types the product registers (pkg/fs/backends), described for
// the screen: the key the product reads, the other keys it accepts for the
// same storage, the option that locates the storage, whether the credentials
// may come from the environment, how the credentials are named, and the
// options the reference documentation lists, with an example of a value
// (the ones of the reference documentation where it gives one). What the
// product applies when an option is left out is the default of its storage
// library (#701), read in its registry (resolved). One source for the list,
// the sheet, the form and the validation; a test confronts it with the
// registry of the product and with the library.

// Kinds of an option field.
const (
	kindText   = "text"
	kindCheck  = "check"
	kindSelect = "select"
)

// Families of the credential labels: what the key and the secret are for
// the storage.
const (
	credsS3       = "s3"
	credsAzure    = "azure"
	credsGCS      = "gcs"
	credsOneDrive = "onedrive"
)

// Option keys the screen knows by name.
const (
	optEnvAuth = "env_auth" // the credentials come from the environment variables
	optToken   = "token"    // the OAuth token the OneDrive storage writes at save time: a secret

	// Prefixes of the catalogue keys chosen by the storage: the label and
	// the sentence of a type, the options ("clouds.opt.bucket.label",
	// ".hint…"), what the key is for a credential family, the sentence of
	// the access card and the environment variables of a storage.
	typeKeyPrefix       = "clouds.type."
	typeHintKeyPrefix   = "clouds.type_hint."
	optKeyPrefix        = "clouds.opt."
	keyLabelPrefix      = "clouds.form.key."
	accessNoteKeyPrefix = "clouds.form.access_note."
	envKeyPrefix        = "clouds.form.env."
)

// option is an option of a storage type.
type option struct {
	Key      string // the key the product reads ("bucket")
	Kind     string // kindText, kindCheck, kindSelect
	Required bool
	Default  string // what the product applies when the option is absent: the library default (resolved)
	Initial  string // the choice a select proposes when the instance has none, the library having no default
	Example  string // a value of the option, shown in its empty field when it has no default
	Mono     bool   // an identifier, rendered in monospace
	Choices  []string
	Hint     string // suffix of the hint key ("hint", "hint_s3"); "" for no hint
}

// storage is a storage type: a backend of the product.
type storage struct {
	Key      string   // the canonical key ("s3")
	Aliases  []string // the other keys the product registers for the same storage ("aws")
	Location string   // the option that locates the storage ("bucket"), "" when the path names it
	EnvAuth  bool     // the credentials may come from the environment variables
	Creds    string   // family of the credential labels (creds*)
	Options  []option
}

//nolint:gochecknoglobals // reference data of the product, mirrored from pkg/fs/backends
var storages = []storage{
	{
		Key: "s3", Aliases: []string{"aws"}, Location: "bucket", EnvAuth: true, Creds: credsS3,
		Options: []option{
			{Key: "bucket", Kind: kindText, Required: true, Example: "archive", Mono: true, Hint: "hint"},
			{Key: "region", Kind: kindText, Example: "eu-west-3", Mono: true, Hint: "hint_s3"},
			{Key: "endpoint", Kind: kindText, Example: "s3.eu-west-3.amazonaws.com", Mono: true, Hint: "hint_s3"},
			{Key: "chunk_size", Kind: kindText, Mono: true, Hint: "hint"},
			{Key: "copy_cutoff", Kind: kindText, Mono: true, Hint: "hint"},
		},
	},
	{
		Key: "azblob", Aliases: []string{"azureblob"}, EnvAuth: true, Creds: credsAzure,
		Options: []option{
			{
				Key: "endpoint", Kind: kindText, Mono: true, Hint: "hint_azblob",
				Example: "https://myaccount.blob.core.windows.net",
			},
			{Key: "chunk_size", Kind: kindText, Mono: true, Hint: "hint"},
			{Key: "list_chunk", Kind: kindText, Mono: true, Hint: "hint"},
		},
	},
	{
		Key: "azfiles", Aliases: []string{"azurefiles"}, Location: "share_name", EnvAuth: true, Creds: credsAzure,
		Options: []option{
			{Key: "share_name", Kind: kindText, Required: true, Example: "archive", Mono: true, Hint: "hint"},
			{
				Key: "endpoint", Kind: kindText, Mono: true, Hint: "hint_azfiles",
				Example: "https://myaccount.file.core.windows.net",
			},
		},
	},
	{
		Key: "gcs", Aliases: []string{"google"}, Location: "bucket", EnvAuth: true, Creds: credsGCS,
		Options: []option{
			{Key: "bucket", Kind: kindText, Required: true, Example: "archive", Mono: true, Hint: "hint"},
			{Key: "project_number", Kind: kindText, Required: true, Example: "123456789012", Mono: true, Hint: "hint"},
			{Key: "bucket_policy_only", Kind: kindCheck},
		},
	},
	{
		Key: "onedrive", Aliases: []string{"sharepoint"}, Location: "drive_id", Creds: credsOneDrive,
		Options: []option{
			{
				Key: "tenant", Kind: kindText, Required: true, Mono: true, Hint: "hint",
				Example: "3f2504e0-4f89-11d3-9a0c-0305e82c3301",
			},
			{
				Key: "drive_id", Kind: kindText, Required: true, Mono: true, Hint: "hint",
				Example: "b!GoXLzaJCDEqhqT5H107_ksbhUsDDwtJPlR5kTWhg2CxYuCODHjHUTJOI3xBLJIMh",
			},
			{
				Key: "drive_type", Kind: kindSelect, Required: true, Initial: "business",
				Choices: []string{"business", "personal", "documentLibrary"},
			},
			{
				Key: "region", Kind: kindSelect,
				Choices: []string{"global", "us", "de", "cn"}, Hint: "hint_onedrive",
			},
			{Key: "chunk_size", Kind: kindText, Mono: true, Hint: "hint"},
		},
	},
}

// credentialOptionKeys are the option keys that carried the credentials
// before 0.17.0: the product reads them from the key and the secret, never
// from the options (migration 0.17.0), so the form never writes them.
//
//nolint:gochecknoglobals // reference data of the product
var credentialOptionKeys = []string{
	"access_key_id", "secret_access_key", "account", "key", "client_id", "client_secret",
	"service_account_credentials", "service_account_file",
}

// resolved returns the options of the storage with the defaults of its
// library: what the product applies when an option is absent (#701). The
// options the library does not know (the bucket) keep the table's.
func (s *storage) resolved() []option {
	options := slices.Clone(s.Options)

	lib, ok := libraryOf(s)
	if !ok {
		return options
	}

	for i := range options {
		if o, known := lib.options[options[i].Key]; known {
			options[i].Default = o.Default
		}
	}

	return options
}

// storageOf finds a storage by its canonical key or one of its aliases.
func storageOf(key string) (*storage, bool) {
	for i := range storages {
		s := &storages[i]
		if s.Key == key || slices.Contains(s.Aliases, key) {
			return s, true
		}
	}

	return nil, false
}

// knows tells whether the storage names the option: such an option has its
// field in the form, the others go to "other options".
func (s *storage) knows(key string) bool {
	return slices.ContainsFunc(s.Options, func(o option) bool { return o.Key == key })
}

// isSecret tells whether an option holds a secret (never rendered) or a
// credential (never written: the product reads them from the key and the
// secret).
func isSecret(key string) bool {
	return key == optToken || slices.Contains(credentialOptionKeys, key)
}

// labelKey is the catalogue key of the label of the storage type.
func (s *storage) labelKey() string { return typeKeyPrefix + s.Key }

// hintKey is the catalogue key of the sentence that says what the storage is.
func (s *storage) hintKey() string { return typeHintKeyPrefix + s.Key }

// keyLabelKey is the catalogue key of what the key is for the storage.
func (s *storage) keyLabelKey() string { return keyLabelPrefix + s.Creds }

// label is the label of a stored type key: the label of its storage, with
// the key when it is an alias ("Amazon S3 (aws)"); an unknown key reads as
// it is.
func label(translate gui.Translator, key string) string {
	s, ok := storageOf(key)
	if !ok {
		return key
	}

	if key != s.Key {
		return translate("clouds.type_stored", "label", translate(s.labelKey()), "key", key)
	}

	return translate(s.labelKey())
}

// labelKey is the catalogue key of the label of the option.
func (o *option) labelKey() string { return optKeyPrefix + o.Key + ".label" }

// hint is the hint of the option, with the default of the product.
func (o *option) hint(translate gui.Translator) string {
	if o.Hint == "" {
		return ""
	}

	return translate(optKeyPrefix+o.Key+"."+o.Hint, "default", o.Default)
}

// placeholder is what the empty field of the option shows: the default of
// the product, which applies when it stays empty, or an example of a value.
func (o *option) placeholder() string {
	if o.Default != "" {
		return o.Default
	}

	return o.Example
}

// isSet tells whether a value of the option differs from what the product
// applies without it: the summary of a folded card counts these.
func (o *option) isSet(value string, checked bool) bool {
	if o.Kind == kindCheck {
		return checked
	}

	return value != "" && value != o.Default
}

// choiceLabel is the label of a choice of a select option: the catalogue
// names the ones that deserve a word, the others read as they are.
func (o *option) choiceLabel(translate gui.Translator, choice string) string {
	if o.Key == "drive_type" {
		return translate(optKeyPrefix + o.Key + "." + choice)
	}

	return choice
}

// authOf says how an instance authenticates: by the environment variables,
// by a key, or not at all.
func authOf(key, secret string, options map[string]string) string {
	switch {
	case strings.EqualFold(options[optEnvAuth], "true"):
		return "env"
	case key != "" || secret != "":
		return "key"
	default:
		return "none"
	}
}
