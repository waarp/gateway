package api

type GetPGPKeyRespObject struct {
	Name       string `json:"name"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

type PostPGPKeyReqObject struct {
	Name       string `json:"name"`
	PublicKey  string `json:"publicKey"`
	PrivateKey string `json:"privateKey"`
}

type PatchPGPKeyReqObject struct {
	Name       Nullable[string] `json:"name"`
	PublicKey  Nullable[string] `json:"publicKey"`
	PrivateKey Nullable[string] `json:"privateKey"`
}
