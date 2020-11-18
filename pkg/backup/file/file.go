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
	Root          string          `json:"root"`
	InDir         string          `json:"inDir"`
	OutDir        string          `json:"outDir"`
	WorkDir       string          `json:"workDir"`
	Address       string          `json:"address"`
	Configuration json.RawMessage `json:"configuration"`
	Accounts      []LocalAccount  `json:"accounts"`
	Certs         []Certificate   `json:"certificates"`
}

// LocalAccount is the JSON struct representing a local account.
type LocalAccount struct {
	Login    string        `json:"login"`
	Password string        `json:"password"`
	Certs    []Certificate `json:"certificates"`
}

// RemoteAgent is the JSON struct representing a remote partner along with its
// accounts.
type RemoteAgent struct {
	Name          string          `json:"name"`
	Address       string          `json:"address"`
	Protocol      string          `json:"protocol"`
	Configuration json.RawMessage `json:"configuration"`
	Accounts      []RemoteAccount `json:"accounts"`
	Certs         []Certificate   `json:"certificates"`
}

// RemoteAccount is the JSON struct representing a local account.
type RemoteAccount struct {
	Login    string        `json:"login"`
	Password string        `json:"password"`
	Certs    []Certificate `json:"certificates"`
}

// Certificate is the JSON struct representing a certificate.
type Certificate struct {
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey"`
	PrivateKey  string `json:"privateKey"`
	Certificate string `json:"Certificate"`
}

// Rule is the JSON struct representing a transfer rule.
type Rule struct {
	Name     string   `json:"name"`
	IsSend   bool     `json:"isSend"`
	Path     string   `json:"path"`
	InPath   string   `json:"inPath"`
	OutPath  string   `json:"outPath"`
	WorkPath string   `json:"workPath"`
	Accesses []string `json:"auth"`
	Pre      []Task   `json:"pre"`
	Post     []Task   `json:"post"`
	Error    []Task   `json:"error"`
}

// Task is the JSON struct representing a rule task.
type Task struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}
