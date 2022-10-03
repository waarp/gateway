// Package file contains the declaration of the struct describing the JSON
// object used in backup files.
package file

import "encoding/json"

// Data is the top-level structure of the dump file.
type Data struct {
	Locals  []LocalAgent  `json:"locals"`
	Remotes []RemoteAgent `json:"remotes"`
	Rules   []Rule        `json:"rules"`
	Users   []User        `json:"users"`
}

// LocalAgent is the JSON struct representing a local server along with its
// accounts.
type LocalAgent struct {
	Name          string          `json:"name"`
	Protocol      string          `json:"protocol"`
	RootDir       string          `json:"rootDir,omitempty"`
	ReceiveDir    string          `json:"receiveDir,omitempty"`
	SendDir       string          `json:"sendDir,omitempty"`
	TmpReceiveDir string          `json:"tmpReceiveDir,omitempty"`
	Address       string          `json:"address"`
	Configuration json.RawMessage `json:"configuration"`
	Accounts      []LocalAccount  `json:"accounts"`
	Certs         []Certificate   `json:"certificates"` //nolint:tagliatelle // doesn't matter

	// Deprecated fields.
	Root    string `json:"root,omitempty"`    // Deprecated: replaced by Root
	InDir   string `json:"inDir,omitempty"`   // Deprecated: replaced by ReceiveDir
	OutDir  string `json:"outDir,omitempty"`  // Deprecated: replaced by SendDir
	WorkDir string `json:"workDir,omitempty"` // Deprecated: replaced by TmpReceiveDir
}

// LocalAccount is the JSON struct representing a local account.
type LocalAccount struct {
	Login        string        `json:"login"`
	Password     string        `json:"password,omitempty"`
	PasswordHash string        `json:"passwordHash,omitempty"`
	Certs        []Certificate `json:"certificates,omitempty"` //nolint:tagliatelle // doesn't matter
}

// RemoteAgent is the JSON struct representing a remote partner along with its
// accounts.
type RemoteAgent struct {
	Name          string          `json:"name"`
	Address       string          `json:"address"`
	Protocol      string          `json:"protocol"`
	Configuration json.RawMessage `json:"configuration"`
	Accounts      []RemoteAccount `json:"accounts"`
	Certs         []Certificate   `json:"certificates"` //nolint:tagliatelle // doesn't matter
}

// RemoteAccount is the JSON struct representing a local account.
type RemoteAccount struct {
	Login    string        `json:"login"`
	Password string        `json:"password,omitempty"`
	Certs    []Certificate `json:"certificates,omitempty"` //nolint:tagliatelle // doesn't matter
}

// Certificate is the JSON struct representing a certificate.
type Certificate struct {
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey,omitempty"`
	PrivateKey  string `json:"privateKey,omitempty"`
	Certificate string `json:"Certificate,omitempty"` //nolint:tagliatelle // doesn't matter
}

// Rule is the JSON struct representing a transfer rule.
type Rule struct {
	Name           string   `json:"name"`
	IsSend         bool     `json:"isSend"`
	Path           string   `json:"path"`
	LocalDir       string   `json:"localDir,omitempty"`
	RemoteDir      string   `json:"remoteDir,omitempty"`
	TmpLocalRcvDir string   `json:"tmpLocalRcvDir,omitempty"`
	Accesses       []string `json:"auth,omitempty"` //nolint:tagliatelle // doesn't matter
	Pre            []Task   `json:"pre,omitempty"`
	Post           []Task   `json:"post,omitempty"`
	Error          []Task   `json:"error,omitempty"`

	// Deprecated fields.
	InPath   string `json:"inPath,omitempty"`   // Deprecated: replaced by LocalDir & RemoteDir
	OutPath  string `json:"outPath,omitempty"`  // Deprecated: replaced by LocalDir & RemoteDir
	WorkPath string `json:"workPath,omitempty"` // Deprecated: replaced by TmpLocalRcvDir
}

// Task is the JSON struct representing a rule task.
type Task struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

// User is the JSON struct representing a gateway user.
type User struct {
	Username     string      `json:"username"`
	Password     string      `json:"password,omitempty"`
	PasswordHash string      `json:"passwordHash,omitempty"`
	Permissions  Permissions `json:"permissions"`
}

// Permissions if the JSON struct representing a gateway user's permissions.
// Each attribute represents a permission target, and its value defines the read,
// write & deletion permissions for that target in a chmod-like ('rwd') format.
type Permissions struct {
	Transfers string `json:"transfers"`
	Servers   string `json:"servers"`
	Partners  string `json:"partners"`
	Rules     string `json:"rules"`
	Users     string `json:"users"`
}
