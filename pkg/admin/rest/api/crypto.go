package api

// InCrypto is the JSON representation of a certificate in requests made to
// the REST interface.
//
// Deprecated: replaced by InCred.
type InCrypto struct {
	Name        Nullable[string] `json:"name,omitzero" yaml:"name,omitempty"`
	PrivateKey  Nullable[string] `json:"privateKey,omitzero" yaml:"privateKey,omitempty"`
	PublicKey   Nullable[string] `json:"publicKey,omitzero" yaml:"publicKey,omitempty"`
	Certificate Nullable[string] `json:"certificate,omitzero" yaml:"certificate,omitempty"`
}

// OutCrypto is the JSON representation of a certificate in responses sent by
// the REST interface.
//
// Deprecated: replaced by OutServer.Credentials, OutPartner.Credentials &
// OutAccount.Credentials.
type OutCrypto struct {
	Name        string `json:"name" yaml:"name"`
	PrivateKey  string `json:"privateKey,omitempty" yaml:"privateKey,omitempty"`
	PublicKey   string `json:"publicKey,omitempty" yaml:"publicKey,omitempty"`
	Certificate string `json:"certificate,omitempty" yaml:"certificate,omitempty"`
}
