// Package backup provides two methods too generate export of the database for
// backup or migration purpose, and to import a previous dump in order to
// restore the database.
package backup

import "encoding/json"

// data is the top-level structure of the dump file.
type data struct {
	Locals  []localAgent  `json:"locals"`
	Remotes []remoteAgent `json:"remotes"`
	Rules   []rule        `json:"rules"`
}

type localAgent struct {
	Name          string          `json:"name"`
	Protocol      string          `json:"protocol"`
	Root          string          `json:"root"`
	InDir         string          `json:"inDir"`
	OutDir        string          `json:"outDir"`
	WorkDir       string          `json:"workDir"`
	Address       string          `json:"address"`
	Configuration json.RawMessage `json:"configuration"`
	Accounts      []localAccount  `json:"accounts"`
	Certs         []certificate   `json:"certificates"`
}

type localAccount struct {
	Login    string        `json:"login"`
	Password string        `json:"password"`
	Certs    []certificate `json:"certificates"`
}

type remoteAgent struct {
	Name          string          `json:"name"`
	Address       string          `json:"address"`
	Protocol      string          `json:"protocol"`
	Configuration json.RawMessage `json:"configuration"`
	Accounts      []remoteAccount `json:"accounts"`
	Certs         []certificate   `json:"certificates"`
}

type remoteAccount struct {
	Login    string        `json:"login"`
	Password string        `json:"password"`
	Certs    []certificate `json:"certificates"`
}

type certificate struct {
	Name        string `json:"name"`
	PublicKey   string `json:"publicKey"`
	PrivateKey  string `json:"privateKey"`
	Certificate string `json:"certificate"`
}

type rule struct {
	Name     string     `json:"name"`
	IsSend   bool       `json:"isSend"`
	Path     string     `json:"path"`
	InPath   string     `json:"inPath"`
	OutPath  string     `json:"outPath"`
	WorkPath string     `json:"workPath"`
	Accesses []string   `json:"auth"`
	Pre      []ruleTask `json:"pre"`
	Post     []ruleTask `json:"post"`
	Error    []ruleTask `json:"error"`
}

type ruleTask struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}
