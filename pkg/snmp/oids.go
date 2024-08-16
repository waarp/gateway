package snmp

// Standard SNMP OIDs.
const (
	EnterprisesOID = ".1.3.6.1.4.1"

	SNMPSysUpTimeOID = ".1.3.6.1.2.1.1.3.0"
	SNMPTrapOID      = ".1.3.6.1.6.3.1.1.4.1.0"
)

const WaarpEnterpriseOID = EnterprisesOID + ".66666"

// Gateway OIDs.
var (
	AppOID = WaarpEnterpriseOID + ".77" // Other applications should change this value.

	NotifsOID  = AppOID + ".0"
	ObjectsOID = AppOID + ".1"
	GroupsOID  = AppOID + ".2"
)

// Traps OIDs.
var (
	NotifObjectsOID = ObjectsOID + ".1"

	ServiceErrorObjectsOID = NotifObjectsOID + ".1"
	TransErrorObjectsOID   = NotifObjectsOID + ".2"
)

// Application info OIDs.
var (
	AppInfoOID = ObjectsOID + ".2"

	AppVersionOID = AppInfoOID + ".1"
	AppUptimeOID  = AppInfoOID + ".2"
)

// Ressources info OIDs.
var (
	RessourcesInfoOID = ObjectsOID + ".3"

	ResObjectMemUsageOID        = RessourcesInfoOID + ".1"
	ResObjectNbCPUOID           = RessourcesInfoOID + ".2"
	ResObjectNbGoroutinesOID    = RessourcesInfoOID + ".3"
	ResObjectNbIncomingConnsOID = RessourcesInfoOID + ".4"
	ResObjectNbOutgoingConnsOID = RessourcesInfoOID + ".5"
	ResObjectIncomingTraficOID  = RessourcesInfoOID + ".6"
	ResObjectOutgoingTraficOID  = RessourcesInfoOID + ".7"
)

// Last transfer OIDs.
var (
	LastTransStatsOID = ObjectsOID + ".4"

	LtObjectLastTransferOID       = LastTransStatsOID + ".1"
	LtObjectLastTransferDateOID   = LastTransStatsOID + ".2"
	LtObjectLastTransferStatusOID = LastTransStatsOID + ".3"
)

// Transfer statistics OIDs.
var (
	TransStatsOID = ObjectsOID + ".5"

	TsObjectNbTotalTransfersOID    = TransStatsOID + ".1"
	TsObjectNbRunningTransfersOID  = TransStatsOID + ".2"
	TsObjectNbInErrorTransfersOID  = TransStatsOID + ".3"
	TsObjectNbFinishedTransfersOID = TransStatsOID + ".4"
	TsObjectNbCanceledTransfersOID = TransStatsOID + ".5"
)

// Transfer error starts OIDs.
var (
	TransErrorStatsOID = ObjectsOID + ".6"

	TesObjectNbUnknownErrOID      = TransErrorStatsOID + ".1"
	TesObjectNbInternalErrOID     = TransErrorStatsOID + ".2"
	TesObjectNbConnectionErrOID   = TransErrorStatsOID + ".3"
	TesObjectNbAuthErrOID         = TransErrorStatsOID + ".4"
	TesObjectNbForbiddenErrOID    = TransErrorStatsOID + ".5"
	TesObjectNbFileNotFoundErrOID = TransErrorStatsOID + ".6"
	TesObjectNbExternalOpErrOID   = TransErrorStatsOID + ".7"
	TesObjectNbFinalizationErrOID = TransErrorStatsOID + ".8"
	TesObjectNbIntegrityErrOID    = TransErrorStatsOID + ".9"
)

// Transfer Error OIDs.
var (
	TransferErrorNotifOID = NotifsOID + ".2"

	TeObjectTransOID      = TransErrorObjectsOID + ".1"
	TeObjectRemoteOID     = TransErrorObjectsOID + ".2"
	TeObjectRuleNameOID   = TransErrorObjectsOID + ".3"
	TeObjectClientNameOID = TransErrorObjectsOID + ".4"
	TeObjectRequesterOID  = TransErrorObjectsOID + ".5"
	TeObjectRequestedOID  = TransErrorObjectsOID + ".6"
	TeObjectFilenameOID   = TransErrorObjectsOID + ".7"
	TeObjectDateOID       = TransErrorObjectsOID + ".8"
	TeObjectErrCodeOID    = TransErrorObjectsOID + ".9"
	TeObjectErrDetailsOID = TransErrorObjectsOID + ".10"
)

// Service Error OIDs.
var (
	ServiceErrorNotifOID = NotifsOID + ".1"

	SeObjectNameOID  = ServiceErrorObjectsOID + ".1"
	SeObjectErrorOID = ServiceErrorObjectsOID + ".2"
)
