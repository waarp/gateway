package snmp

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
	"time"

	"github.com/gosnmp/gosnmp"
	snmplib "github.com/slayercat/GoSNMPServer"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const noTransferMsg = "ERROR: no transfer in database"

func getServerPDUs() []*snmplib.PDUValueControlItem {
	pdus := slices.Clone(ServerPDUs)
	for i := range pdus {
		pdus[i].OID += ".0"
	}

	return pdus
}

//nolint:gochecknoglobals //global var is better here
var ServerPDUs = []*snmplib.PDUValueControlItem{
	// ### App info PDUs ###
	{
		Document: "The application version",
		OID:      AppVersionOID,
		Type:     gosnmp.OctetString,
		OnGet: func() (any, error) {
			if analytics.GlobalService == nil {
				return nil, ErrNoAnalytics
			}

			return analytics.GlobalService.GetVersion(), nil
		},
	},
	{
		Document: "The time the server has been running (in 100th of a second)",
		OID:      AppUptimeOID,
		Type:     gosnmp.TimeTicks,
		OnGet: func() (any, error) {
			if analytics.GlobalService == nil {
				return nil, ErrNoAnalytics
			}

			return timeTicksSince(*analytics.GlobalService.StartTime.Load()), nil
		},
	},

	// ### RessourcesPDUs ###
	{
		Document: "The memory used by the application (in kB)",
		OID:      ResObjectMemUsageOID,
		Type:     gosnmp.Gauge32,
		OnGet: func() (any, error) {
			if analytics.GlobalService == nil {
				return nil, ErrNoAnalytics
			}

			//nolint:mnd //magic number needed here
			return uint32(analytics.GlobalService.GetUsedMemory() / 1000), nil
		},
	},
	{
		Document: "The number of available CPU cores on the machine",
		OID:      ResObjectNbCPUOID,
		Type:     gosnmp.Uinteger32,
		OnGet:    func() (any, error) { return uint32(runtime.NumCPU()), nil },
	},
	{
		Document: "The current number of goroutines",
		OID:      ResObjectNbGoroutinesOID,
		Type:     gosnmp.Gauge32,
		OnGet:    func() (any, error) { return uint32(runtime.NumGoroutine()), nil },
	},
	{
		Document: "The number of open incoming connections",
		OID:      ResObjectNbIncomingConnsOID,
		Type:     gosnmp.Gauge32,
		OnGet: func() (any, error) {
			if analytics.GlobalService == nil {
				return nil, ErrNoAnalytics
			}

			return uint32(analytics.GlobalService.OpenIncomingConnections.Load()), nil
		},
	},
	{
		Document: "The number of open outgoing connections",
		OID:      ResObjectNbOutgoingConnsOID,
		Type:     gosnmp.Gauge32,
		OnGet: func() (any, error) {
			if analytics.GlobalService == nil {
				return nil, ErrNoAnalytics
			}

			return uint32(analytics.GlobalService.OpenOutgoingConnections.Load()), nil
		},
	},
	/*{
		Document: "The total incoming bandwidth (in kB/s)",
		OID:      ResObjectIncomingTraficOID,
		Type:     gosnmp.Gauge32,
		OnGet: func() (any, error) { // TODO
			return nil, errors.New("not implemented")
		},
	}, {
		Document: "The total outgoing bandwidth (in kB/s)",
		OID:      ResObjectOutgoingTraficOID,
		Type:     gosnmp.Gauge32,
		OnGet: func() (any, error) { // TODO
			return nil, errors.New("not implemented")
		},
	},*/

	// ### Last transfer info ###
	{
		Document: "The last transfer's ID",
		OID:      LtObjectLastTransferOID,
		Type:     gosnmp.OctetString,
		OnGet: func() (any, error) {
			if analytics.GlobalService == nil {
				return nil, ErrNoAnalytics
			}

			trans, err := analytics.GlobalService.GetLastTransfer()
			if errors.Is(err, analytics.ErrNoTransfer) {
				return noTransferMsg, nil
			} else if err != nil {
				return nil, fmt.Errorf("failed to retrieve last transfer: %w", err)
			}

			return utils.FormatInt(trans.ID), nil
		},
	},
	{
		Document: "The last transfer's start time (in ISO 8601 format)",
		OID:      LtObjectLastTransferDateOID,
		Type:     gosnmp.OctetString,
		OnGet: func() (any, error) {
			if analytics.GlobalService == nil {
				return nil, ErrNoAnalytics
			}

			trans, err := analytics.GlobalService.GetLastTransfer()
			if errors.Is(err, analytics.ErrNoTransfer) {
				return noTransferMsg, nil
			} else if err != nil {
				return nil, fmt.Errorf("failed to retrieve last transfer: %w", err)
			}

			return trans.Start.Format(time.RFC3339Nano), nil
		},
	},
	{
		Document: "The last transfer's status",
		OID:      LtObjectLastTransferStatusOID,
		Type:     gosnmp.OctetString,
		OnGet: func() (any, error) {
			if analytics.GlobalService == nil {
				return nil, ErrNoAnalytics
			}

			trans, err := analytics.GlobalService.GetLastTransfer()
			if errors.Is(err, analytics.ErrNoTransfer) {
				return noTransferMsg, nil
			} else if err != nil {
				return nil, fmt.Errorf("failed to retrieve last transfer: %w", err)
			}

			return trans.Status, nil
		},
	},

	// ### Transfer statistics PDUs ###
	{
		Document: "The total number of transfers",
		OID:      TsObjectNbTotalTransfersOID,
		Type:     gosnmp.Counter64,
		OnGet:    mkTsGetFunc[uint64](""),
	},
	{
		Document: "The number of running transfers",
		OID:      TsObjectNbRunningTransfersOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTsGetFunc[uint32](types.StatusRunning),
	},
	{
		Document: "The number of in error transfers",
		OID:      TsObjectNbInErrorTransfersOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTsGetFunc[uint32](types.StatusError),
	},
	{
		Document: "The number of finished transfers",
		OID:      TsObjectNbFinishedTransfersOID,
		Type:     gosnmp.Counter64,
		OnGet:    mkTsGetFunc[uint64](types.StatusDone),
	},
	{
		Document: "The number of canceled transfers",
		OID:      TsObjectNbCanceledTransfersOID,
		Type:     gosnmp.Counter64,
		OnGet:    mkTsGetFunc[uint64](types.StatusCancelled),
	},

	// ### Transfer error statistics ###
	{
		Document: "The number of transfer with unknown errors",
		OID:      TesObjectNbUnknownErrOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTeGetFunc[uint32](types.TeUnknown),
	},
	{
		Document: "The number of transfer with internal errors",
		OID:      TesObjectNbInternalErrOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTeGetFunc[uint32](types.TeInternal),
	},
	{
		Document: "The number of transfer with connection errors",
		OID:      TesObjectNbConnectionErrOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTeGetFunc[uint32](types.TeConnection),
	},
	{
		Document: "The number of transfer with authentication errors",
		OID:      TesObjectNbAuthErrOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTeGetFunc[uint32](types.TeBadAuthentication),
	},
	{
		Document: "The number of transfer with authorization errors",
		OID:      TesObjectNbForbiddenErrOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTeGetFunc[uint32](types.TeForbidden),
	},
	{
		Document: "The number of transfer with file not found errors",
		OID:      TesObjectNbFileNotFoundErrOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTeGetFunc[uint32](types.TeFileNotFound),
	},
	{
		Document: "The number of transfer with external operations errors",
		OID:      TesObjectNbExternalOpErrOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTeGetFunc[uint32](types.TeExternalOperation),
	},
	{
		Document: "The number of transfer with finalization errors",
		OID:      TesObjectNbFinalizationErrOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTeGetFunc[uint32](types.TeFinalization),
	},
	{
		Document: "The number of transfer with file integrity errors",
		OID:      TesObjectNbIntegrityErrOID,
		Type:     gosnmp.Gauge32,
		OnGet:    mkTeGetFunc[uint32](types.TeIntegrity),
	},
}
