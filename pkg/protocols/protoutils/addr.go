package protoutils

import (
	"net/netip"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func GetIP(addr string) string {
	ip, err := netip.ParseAddrPort(addr)
	if err != nil {
		return addr
	}

	return ip.Addr().String()
}

func GetRealAddress(overrides *conf.ConfigOverride, addr types.Address) string {
	if !addr.IsSet() {
		return ""
	}

	return overrides.GetRealAddress(addr.Host, utils.FormatUint(addr.Port))
}
