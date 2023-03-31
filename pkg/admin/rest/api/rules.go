package api

// AuthorizedRules represents a list of all the rules which an agent/account
// is allowed to use.
type AuthorizedRules struct {
	Sending   []string `json:"sending"`
	Reception []string `json:"reception"`
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
	*UptRule
	IsSend *bool `json:"isSend,omitempty"`
}

// UptRule is the JSON representation of a transfer rule in updated requests made to
// the REST interface.
type UptRule struct {
	Name           *string `json:"name,omitempty"`
	Comment        *string `json:"comment,omitempty"`
	Path           *string `json:"path,omitempty"`
	LocalDir       *string `json:"localDir,omitempty"`
	RemoteDir      *string `json:"remoteDir,omitempty"`
	TmpLocalRcvDir *string `json:"tmpLocalRcvDir,omitempty"`
	PreTasks       []*Task `json:"preTasks"`
	PostTasks      []*Task `json:"postTasks"`
	ErrorTasks     []*Task `json:"errorTasks"`

	// Deprecated fields
	InPath   *string `json:"inPath,omitempty"`   // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  *string `json:"outPath,omitempty"`  // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath *string `json:"workPath,omitempty"` // Deprecated: replaced by TmpLocalRcvDir
}

// OutRule is the JSON representation of a transfer rule in responses sent by
// the REST interface.
type OutRule struct {
	Name           string      `json:"name"`
	Comment        string      `json:"comment,omitempty"`
	IsSend         bool        `json:"isSend"`
	Path           string      `json:"path"`
	LocalDir       string      `json:"localDir,omitempty"`
	RemoteDir      string      `json:"remoteDir,omitempty"`
	TmpLocalRcvDir string      `json:"tmpLocalRcvDir,omitempty"`
	Authorized     *RuleAccess `json:"authorized,omitempty"`
	PreTasks       []*Task     `json:"preTasks,omitempty"`
	PostTasks      []*Task     `json:"postTasks,omitempty"`
	ErrorTasks     []*Task     `json:"errorTasks,omitempty"`

	// Deprecated fields
	InPath   string `json:"inPath,omitempty"`   // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  string `json:"outPath,omitempty"`  // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath string `json:"workPath,omitempty"` // Deprecated: replaced by TmpLocalRcvDir
}
