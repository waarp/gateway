package snmp

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
	"golang.org/x/exp/constraints"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var ErrInvalidSNMPVersion = errors.New("invalid SNMP version")

func toSNMPVersion(str string) (gosnmp.SnmpVersion, error) {
	switch {
	case strings.EqualFold(str, "SNMPv1"):
		return 0, errors.New("SNMPv1 is not supported") //nolint:goerr113 //too specific
	case strings.EqualFold(str, Version2) || strings.EqualFold(str, Version2c):
		return gosnmp.Version2c, nil
	case strings.EqualFold(str, Version3):
		return gosnmp.Version3, nil
	default:
		return 0, ErrInvalidSNMPVersion
	}
}

const unknownV3MsgFlag gosnmp.SnmpV3MsgFlags = math.MaxUint8

func getSNMPv3MsgFlags(sec string) gosnmp.SnmpV3MsgFlags {
	switch sec {
	case "", V3SecurityNoAuthNoPriv:
		return gosnmp.NoAuthNoPriv
	case V3SecurityAuthNoPriv:
		return gosnmp.AuthNoPriv
	case V3SecurityAuthPriv:
		return gosnmp.AuthPriv
	default:
		return unknownV3MsgFlag
	}
}

func getAuthProtocol(name string) gosnmp.SnmpV3AuthProtocol {
	switch name {
	case AuthMD5:
		return gosnmp.MD5
	case AuthSHA:
		return gosnmp.SHA
	case AuthSHA224:
		return gosnmp.SHA224
	case AuthSHA256:
		return gosnmp.SHA256
	case AuthSHA384:
		return gosnmp.SHA384
	case AuthSHA512:
		return gosnmp.SHA512
	case "":
		return gosnmp.NoAuth
	default:
		return 0
	}
}

func getPrivProtocol(name string) gosnmp.SnmpV3PrivProtocol {
	switch name {
	case PrivDES:
		return gosnmp.DES
	case PrivAES:
		return gosnmp.AES
	case PrivAES192:
		return gosnmp.AES192
	case PrivAES256:
		return gosnmp.AES256
	case PrivAES192C:
		return gosnmp.AES192C
	case PrivAES256C:
		return gosnmp.AES256C
	case "":
		return gosnmp.NoPriv
	default:
		return 0
	}
}

func timeTicksSince(t time.Time) uint32 {
	const milliSecTo100thSecRatio = 10

	//nolint:gosec //SNMP requires we convert to uint32, no other option here
	return uint32(time.Since(t).Milliseconds() / milliSecTo100thSecRatio)
}

func mkTsGetFunc[T constraints.Integer](status types.TransferStatus) func() (any, error) {
	return func() (any, error) {
		if analytics.GlobalService == nil {
			return nil, ErrNoAnalytics
		}

		count, err := analytics.GlobalService.CountTransferWithStatus(status)
		if err != nil {
			return nil, fmt.Errorf("failed to count transfers with status %q: %w", status, err)
		}

		return T(count), nil
	}
}

func mkTeGetFunc[T constraints.Integer](tec types.TransferErrorCode) func() (any, error) {
	return func() (any, error) {
		if analytics.GlobalService == nil {
			return nil, ErrNoAnalytics
		}

		count, err := analytics.GlobalService.CountTransfersWithErrorCode(tec)
		if err != nil {
			return nil, fmt.Errorf("failed to count transfers with error code %q: %w",
				tec.String(), err)
		}

		return T(count), nil
	}
}
