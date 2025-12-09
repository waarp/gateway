package api

// AuthorizedRules represents a list of all the rules which an agent/account
// is allowed to use.
type AuthorizedRules struct {
	Sending   []string `json:"sending,omitempty" yaml:"sending,omitempty"`
	Reception []string `json:"reception,omitempty" yaml:"reception,omitempty"`
}

// RuleAccess is the struct containing all the agents/accounts which are allowed
// to use a given rule.
type RuleAccess struct {
	Servers        []string            `json:"servers,omitempty" yaml:"servers,omitempty"`
	Partners       []string            `json:"partners,omitempty" yaml:"partners,omitempty"`
	LocalAccounts  map[string][]string `json:"localAccounts,omitempty" yaml:"localAccounts,omitempty"`
	RemoteAccounts map[string][]string `json:"remoteAccounts,omitempty" yaml:"remoteAccounts,omitempty"`
}

// InRule is the JSON representation of a transfer rule in requests made to
// the REST interface.
type InRule struct {
	Name           Nullable[string] `json:"name,omitzero" yaml:"name,omitempty"`
	IsSend         Nullable[bool]   `json:"isSend,omitzero" yaml:"isSend,omitempty"`
	Comment        Nullable[string] `json:"comment,omitzero" yaml:"comment,omitempty"`
	Path           Nullable[string] `json:"path,omitzero" yaml:"path,omitempty"`
	LocalDir       Nullable[string] `json:"localDir,omitzero" yaml:"localDir,omitempty"`
	RemoteDir      Nullable[string] `json:"remoteDir,omitzero" yaml:"remoteDir,omitempty"`
	TmpLocalRcvDir Nullable[string] `json:"tmpLocalRcvDir,omitzero" yaml:"tmpLocalRcvDir,omitempty"`
	PreTasks       []*Task          `json:"preTasks,omitempty" yaml:"preTasks,omitempty"`
	PostTasks      []*Task          `json:"postTasks,omitempty" yaml:"postTasks,omitempty"`
	ErrorTasks     []*Task          `json:"errorTasks,omitempty" yaml:"errorTasks,omitempty"`

	// Deprecated fields
	InPath   Nullable[string] `json:"inPath,omitzero"`   // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  Nullable[string] `json:"outPath,omitzero"`  // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath Nullable[string] `json:"workPath,omitzero"` // Deprecated: replaced by TmpLocalRcvDir
}

// OutRule is the JSON representation of a transfer rule in responses sent by
// the REST interface.
type OutRule struct {
	Name           string     `json:"name" yaml:"name"`
	Comment        string     `json:"comment,omitempty" yaml:"comment,omitempty"`
	IsSend         bool       `json:"isSend" yaml:"isSend"`
	Path           string     `json:"path" yaml:"path"`
	LocalDir       string     `json:"localDir,omitempty" yaml:"localDir,omitempty"`
	RemoteDir      string     `json:"remoteDir,omitempty" yaml:"remoteDir,omitempty"`
	TmpLocalRcvDir string     `json:"tmpLocalRcvDir,omitempty" yaml:"tmpLocalRcvDir,omitempty"`
	Authorized     RuleAccess `json:"authorized,omitzero" yaml:"authorized,omitempty"`
	PreTasks       []*Task    `json:"preTasks,omitempty" yaml:"preTasks,omitempty"`
	PostTasks      []*Task    `json:"postTasks,omitempty" yaml:"postTasks,omitempty"`
	ErrorTasks     []*Task    `json:"errorTasks,omitempty" yaml:"errorTasks,omitempty"`

	// Deprecated fields
	InPath   string `json:"inPath,omitempty"`   // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  string `json:"outPath,omitempty"`  // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath string `json:"workPath,omitempty"` // Deprecated: replaced by TmpLocalRcvDir
}
