package snmp

import (
	"errors"
	"math"
	"strings"

	"github.com/gosnmp/gosnmp"
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
