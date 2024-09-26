//go:build manual_test

package snmp

import (
	"errors"
	"testing"
	"time"

	"code.waarp.fr/lib/log"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestSNMPClient(t *testing.T) {
	Convey("Testing the SNMP client", t, func(c C) {
		db := database.TestDatabase(c)
		service := &Service{DB: db}

		snmpV2Monitor := MonitorConfig{
			Name:       "snmp-v2-monitor",
			Version:    Version2,
			UDPAddress: "localhost:162",
			Community:  "public",
		}
		So(db.Insert(&snmpV2Monitor).Run(), ShouldBeNil)

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
		So(db.Insert(&snmpV3Monitor).Run(), ShouldBeNil)

		So(service.Start(), ShouldBeNil)
		Reset(func() {
			ctx, cancel := testhelpers.ContextWithTimeout(5 * time.Second)
			defer cancel()
			So(service.Stop(ctx), ShouldBeNil)
		})

		service.Logger = testhelpers.TestLoggerWithLevel(c, ServiceName, log.LevelTrace)
		service.startTime = time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

		t.Run("Reporting a service error", func(t *testing.T) {
			testError := errors.New("test error message")
			require.NoError(t, service.sendServiceError("test-service", testError))
		})

		t.Run("Reporting a transfer error", func(t *testing.T) {
			t.SkipNow()
			transfer := &model.NormalizedTransferView{
				HistoryEntry: model.HistoryEntry{
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
					LocalPath: types.FSPath{
						Path: "/full/local/path/file.name",
					},
					RemotePath: "/full/remote/path/file.name",
					Filesize:   1000,
					Start:      time.Date(2024, 1, 1, 1, 1, 1, 0, time.UTC),
					Stop:       time.Date(2024, 1, 1, 2, 2, 2, 0, time.UTC),
					Status:     types.StatusCancelled,
					Step:       types.StepData,
					Progress:   500,
					ErrCode:    types.TeCanceled,
					ErrDetails: "transfer canceled by user",
				},
				IsTransfer: false,
			}

			require.NoError(t, service.sendTransferError(transfer))
		})
	})
}
