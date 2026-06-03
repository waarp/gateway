package protoutils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //this is basically a constant
var TLSCiphers = map[string]*tls.CipherSuite{}

//nolint:gochecknoinits //this is needed to initialize the TLSCiphers map
func init() {
	for _, c := range tls.CipherSuites() {
		TLSCiphers[c.Name] = c
	}

	for _, c := range tls.InsecureCipherSuites() {
		TLSCiphers[c.Name] = c
	}
}

type TLSCiphersList []uint16

//nolint:wrapcheck //wrapping adds nothing here
func (t *TLSCiphersList) UnmarshalJSON(bytes []byte) error {
	var names []string
	if err := json.Unmarshal(bytes, &names); err != nil {
		return err
	}

	for _, name := range names {
		cipher := TLSCiphers[name]
		if cipher == nil {
			return UnknownCipherError(name)
		}

		*t = append(*t, cipher.ID)
	}

	return nil
}

//nolint:wrapcheck //wrapping adds nothing here
func (t *TLSCiphersList) MarshalJSON() ([]byte, error) {
	names := make([]string, len(*t))
	for i, id := range *t {
		names[i] = tls.CipherSuiteName(id)
	}

	return json.Marshal(names)
}

type UnknownCipherError string

func (e UnknownCipherError) Error() string {
	return fmt.Sprintf("unknown TLS cipher name %q", string(e))
}

func GetTLSCiphers(config map[string]any) []uint16 {
	var ids TLSCiphersList
	if err := utils.JSONConvert(config["cipherSuites"], &ids); err != nil {
		return nil
	}

	return ids
}
