package api

// InCrypto is the JSON representation of a certificate in requests made to
// the REST interface.
// Deprecated: replaced by InCred.
type InCrypto struct {
	Name        Nullable[string] `json:"name,omitempty"`
	PrivateKey  Nullable[string] `json:"privateKey,omitempty"`
	PublicKey   Nullable[string] `json:"publicKey,omitempty"`
	Certificate Nullable[string] `json:"certificate,omitempty"`
}

// OutCrypto is the JSON representation of a certificate in responses sent by
// the REST interface.
// Deprecated: replaced by OutServer.Credentials, OutPartner.Credentials &
// OutAccount.Credentials.
type OutCrypto struct {
	Name        string `json:"name"`
	PrivateKey  string `json:"privateKey,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	Certificate string `json:"certificate,omitempty"`
}
