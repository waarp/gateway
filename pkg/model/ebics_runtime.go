package model

import "code.waarp.fr/apps/gateway/gateway/pkg/conf"

// EbicsGatewayOwnerForRuntime exposes the current gateway owner name to runtime packages.
func EbicsGatewayOwnerForRuntime() string {
	return conf.GlobalConfig.GatewayName
}

// EbicsOperationDirectionInboundForRuntime exposes the inbound direction to runtime packages.
func EbicsOperationDirectionInboundForRuntime() string {
	return ebicsOperationDirectionInbound
}

// EbicsOperationDirectionOutboundForRuntime exposes the outbound direction to runtime packages.
func EbicsOperationDirectionOutboundForRuntime() string {
	return ebicsOperationDirectionOutbound
}

// EbicsOperationDirectionInternalForRuntime exposes the internal direction to runtime packages.
func EbicsOperationDirectionInternalForRuntime() string {
	return ebicsOperationDirectionInternal
}

// EbicsTransportModeSyncForRuntime exposes the synchronous transport mode to runtime packages.
func EbicsTransportModeSyncForRuntime() string {
	return ebicsTransportModeSync
}

// EbicsOperationTypeReportingForRuntime exposes the reporting operation type to runtime packages.
func EbicsOperationTypeReportingForRuntime() string {
	return ebicsOperationTypeReporting
}

// EbicsOperationStatusRunningForRuntime exposes the running status to runtime packages.
func EbicsOperationStatusRunningForRuntime() string {
	return ebicsOperationStatusRunning
}
