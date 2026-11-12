package api

// GetCryptoKeyRespObject is what the API says of a cryptographic key. The value
// of a secret key (AES, HMAC, PGP private) is written through the API and never
// read back.
type GetCryptoKeyRespObject struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`

	// Key is the value of a PGP public key. A secret key comes without it.
	Key string `json:"key,omitempty" yaml:"key,omitempty"`

	// Fingerprint and Identities are what a PGP key states of itself. They are
	// public, for a private key as for a public one.
	Fingerprint string   `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	Identities  []string `json:"identities,omitempty" yaml:"identities,omitempty"`
}

type PostCryptoKeyReqObject struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`
	Key  string `json:"key" yaml:"key"`
}

type PatchCryptoKeyReqObject struct {
	Name Nullable[string] `json:"name" yaml:"name"`
	Type Nullable[string] `json:"type" yaml:"type"`
	Key  Nullable[string] `json:"key" yaml:"key"`
}
