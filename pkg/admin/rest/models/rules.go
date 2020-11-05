package models

// AuthorizedRules represents a list of all the rules which an agent/account
// is allowed to use
type AuthorizedRules struct {
	Sending   []string `json:"sending"`
	Reception []string `json:"reception"`
}

// RuleAccess is the struct containing all the agents/accounts which are allowed
// to use a given rule.
type RuleAccess struct {
	LocalServers   []string            `json:"servers,omitempty"`
	RemotePartners []string            `json:"partners,omitempty"`
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
	Name       *string `json:"name,omitempty"`
	Comment    *string `json:"comment,omitempty"`
	Path       *string `json:"path,omitempty"`
	InPath     *string `json:"inPath,omitempty"`
	OutPath    *string `json:"outPath,omitempty"`
	WorkPath   *string `json:"workPath,omitempty"`
	PreTasks   []Task  `json:"preTasks,omitempty"`
	PostTasks  []Task  `json:"postTasks,omitempty"`
	ErrorTasks []Task  `json:"errorTasks,omitempty"`
}

// OutRule is the JSON representation of a transfer rule in responses sent by
// the REST interface.
type OutRule struct {
	Name       string      `json:"name"`
	Comment    string      `json:"comment,omitempty"`
	IsSend     bool        `json:"isSend"`
	Path       string      `json:"path"`
	InPath     string      `json:"inPath,omitempty"`
	OutPath    string      `json:"outPath,omitempty"`
	WorkPath   string      `json:"workPath,omitempty"`
	Authorized *RuleAccess `json:"authorized,omitempty"`
	PreTasks   []Task      `json:"preTasks,omitempty"`
	PostTasks  []Task      `json:"postTasks,omitempty"`
	ErrorTasks []Task      `json:"errorTasks,omitempty"`
}
