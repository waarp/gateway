package wg

import (
	"encoding/json"
	"errors"
	"fmt"
)

const ebicsProtocolName = "ebics"

var errEBICSFlagsRequireEBICSProtocol = errors.New("EBICS-specific flags require protocol ebics")

//nolint:lll // command line tags are intentionally explicit
type ebicsServerProtoConfigArgs struct {
	ProtocolVersion string  `long:"ebics-protocol-version" choice:"H005" description:"Override the EBICS protocol version stored in protoConfig" json:"-"`
	RequestTimeout  *int32  `long:"ebics-request-timeout" description:"Override the EBICS request timeout in seconds stored in protoConfig" json:"-"`
	MaxSegmentSize  *int64  `long:"ebics-max-segment-size" description:"Override the EBICS maximum segment size in bytes stored in protoConfig" json:"-"`
	AllowRecovery   *bool   `long:"ebics-allow-recovery" description:"Override whether EBICS recovery is allowed in protoConfig" json:"-"`
	MinTLSVersion   *string `long:"ebics-min-tls-version" choice:"v1.0" choice:"v1.1" choice:"v1.2" choice:"v1.3" description:"Override the minimum TLS version stored in protoConfig" json:"-"`
	VerifyBankKeys  *bool   `long:"ebics-verify-bank-keys" description:"Override whether EBICS bank key verification is enabled in protoConfig" json:"-"`
}

//nolint:lll // command line tags are intentionally explicit
type ebicsClientProtoConfigArgs struct {
	ProtocolVersion          string  `long:"ebics-protocol-version" choice:"H005" description:"Override the EBICS protocol version stored in protoConfig" json:"-"`
	EndpointURL              string  `long:"ebics-endpoint-url" description:"Override the EBICS bank endpoint URL stored in protoConfig" json:"-"`
	RequestTimeout           *int32  `long:"ebics-request-timeout" description:"Override the EBICS request timeout in seconds stored in protoConfig" json:"-"`
	MaxSegmentSize           *int64  `long:"ebics-max-segment-size" description:"Override the EBICS maximum segment size in bytes stored in protoConfig" json:"-"`
	AllowRecovery            *bool   `long:"ebics-allow-recovery" description:"Override whether EBICS recovery is allowed in protoConfig" json:"-"`
	MinTLSVersion            *string `long:"ebics-min-tls-version" choice:"v1.0" choice:"v1.1" choice:"v1.2" choice:"v1.3" description:"Override the minimum TLS version stored in protoConfig" json:"-"`
	VerifyBankKeys           *bool   `long:"ebics-verify-bank-keys" description:"Override whether EBICS bank key verification is enabled in protoConfig" json:"-"`
	DefaultOrderDataEncoding string  `long:"ebics-order-data-encoding" description:"Override the default EBICS order data encoding stored in protoConfig" json:"-"`
	ProfilePolicy            string  `long:"ebics-profile-policy" choice:"profile-required" choice:"profile-preferred" choice:"free-input-allowed" description:"Override the EBICS payload profile policy stored in protoConfig" json:"-"`
}

//nolint:lll // command line tags are intentionally explicit
type ebicsPartnerProtoConfigArgs struct {
	ProtocolVersion string `long:"ebics-protocol-version" choice:"H005" description:"Override the EBICS protocol version stored in protoConfig" json:"-"`
	EndpointURL     string `long:"ebics-endpoint-url" description:"Override the EBICS bank endpoint URL stored in protoConfig" json:"-"`
	HostID          string `long:"ebics-host-id" description:"Override the EBICS host ID stored in protoConfig" json:"-"`
	UseWSSRTN       *bool  `long:"ebics-use-wss-rtn" description:"Override whether the EBICS partner uses WSS RTN in protoConfig" json:"-"`
}

func (a *ebicsServerProtoConfigArgs) apply(
	proto string, conf map[string]confVal, requireEBICSProtocol bool,
) (map[string]confVal, error) {
	if !a.hasValues() {
		return conf, nil
	}

	if err := validateEBICSCLIProtocol(proto, requireEBICSProtocol); err != nil {
		return nil, err
	}

	dst := ensureProtoConfigMap(conf)
	if err := setProtoConfigLiteral(dst, "protocolVersion", a.ProtocolVersion); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "requestTimeout", a.RequestTimeout); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "maxSegmentSize", a.MaxSegmentSize); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "allowRecovery", a.AllowRecovery); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "minTLSVersion", a.MinTLSVersion); err != nil {
		return nil, err
	}
	err := setProtoConfigLiteral(dst, "verifyBankKeys", a.VerifyBankKeys)

	return dst, err
}

func (a *ebicsClientProtoConfigArgs) apply(
	proto string, conf map[string]confVal, requireEBICSProtocol bool,
) (map[string]confVal, error) {
	if !a.hasValues() {
		return conf, nil
	}

	if err := validateEBICSCLIProtocol(proto, requireEBICSProtocol); err != nil {
		return nil, err
	}

	dst := ensureProtoConfigMap(conf)
	if err := setProtoConfigLiteral(dst, "protocolVersion", a.ProtocolVersion); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "endpointURL", a.EndpointURL); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "requestTimeout", a.RequestTimeout); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "maxSegmentSize", a.MaxSegmentSize); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "allowRecovery", a.AllowRecovery); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "minTLSVersion", a.MinTLSVersion); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "verifyBankKeys", a.VerifyBankKeys); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "defaultOrderDataEncoding", a.DefaultOrderDataEncoding); err != nil {
		return nil, err
	}
	err := setProtoConfigLiteral(dst, "profilePolicy", a.ProfilePolicy)

	return dst, err
}

func (a *ebicsPartnerProtoConfigArgs) apply(
	proto string, conf map[string]confVal, requireEBICSProtocol bool,
) (map[string]confVal, error) {
	if !a.hasValues() {
		return conf, nil
	}

	if err := validateEBICSCLIProtocol(proto, requireEBICSProtocol); err != nil {
		return nil, err
	}

	dst := ensureProtoConfigMap(conf)
	if err := setProtoConfigLiteral(dst, "protocolVersion", a.ProtocolVersion); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "endpointURL", a.EndpointURL); err != nil {
		return nil, err
	}
	if err := setProtoConfigLiteral(dst, "hostID", a.HostID); err != nil {
		return nil, err
	}
	err := setProtoConfigLiteral(dst, "useWSSRTN", a.UseWSSRTN)

	return dst, err
}

func validateEBICSCLIProtocol(proto string, requireEBICSProtocol bool) error {
	if !requireEBICSProtocol {
		if proto == "" || proto == ebicsProtocolName {
			return nil
		}
	} else if proto == ebicsProtocolName {
		return nil
	}

	return errEBICSFlagsRequireEBICSProtocol
}

func ensureProtoConfigMap(conf map[string]confVal) map[string]confVal {
	if conf == nil {
		return make(map[string]confVal)
	}

	return conf
}

func setProtoConfigLiteral(dst map[string]confVal, key string, value any) error {
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil
		}
	case *string:
		if v == nil || *v == "" {
			return nil
		}
	case *int32:
		if v == nil {
			return nil
		}
	case *int64:
		if v == nil {
			return nil
		}
	case *bool:
		if v == nil {
			return nil
		}
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to encode protoConfig value %q: %w", key, err)
	}

	dst[key] = confVal(string(raw))

	return nil
}

func (a *ebicsServerProtoConfigArgs) hasValues() bool {
	return a.ProtocolVersion != "" || a.RequestTimeout != nil || a.MaxSegmentSize != nil ||
		a.AllowRecovery != nil || a.MinTLSVersion != nil || a.VerifyBankKeys != nil
}

func (a *ebicsClientProtoConfigArgs) hasValues() bool {
	return a.ProtocolVersion != "" || a.EndpointURL != "" || a.RequestTimeout != nil ||
		a.MaxSegmentSize != nil || a.AllowRecovery != nil || a.MinTLSVersion != nil ||
		a.VerifyBankKeys != nil || a.DefaultOrderDataEncoding != "" || a.ProfilePolicy != ""
}

func (a *ebicsPartnerProtoConfigArgs) hasValues() bool {
	return a.ProtocolVersion != "" || a.EndpointURL != "" || a.HostID != "" || a.UseWSSRTN != nil
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
