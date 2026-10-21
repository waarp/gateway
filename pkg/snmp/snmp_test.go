package snmp

import (
	"errors"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestSNMPClient(t *testing.T) {
	t.Parallel()

	trapsTimeout = time.Second
	trapsRetries = 0

	v2Traps, v2Addr := makeTrapListener(t, gosnmp.Version2c)

	t.Run("Testing the SNMP client", func(t *testing.T) {
		db := dbtest.TestDatabase(t)
		service := &Service{DB: db}

		snmpV3Monitor := MonitorConfig{
			Name:           "snmp-v3-monitor",
			Version:        Version3,
			UDPAddress:     "127.0.0.1:162",
			UseInforms:     true,
			SNMPv3Security: V3SecurityAuthPriv,
			AuthProtocol:   "SHA",
			PrivProtocol:   "AES",
			AuthUsername:   "informtest",
			AuthPassphrase: "mypassword",
			PrivPassphrase: "mypassword",
		}
		require.NoError(t, db.Insert(&snmpV3Monitor).Run())

		snmpV2Monitor := MonitorConfig{
			Name:       "snmp-v2-monitor",
			Version:    Version2,
			UDPAddress: v2Addr.String(),
			Community:  "public",
		}
		require.NoError(t, db.Insert(&snmpV2Monitor).Run())

		require.NoError(t, service.Start())
		t.Cleanup(func() {
			ctx, cancel := testhelpers.ContextWithTimeout(5 * time.Second)
			defer cancel()
			require.NoError(t, service.Stop(ctx))
		})

		service.Logger = logtest.GetTestLogger(t,
			logtest.WithName(ServiceName), logtest.WithLevel("TRACE"))
		service.startTime = time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

		t.Run("Reporting a service error", func(t *testing.T) {
			testError := errors.New("test error message")
			require.ErrorContains(t, service.sendServiceError("test-service", testError),
				"failed to send SNMP trap")

			expectedVars := []gosnmp.SnmpPDU{{
				Name:  SNMPTrapOID,
				Type:  gosnmp.ObjectIdentifier,
				Value: ServiceErrorNotifOID,
			}, {
				Name:  InstanceNameOID,
				Type:  gosnmp.OctetString,
				Value: []byte(db.Config.GatewayName),
			}}
			assertReceivedVar(t, v2Traps, expectedVars...)
		})

		t.Run("Reporting a transfer error", func(t *testing.T) {
			transfer := &model.NormalizedTransferView{
				ID:               1234,
				RemoteTransferID: "abcd",
				IsServer:         false,
				IsSend:           true,
				Rule:             "ruleName",
				Account:          "accountLogin",
				Agent:            "partnerName",
				Client:           "clientName",
				Protocol:         "test_proto",
				SrcFilename:      "src/file.name",
				DestFilename:     "dst/file.name",
				LocalPath:        "/full/local/path/file.name",
				RemotePath:       "/full/remote/path/file.name",
				Filesize:         1000,
				Start:            time.Date(2024, 1, 1, 1, 1, 1, 0, time.UTC),
				Stop:             time.Date(2024, 1, 1, 2, 2, 2, 0, time.UTC),
				Status:           types.StatusCancelled,
				Step:             types.StepData,
				Progress:         500,
				ErrCode:          types.TeCanceled,
				ErrDetails:       "transfer canceled by user",
				IsTransfer:       false,
			}

			require.ErrorContains(t, service.sendTransferError(transfer),
				"failed to send SNMP trap")

			expectedVars := []gosnmp.SnmpPDU{{
				Name:  SNMPTrapOID,
				Type:  gosnmp.ObjectIdentifier,
				Value: TransferErrorNotifOID,
			}, {
				Name:  InstanceNameOID,
				Type:  gosnmp.OctetString,
				Value: []byte(db.Config.GatewayName),
			}}
			assertReceivedVar(t, v2Traps, expectedVars...)
		})
	})
}
