package api

type GetCryptoKeyRespObject struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`
	Key  string `json:"key" yaml:"key"`
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
