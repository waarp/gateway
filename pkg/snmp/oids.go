package snmp

// Standard SNMP OIDs.
const (
	EnterprisesOID = ".1.3.6.1.4.1"

	SNMPSysUpTimeOID = ".1.3.6.1.2.1.1.3.0"
	SNMPTrapOID      = ".1.3.6.1.6.3.1.1.4.1.0"
)

const WaarpEnterpriseOID = EnterprisesOID + ".66666"

// Gateway OIDs.
//
//nolint:gochecknoglobals //global vars are required here
var (
	AppOID = WaarpEnterpriseOID + ".77" // Other applications should change this value.

	NotifsOID  = AppOID + ".0"
	ObjectsOID = AppOID + ".1"
	GroupsOID  = AppOID + ".2"
)

// Traps OIDs.
//
//nolint:gochecknoglobals //global vars are required here
var (
	NotifObjectsOID = ObjectsOID + ".1"

	ServiceErrorObjectsOID = NotifObjectsOID + ".1"
	TransErrorObjectsOID   = NotifObjectsOID + ".2"
)

// Transfer Error OIDs
//
//nolint:gochecknoglobals //global vars are required here
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

// Service Error OIDs
//
//nolint:gochecknoglobals //global vars are required here
var (
	ServiceErrorNotifOID = NotifsOID + ".1"

	SeObjectNameOID  = ServiceErrorObjectsOID + ".1"
	SeObjectErrorOID = ServiceErrorObjectsOID + ".2"
)
