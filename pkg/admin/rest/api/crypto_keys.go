package api

type GetCryptoKeyRespObject struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Key  string `json:"key"`
}

type PostCryptoKeyReqObject struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Key  string `json:"key"`
}

type PatchCryptoKeyReqObject struct {
	Name Nullable[string] `json:"name"`
	Type Nullable[string] `json:"type"`
	Key  Nullable[string] `json:"key"`
}
