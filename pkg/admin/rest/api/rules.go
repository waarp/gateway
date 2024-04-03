package api

// AuthorizedRules represents a list of all the rules which an agent/account
// is allowed to use.
type AuthorizedRules struct {
	Sending   []string `json:"sending,omitempty"`
	Reception []string `json:"reception,omitempty"`
}

// RuleAccess is the struct containing all the agents/accounts which are allowed
// to use a given rule.
type RuleAccess struct {
	LocalServers   []string            `json:"servers,omitempty"`  //nolint:tagliatelle // ok here
	RemotePartners []string            `json:"partners,omitempty"` //nolint:tagliatelle // ok here
	LocalAccounts  map[string][]string `json:"localAccounts,omitempty"`
	RemoteAccounts map[string][]string `json:"remoteAccounts,omitempty"`
}

// InRule is the JSON representation of a transfer rule in requests made to
// the REST interface.
type InRule struct {
	Name           Nullable[string] `json:"name,omitempty"`
	IsSend         Nullable[bool]   `json:"isSend,omitempty"`
	Comment        Nullable[string] `json:"comment,omitempty"`
	Path           Nullable[string] `json:"path,omitempty"`
	LocalDir       Nullable[string] `json:"localDir,omitempty"`
	RemoteDir      Nullable[string] `json:"remoteDir,omitempty"`
	TmpLocalRcvDir Nullable[string] `json:"tmpLocalRcvDir,omitempty"`
	PreTasks       []*Task          `json:"preTasks,omitempty"`
	PostTasks      []*Task          `json:"postTasks,omitempty"`
	ErrorTasks     []*Task          `json:"errorTasks,omitempty"`

	// Deprecated fields
	InPath   Nullable[string] `json:"inPath,omitempty"`   // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  Nullable[string] `json:"outPath,omitempty"`  // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath Nullable[string] `json:"workPath,omitempty"` // Deprecated: replaced by TmpLocalRcvDir
}

// OutRule is the JSON representation of a transfer rule in responses sent by
// the REST interface.
type OutRule struct {
	Name           string     `json:"name"`
	Comment        string     `json:"comment,omitempty"`
	IsSend         bool       `json:"isSend"`
	Path           string     `json:"path"`
	LocalDir       string     `json:"localDir,omitempty"`
	RemoteDir      string     `json:"remoteDir,omitempty"`
	TmpLocalRcvDir string     `json:"tmpLocalRcvDir,omitempty"`
	Authorized     RuleAccess `json:"authorized,omitempty"`
	PreTasks       []*Task    `json:"preTasks,omitempty"`
	PostTasks      []*Task    `json:"postTasks,omitempty"`
	ErrorTasks     []*Task    `json:"errorTasks,omitempty"`

	// Deprecated fields
	InPath   string `json:"inPath,omitempty"`   // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  string `json:"outPath,omitempty"`  // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath string `json:"workPath,omitempty"` // Deprecated: replaced by TmpLocalRcvDir
}
