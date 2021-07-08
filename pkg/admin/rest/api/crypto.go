package api

// InCrypto is the JSON representation of a certificate in requests made to
// the REST interface.
type InCrypto struct {
	Name        *string `json:"name,omitempty"`
	PrivateKey  *string `json:"privateKey,omitempty"`
	PublicKey   *string `json:"publicKey,omitempty"`
	Certificate *string `json:"certificate,omitempty"`
}

// OutCrypto is the JSON representation of a certificate in responses sent by
// the REST interface.
type OutCrypto struct {
	Name        string `json:"name"`
	PrivateKey  string `json:"privateKey"`
	PublicKey   string `json:"publicKey"`
	Certificate string `json:"certificate"`
}
