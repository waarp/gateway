// Package file contains the declaration of the struct describing the JSON
// object used in backup files.
package file

import "encoding/json"

// Data is the top-level structure of the dump file.
type Data struct {
	Locals  []LocalAgent  `json:"locals"`
	Remotes []RemoteAgent `json:"remotes"`
	Rules   []Rule        `json:"rules"`
}

// LocalAgent is the JSON struct representing a local server along with its
// accounts.
type LocalAgent struct {
	Name          string          `json:"name"`
	Protocol      string          `json:"protocol"`
	Root          string          `json:"root,omitempty"`
	InDir         string          `json:"inDir,omitempty"`
	OutDir        string          `json:"outDir,omitempty"`
	WorkDir       string          `json:"workDir,omitempty"`
	Address       string          `json:"address"`
	Configuration json.RawMessage `json:"configuration"`
	Accounts      []LocalAccount  `json:"accounts"`
	Certs         []Certificate   `json:"certificates"` //nolint:tagliatelle // doesn't matter
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
	Name     string   `json:"name"`
	IsSend   bool     `json:"isSend"`
	Path     string   `json:"path"`
	InPath   string   `json:"inPath,omitempty"`
	OutPath  string   `json:"outPath,omitempty"`
	WorkPath string   `json:"workPath,omitempty"`
	Accesses []string `json:"auth,omitempty"` //nolint:tagliatelle // doesn't matter
	Pre      []Task   `json:"pre,omitempty"`
	Post     []Task   `json:"post,omitempty"`
	Error    []Task   `json:"error,omitempty"`
}

// Task is the JSON struct representing a rule task.
type Task struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}
