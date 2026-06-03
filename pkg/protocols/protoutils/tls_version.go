package protoutils

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type TLSVersion int

const (
	TLSv10 = "v1.0"
	TLSv11 = "v1.1"
	TLSv12 = "v1.2"
	TLSv13 = "v1.3"

	DefaultTLSVersion = tls.VersionTLS12
)

func GetMinTLSVersion(m map[string]any) uint16 {
	var version TLSVersion
	if err := utils.JSONConvert(m["minTLSVersion"], &version); err != nil {
		return DefaultTLSVersion
	}

	return version.TLS()
}

func TLSVersionFromString(v string) (TLSVersion, error) {
	switch v {
	case "", "null":
		return DefaultTLSVersion, nil
	case TLSv10:
		return tls.VersionTLS10, nil
	case TLSv11:
		return tls.VersionTLS11, nil
	case TLSv12:
		return tls.VersionTLS12, nil
	case TLSv13:
		return tls.VersionTLS13, nil
	default:
		return 0, UnsupportedTLSVersionError(v)
	}
}

func (t TLSVersion) TLS() uint16 { return uint16(t) }

func (t TLSVersion) String() string {
	switch t {
	case 0:
		return TLSVersion(DefaultTLSVersion).String()
	case tls.VersionTLS10:
		return TLSv10
	case tls.VersionTLS11:
		return TLSv11
	case tls.VersionTLS12:
		return TLSv12
	case tls.VersionTLS13:
		return TLSv13
	default:
		return fmt.Sprintf("<unknown TLS version %d>", t)
	}
}

func (t *TLSVersion) UnmarshalJSON(b []byte) error {
	var v string
	if err := json.Unmarshal(b, &v); err != nil {
		return err //nolint:wrapcheck //no need to wrap here
	}

	var err error
	*t, err = TLSVersionFromString(v)

	return err
}

func (t TLSVersion) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(t.String())), nil
}

type UnsupportedTLSVersionError string

func (e UnsupportedTLSVersionError) Error() string {
	return fmt.Sprintf("unknown TLS version %q (supported TLS versions: %s)", string(e),
		strings.Join([]string{TLSv10, TLSv11, TLSv12, TLSv13}, ", "))
}
