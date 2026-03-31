package ebics

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	libebics "code.waarp.fr/lib/ebics/ebics"
	liborders "code.waarp.fr/lib/ebics/ebics/orders"
	libreturncode "code.waarp.fr/lib/ebics/ebics/returncode"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	ebicsruntime "code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/ebics/runtime"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestMatchOptionalProfileField(t *testing.T) {
	t.Run("wildcard field does not change score", func(t *testing.T) {
		score := 2

		ok := matchOptionalProfileField("", "pain.001", &score)

		require.True(t, ok)
		require.Equal(t, 2, score)
	})

	t.Run("exact match increments score", func(t *testing.T) {
		score := 1

		ok := matchOptionalProfileField("pain.001", "pain.001", &score)

		require.True(t, ok)
		require.Equal(t, 2, score)
	})

	t.Run("mismatch rejects profile", func(t *testing.T) {
		score := 1

		ok := matchOptionalProfileField("pain.001", "camt.053", &score)

		require.False(t, ok)
		require.Equal(t, 1, score)
	})
}

func TestScorePayloadProfile(t *testing.T) {
	service := ebicsruntime.PayloadServiceRef{
		OrderType:     "BTU",
		ServiceName:   "MCT",
		ServiceOption: "SDD",
		Scope:         "FR",
		MsgName:       "pain.001",
		ContainerType: "ZIP",
	}

	t.Run("nil profile is rejected", func(t *testing.T) {
		score, ok := scorePayloadProfile(nil, service)

		require.False(t, ok)
		require.Zero(t, score)
	})

	t.Run("exact match scores every constrained field", func(t *testing.T) {
		score, ok := scorePayloadProfile(&model.EbicsPayloadProfile{
			ServiceName:   "MCT",
			ServiceOption: "SDD",
			Scope:         "FR",
			MsgName:       "pain.001",
			ContainerType: "ZIP",
		}, service)

		require.True(t, ok)
		require.Equal(t, 5, score)
	})

	t.Run("wildcard fields remain eligible", func(t *testing.T) {
		score, ok := scorePayloadProfile(&model.EbicsPayloadProfile{
			ServiceName: "MCT",
			Scope:       "FR",
		}, service)

		require.True(t, ok)
		require.Equal(t, 2, score)
	})

	t.Run("single mismatch rejects profile", func(t *testing.T) {
		score, ok := scorePayloadProfile(&model.EbicsPayloadProfile{
			ServiceName: "MCT",
			Scope:       "DE",
		}, service)

		require.False(t, ok)
		require.Zero(t, score)
	})
}

func TestMatchPayloadProfile(t *testing.T) {
	db := dbtest.TestDatabase(t)
	router := newPayloadOrderRouter(db, nil)

	receiveRule := insertTestRule(t, db, "ebics-receive", false)
	insertTestPayloadProfile(t, db, &model.EbicsPayloadProfile{
		Name:          "generic",
		OrderType:     "BTU",
		Direction:     "UPLOAD",
		ServiceName:   "MCT",
		DefaultRuleID: utils.NewNullInt64(receiveRule.ID),
		IsEnabled:     true,
	})
	insertTestPayloadProfile(t, db, &model.EbicsPayloadProfile{
		Name:          "specific",
		OrderType:     "BTU",
		Direction:     "UPLOAD",
		ServiceName:   "MCT",
		Scope:         "FR",
		MsgName:       "pain.001",
		DefaultRuleID: utils.NewNullInt64(receiveRule.ID),
		IsEnabled:     true,
	})

	profile, rule, err := router.matchPayloadProfile("BTU", ebicsruntime.PayloadServiceRef{
		OrderType:   "BTU",
		ServiceName: "MCT",
		Scope:       "FR",
		MsgName:     "pain.001",
	})

	require.NoError(t, err)
	require.NotNil(t, profile)
	require.NotNil(t, rule)
	require.Equal(t, "specific", profile.Name)
	require.Equal(t, receiveRule.ID, rule.ID)
	require.Equal(t, receiveRule.Name, profile.DefaultRuleName)
}

func TestMatchPayloadProfileRejectsAmbiguousProfiles(t *testing.T) {
	db := dbtest.TestDatabase(t)
	router := newPayloadOrderRouter(db, nil)

	receiveRule := insertTestRule(t, db, "ebics-ambiguous", false)
	for _, name := range []string{"profile-a", "profile-b"} {
		insertTestPayloadProfile(t, db, &model.EbicsPayloadProfile{
			Name:          name,
			OrderType:     "BTU",
			Direction:     "UPLOAD",
			ServiceName:   "MCT",
			Scope:         "FR",
			DefaultRuleID: utils.NewNullInt64(receiveRule.ID),
			IsEnabled:     true,
		})
	}

	_, _, err := router.matchPayloadProfile("BTU", ebicsruntime.PayloadServiceRef{
		OrderType:   "BTU",
		ServiceName: "MCT",
		Scope:       "FR",
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "ambiguous EBICS payload profile match")

	code, ok := libreturncode.FromError(err)
	require.True(t, ok)
	require.Equal(t, libreturncode.CodeInvalidOrderParams, code)
}

func TestValidateRouterContract(t *testing.T) {
	resolved := &ebicsruntime.ResolvedPayloadRequest{ProfileName: "payments-fr"}

	t.Run("matched contract is accepted", func(t *testing.T) {
		err := validateRouterContract(resolved, &ebicsruntime.ContractValidationResult{
			Status: "MATCHED",
		})

		require.NoError(t, err)
	})

	t.Run("missing validation base becomes invalid order params", func(t *testing.T) {
		err := validateRouterContract(resolved, &ebicsruntime.ContractValidationResult{
			Status:  "NO_VALIDATION_BASE",
			Message: "no active contract",
		})

		require.Error(t, err)
		require.ErrorContains(t, err, "no active contract")

		code, ok := libreturncode.FromError(err)
		require.True(t, ok)
		require.Equal(t, libreturncode.CodeInvalidOrderParams, code)
	})

	t.Run("missing matching item keeps profile name in message", func(t *testing.T) {
		err := validateRouterContract(resolved, &ebicsruntime.ContractValidationResult{
			Status:  "NO_MATCHING_ITEM",
			Message: "service not allowed",
		})

		require.Error(t, err)
		require.ErrorContains(t, err, "payments-fr")
		require.ErrorContains(t, err, "service not allowed")

		code, ok := libreturncode.FromError(err)
		require.True(t, ok)
		require.Equal(t, libreturncode.CodeInvalidOrderParams, code)
	})

	t.Run("unsupported status becomes processing error", func(t *testing.T) {
		err := validateRouterContract(resolved, &ebicsruntime.ContractValidationResult{
			Status: "BROKEN",
		})

		require.Error(t, err)

		code, ok := libreturncode.FromError(err)
		require.True(t, ok)
		require.Equal(t, libreturncode.CodeProcessingError, code)
	})

	t.Run("nil validation becomes processing error", func(t *testing.T) {
		err := validateRouterContract(resolved, nil)

		require.Error(t, err)

		code, ok := libreturncode.FromError(err)
		require.True(t, ok)
		require.Equal(t, libreturncode.CodeProcessingError, code)
	})
}

func TestDeriveIncomingFilename(t *testing.T) {
	t.Run("explicit file name wins", func(t *testing.T) {
		req := &libebics.OrderContext{OrderID: "ORD-1", OrderType: "BTU"}

		name := deriveIncomingFilename(&ebicsxml.BTUOrderParams{FileName: " incoming/file.xml "}, req)

		require.Equal(t, "incoming/file.xml", name)
	})

	t.Run("order id is used when file name is absent", func(t *testing.T) {
		req := &libebics.OrderContext{OrderID: "ORD-2", OrderType: "BTU"}

		name := deriveIncomingFilename(nil, req)

		require.Equal(t, "ORD-2.xml", name)
	})

	t.Run("order type fallback is deterministic", func(t *testing.T) {
		req := &libebics.OrderContext{OrderType: "FUL"}

		name := deriveIncomingFilename(nil, req)

		require.Equal(t, "ebics-ful.xml", name)
	})
}

func TestResolveRuntimeCorrelationID(t *testing.T) {
	t.Run("explicit correlation id wins", func(t *testing.T) {
		req := &libebics.OrderContext{
			CorrelationID: "corr-1",
			OrderID:       "order-1",
			TransID:       "tx-1",
		}

		require.Equal(t, "corr-1", resolveRuntimeCorrelationID(req))
	})

	t.Run("order id is used as fallback", func(t *testing.T) {
		req := &libebics.OrderContext{
			OrderID: "order-2",
			TransID: "tx-2",
		}

		require.Equal(t, "order-2", resolveRuntimeCorrelationID(req))
	})

	t.Run("transaction id is the last fallback", func(t *testing.T) {
		req := &libebics.OrderContext{TransID: "tx-3"}

		require.Equal(t, "tx-3", resolveRuntimeCorrelationID(req))
	})
}

func TestEnrichTransferInfo(t *testing.T) {
	transfer := &model.Transfer{
		TransferInfo: map[string]any{
			"existing": "value",
		},
	}
	operation := &model.EbicsOperation{
		ID:            42,
		OrderType:     "BTU",
		RequestID:     "REQ-42",
		CorrelationID: "CORR-42",
		EbicsVersion:  "H005",
	}
	resolved := &ebicsruntime.ResolvedPayloadRequest{
		Subscriber: ebicsruntime.PayloadSubscriberRef{
			HostID:    "HOST1",
			PartnerID: "PARTNER1",
			UserID:    "USER1",
		},
		ResolvedService: ebicsruntime.PayloadServiceRef{
			ServiceName:   "MCT",
			ServiceOption: "SDD",
			Scope:         "FR",
			MsgName:       "pain.001",
			ContainerType: "ZIP",
		},
	}

	enrichTransferInfo(transfer, operation, resolved)

	require.Equal(t, "value", transfer.TransferInfo["existing"])
	require.EqualValues(t, 42, transfer.TransferInfo[transferInfoKeyEbicsOperationID])
	require.Equal(t, "BTU", transfer.TransferInfo[transferInfoKeyEbicsOrderType])
	require.Equal(t, "HOST1", transfer.TransferInfo[transferInfoKeyEbicsHostID])
	require.Equal(t, "PARTNER1", transfer.TransferInfo[transferInfoKeyEbicsPartnerID])
	require.Equal(t, "USER1", transfer.TransferInfo[transferInfoKeyEbicsUserID])
	require.Equal(t, "REQ-42", transfer.TransferInfo[transferInfoKeyEbicsRequestID])
	require.Equal(t, "CORR-42", transfer.TransferInfo[transferInfoKeyEbicsCorrelationID])
	require.Equal(t, "H005", transfer.TransferInfo[transferInfoKeyEbicsProtocol])

	serviceInfo, ok := transfer.TransferInfo[transferInfoKeyEbicsService].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "MCT", serviceInfo["serviceName"])
	require.Equal(t, "SDD", serviceInfo["serviceOption"])
	require.Equal(t, "FR", serviceInfo["scope"])
	require.Equal(t, "pain.001", serviceInfo["msgName"])
	require.Equal(t, "ZIP", serviceInfo["containerType"])
}

func TestMappedOrderError(t *testing.T) {
	t.Run("invalid order params keeps technical scope", func(t *testing.T) {
		err := mappedOrderError(liborders.ErrInvalidOrderParams)

		code, ok := libreturncode.FromError(err)
		require.True(t, ok)
		require.Equal(t, libreturncode.CodeInvalidOrderParams, code)
	})

	t.Run("processing error keeps business scope", func(t *testing.T) {
		err := mappedOrderError(liborders.ErrProcessing)

		code, ok := libreturncode.FromError(err)
		require.True(t, ok)
		require.Equal(t, libreturncode.CodeProcessingError, code)
	})

	t.Run("wrapped sentinel is still recognized", func(t *testing.T) {
		err := mappedOrderError(errors.New("unexpected"))

		code, ok := libreturncode.FromError(err)
		require.False(t, ok)
		require.ErrorContains(t, err, "unexpected")
		require.Equal(t, libreturncode.Code{}, code)
	})
}

func TestBuildResolvedRequests(t *testing.T) {
	req := libebics.OrderContext{
		HostID:          " HOST1 ",
		PartnerID:       " PARTNER1 ",
		UserID:          " USER1 ",
		OrderType:       "FUL",
		OrderID:         " ORDER-1 ",
		ProtocolVersion: " h005 ",
		TransID:         "TX-1",
	}

	t.Run("upload request keeps normalized payload metadata", func(t *testing.T) {
		resolved := buildUploadResolvedRequest(&req, &ebicsxml.BTUOrderParams{
			FileName: " incoming.xml ",
			Service: ebicsxml.RestrictedService{
				ServiceName:   " MCT ",
				ServiceOption: " SDD ",
				Scope:         " FR ",
				MsgName: ebicsxml.MessageType{
					Value: " pain.001 ",
				},
			},
		})

		require.Equal(t, "BTU", resolved.OrderType)
		require.Equal(t, "HOST1", resolved.Subscriber.HostID)
		require.Equal(t, "PARTNER1", resolved.Subscriber.PartnerID)
		require.Equal(t, "USER1", resolved.Subscriber.UserID)
		require.NotNil(t, resolved.ResolvedFile)
		require.Equal(t, "incoming.xml", resolved.ResolvedFile.OutputName)
		require.Equal(t, "MCT", resolved.ResolvedService.ServiceName)
		require.Equal(t, "SDD", resolved.ResolvedService.ServiceOption)
		require.Equal(t, "FR", resolved.ResolvedService.Scope)
		require.Equal(t, "pain.001", resolved.ResolvedService.MsgName)
		require.Equal(t, "ORDER-1", resolved.ResolvedMetadata["requestID"])
		require.Equal(t, "ORDER-1", resolved.ResolvedMetadata["correlationID"])
		require.Equal(t, "H005", resolved.ResolvedMetadata["protocol"])
	})

	t.Run("download request keeps normalized service metadata", func(t *testing.T) {
		resolved := buildDownloadResolvedRequest(&req, &ebicsxml.BTDOrderParams{
			Service: ebicsxml.RestrictedService{
				ServiceName: " MCT ",
				Scope:       " FR ",
				MsgName: ebicsxml.MessageType{
					Value: " camt.054 ",
				},
			},
		})

		require.Equal(t, "BTU", resolved.OrderType)
		require.Equal(t, "MCT", resolved.ResolvedService.ServiceName)
		require.Equal(t, "FR", resolved.ResolvedService.Scope)
		require.Equal(t, "camt.054", resolved.ResolvedService.MsgName)
		require.Equal(t, "ORDER-1", resolved.ResolvedMetadata["requestID"])
	})
}

func TestResolveServerAccount(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	router := newPayloadOrderRouter(db, nil)
	agent := insertTestEBICSServer(t, db, 0)
	localAccount := insertTestLocalAccount(t, db, agent.ID, "ebics-local")

	t.Run("linked EBICS account is resolved", func(t *testing.T) {
		subscriber := &model.EbicsSubscriber{
			Name:           "subscriber-ok",
			LocalAccountID: sql.NullInt64{Int64: localAccount.ID, Valid: true},
		}

		resolvedAccount, resolvedAgent, err := router.resolveServerAccount(subscriber)

		require.NoError(t, err)
		require.Equal(t, localAccount.ID, resolvedAccount.ID)
		require.Equal(t, agent.ID, resolvedAgent.ID)
	})

	t.Run("missing link is rejected", func(t *testing.T) {
		_, _, err := router.resolveServerAccount(&model.EbicsSubscriber{Name: "subscriber-missing"})

		require.Error(t, err)
		require.ErrorContains(t, err, "not linked to a Gateway local account")

		code, ok := libreturncode.FromError(err)
		require.True(t, ok)
		require.Equal(t, libreturncode.CodeProcessingError, code)
	})
}

func TestResolveHostAndSubscriber(t *testing.T) {
	db := dbtest.TestDatabase(t)
	router := newPayloadOrderRouter(db, nil)
	host := insertTestEbicsHost(t, db, "HOST1")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER1", "USER1", true)

	t.Run("known enabled subscriber is resolved", func(t *testing.T) {
		resolvedHost, resolvedSubscriber, err := router.resolveHostAndSubscriber(libebics.OrderContext{
			HostID:    "HOST1",
			PartnerID: "PARTNER1",
			UserID:    "USER1",
		})

		require.NoError(t, err)
		require.Equal(t, host.ID, resolvedHost.ID)
		require.Equal(t, subscriber.ID, resolvedSubscriber.ID)
	})

	t.Run("unknown host is rejected", func(t *testing.T) {
		_, _, err := router.resolveHostAndSubscriber(libebics.OrderContext{
			HostID:    "UNKNOWN",
			PartnerID: "PARTNER1",
			UserID:    "USER1",
		})

		require.Error(t, err)
		require.ErrorContains(t, err, "unknown EBICS host")

		code, ok := libreturncode.FromError(err)
		require.True(t, ok)
		require.Equal(t, libreturncode.CodeInvalidOrderParams, code)
	})

	t.Run("disabled subscriber is rejected", func(t *testing.T) {
		disabled := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER2", "USER2", false)

		_, _, err := router.resolveHostAndSubscriber(libebics.OrderContext{
			HostID:    "HOST1",
			PartnerID: libebics.PartnerID(disabled.PartnerID),
			UserID:    libebics.UserID(disabled.UserID),
		})

		require.Error(t, err)
		require.ErrorContains(t, err, "is disabled")

		code, ok := libreturncode.FromError(err)
		require.True(t, ok)
		require.Equal(t, libreturncode.CodeInvalidOrderParams, code)
	})
}

func TestPrepareRouteRejectsDirectionMismatch(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	router := newPayloadOrderRouter(db, nil)
	agent := insertTestEBICSServer(t, db, 0)
	account := insertTestLocalAccount(t, db, agent.ID, "ebics-route")
	host := insertTestEbicsHost(t, db, "HOST1")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER1", "USER1", true)
	subscriber.LocalAccountID = utils.NewNullInt64(account.ID)
	require.NoError(t, db.Update(subscriber).Run())
	insertTestActiveContractItem(t, db, host.ID, subscriber.ID, &model.EbicsContractViewItem{
		ItemType:    "BTF",
		ItemKey:     "btu-mct",
		OrderType:   "BTU",
		ServiceName: "MCT",
		IsEnabled:   true,
	})

	sendRule := insertTestRule(t, db, "send-only", true)
	insertTestPayloadProfile(t, db, &model.EbicsPayloadProfile{
		Name:          "upload-bound-to-send",
		OrderType:     "BTU",
		Direction:     "UPLOAD",
		ServiceName:   "MCT",
		DefaultRuleID: utils.NewNullInt64(sendRule.ID),
		IsEnabled:     true,
	})

	resolved := &ebicsruntime.ResolvedPayloadRequest{
		OrderType: "BTU",
		ResolvedService: ebicsruntime.PayloadServiceRef{
			OrderType:   "BTU",
			ServiceName: "MCT",
		},
		Subscriber: ebicsruntime.PayloadSubscriberRef{
			HostID:    "HOST1",
			PartnerID: "PARTNER1",
			UserID:    "USER1",
		},
	}

	_, _, err := router.prepareRoute(context.Background(), libebics.OrderContext{
		HostID:    "HOST1",
		PartnerID: "PARTNER1",
		UserID:    "USER1",
		OrderType: "BTU",
		OrderID:   "ORDER-1",
	}, resolved)

	require.Error(t, err)
	require.ErrorContains(t, err, "incompatible Gateway rule")

	code, ok := libreturncode.FromError(err)
	require.True(t, ok)
	require.Equal(t, libreturncode.CodeProcessingError, code)
}

func insertTestRule(t *testing.T, db *database.DB, name string, isSend bool) *model.Rule {
	t.Helper()

	rule := &model.Rule{
		Name:           name,
		IsSend:         isSend,
		Path:           name,
		TmpLocalRcvDir: ".",
	}
	require.NoError(t, db.Insert(rule).Run())

	return rule
}

func insertTestPayloadProfile(t *testing.T, db *database.DB, profile *model.EbicsPayloadProfile) *model.EbicsPayloadProfile {
	t.Helper()

	require.NoError(t, db.Insert(profile).Run())

	return profile
}

func insertTestLocalAccount(t *testing.T, db *database.DB, agentID int64, login string) *model.LocalAccount {
	t.Helper()

	account := &model.LocalAccount{
		LocalAgentID: agentID,
		Login:        login,
	}
	require.NoError(t, db.Insert(account).Run())

	return account
}

func insertTestEbicsHost(t *testing.T, db *database.DB, hostID string) *model.EbicsHost {
	t.Helper()

	host := &model.EbicsHost{
		Name:            "host-" + hostID,
		HostID:          hostID,
		Enabled:         true,
		IsServer:        true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())

	return host
}

func insertTestEbicsSubscriber(
	t *testing.T,
	db *database.DB,
	hostPK int64,
	partnerID, userID string,
	enabled bool,
) *model.EbicsSubscriber {
	t.Helper()

	subscriber := &model.EbicsSubscriber{
		Name:        partnerID + "-" + userID,
		EbicsHostID: hostPK,
		PartnerID:   partnerID,
		UserID:      userID,
		Enabled:     enabled,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	return subscriber
}

func insertTestActiveContractItem(
	t *testing.T,
	db *database.DB,
	hostPK, subscriberPK int64,
	item *model.EbicsContractViewItem,
) *model.EbicsContractViewItem {
	t.Helper()

	view := &model.EbicsContractView{
		EbicsHostID:       hostPK,
		EbicsSubscriberID: sql.NullInt64{Int64: subscriberPK, Valid: true},
		SourceOrderType:   "HPD",
		VersionTag:        "v1",
		Status:            "ACTIVE",
	}
	require.NoError(t, db.Insert(view).Run())

	item.ContractViewID = view.ID
	require.NoError(t, db.Insert(item).Run())

	return item
}
